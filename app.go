package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"

	"github.com/taylor-r-miller/Flik/internal/audio"
	"github.com/taylor-r-miller/Flik/internal/display"
	"github.com/taylor-r-miller/Flik/internal/permissions"
	"github.com/taylor-r-miller/Flik/internal/spaces"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// HotkeyStatus represents the current state of hotkey functionality
type HotkeyStatus string

const (
	HotkeyStatusDisabled HotkeyStatus = "disabled"
	HotkeyStatusEnabled  HotkeyStatus = "enabled"
	HotkeyStatusFailed   HotkeyStatus = "failed"
	HotkeyStatusPending  HotkeyStatus = "pending"
)

// AppMode represents the current mode of the application
type AppMode string

const (
	AppModeMain    AppMode = "main"    // W M D - default mode
	AppModeWindow  AppMode = "window"  // H M L - window/space navigation
	AppModeDisplay AppMode = "display" // H M L - display navigation
)

// App struct
type App struct {
	ctx          context.Context
	numberBuffer string
	displayMover *display.Mover
	audioManager *audio.Manager
	spacesManager *spaces.Manager
	hotkeyStatus HotkeyStatus
	currentMode  AppMode
	hotkeyInstance *hotkey.Hotkey
	hotkeyShutdown chan bool
	lastHotkeyTime time.Time
}

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
)

// logf logs a formatted message with the specified level
func (a *App) logf(level LogLevel, format string, args ...interface{}) {
	prefix := ""
	switch level {
	case LogLevelInfo:
		prefix = "INFO"
	case LogLevelWarn:
		prefix = "WARN"
	case LogLevelError:
		prefix = "ERROR"
	}

	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", prefix, message)

	// Show user-visible notifications for warnings and errors
	if level >= LogLevelWarn && a.ctx != nil {
		go func() {
			title := "Flik"
			if level == LogLevelError {
				title = "Flik Error"
			}

			runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.InfoDialog,
				Title:   title,
				Message: message,
			})
		}()
	}
}

func (a *App) setupHotkey() {
	defer func() {
		if r := recover(); r != nil {
			a.logf(LogLevelError, "hotkey setup panicked: %v", r)
			a.hotkeyStatus = HotkeyStatusFailed
		}
	}()

	a.logf(LogLevelInfo, "Starting hotkey setup...")
	a.hotkeyStatus = HotkeyStatusPending

	// Create a context with timeout ONLY for setup phase
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()

	// Check accessibility permissions with timeout
	permissionsChan := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.logf(LogLevelError, "permission check panicked: %v", r)
				permissionsChan <- false
			}
		}()
		a.logf(LogLevelInfo, "Checking accessibility permissions...")
		
		// Add detailed permission checking
		a.logPermissionDetails()
		
		hasPerms := permissions.CheckAccessibilityPermissions()
		a.logf(LogLevelInfo, "Accessibility permissions check result: %v", hasPerms)
		permissionsChan <- hasPerms
	}()

	var hasPermissions bool
	select {
	case hasPermissions = <-permissionsChan:
	case <-setupCtx.Done():
		a.logf(LogLevelError, "permission check timed out, continuing without hotkey support")
		a.hotkeyStatus = HotkeyStatusFailed
		return
	}

	if !hasPermissions {
		a.logf(LogLevelWarn, "accessibility permissions not granted, app will continue without hotkeys")
		a.hotkeyStatus = HotkeyStatusDisabled
		
		// Non-blocking permission request - don't show dialog immediately on startup
		go func() {
			time.Sleep(2 * time.Second) // Wait before showing permission dialog
			defer func() {
				if r := recover(); r != nil {
					a.logf(LogLevelError, "permission request panicked: %v", r)
				}
			}()
			a.logf(LogLevelInfo, "Requesting accessibility permissions...")
			permissions.RequestAccessibilityPermissions()
		}()
		
		return // Don't show error dialog on startup
	}

	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeySpace)

	// Retry hotkey registration with exponential backoff
	maxRetries := 3
	retryDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-setupCtx.Done():
			a.logf(LogLevelError, "hotkey setup timed out during registration, continuing without hotkey support")
			a.hotkeyStatus = HotkeyStatusFailed
			return
		default:
		}

		err := hk.Register()
		if err == nil {
			a.logf(LogLevelInfo, "global hotkey (Ctrl+Space) registered successfully")
			a.hotkeyStatus = HotkeyStatusEnabled
			a.hotkeyInstance = hk // Store the hotkey instance for cleanup
			break
		}

		a.logf(LogLevelWarn, "hotkey registration attempt %d failed: %v", attempt+1, err)

		if attempt == maxRetries-1 {
			a.logf(LogLevelError, "failed to register hotkey after %d attempts, continuing without hotkey support", maxRetries)
			a.hotkeyStatus = HotkeyStatusFailed
			a.showPermissionError("Global hotkey registration failed despite having permissions. This may be a system issue. Please try restarting the app.")
			return
		}

		// Sleep with context cancellation support
		select {
		case <-time.After(retryDelay):
		case <-setupCtx.Done():
			a.logf(LogLevelError, "hotkey setup cancelled during retry delay")
			a.hotkeyStatus = HotkeyStatusFailed
			return
		}
		retryDelay *= 2
	}

	if a.hotkeyStatus != HotkeyStatusEnabled {
		return
	}

	// Setup completed successfully, now start infinite listening
	// Use a separate context that won't timeout for the listening loop
	a.logf(LogLevelInfo, "hotkey setup complete, starting infinite listener...")
	a.startHotkeyListener()
}

// startHotkeyListener runs the hotkey listening loop indefinitely
func (a *App) startHotkeyListener() {
	defer func() {
		if r := recover(); r != nil {
			a.logf(LogLevelError, "hotkey listener panicked: %v, attempting restart", r)
			// Attempt to restart the listener after a delay
			go func() {
				time.Sleep(5 * time.Second)
				a.logf(LogLevelInfo, "attempting to restart hotkey listener...")
				a.startHotkeyListener()
			}()
		}
	}()

	if a.hotkeyInstance == nil {
		a.logf(LogLevelError, "no hotkey instance available, cannot start listener")
		return
	}

	a.logf(LogLevelInfo, "hotkey listener started - listening for Ctrl+Space")
	
	// Infinite loop without timeout - this should run forever
	for {
		select {
		case <-a.hotkeyInstance.Keydown():
			a.logf(LogLevelInfo, "global hotkey activated, showing window")

			// Handle hotkey activation safely with race condition protection
			func() {
				defer func() {
					if r := recover(); r != nil {
						a.logf(LogLevelError, "hotkey activation handler panicked: %v", r)
					}
				}()

				// Debounce hotkey to prevent rapid activations
				now := time.Now()
				if now.Sub(a.lastHotkeyTime) < 200*time.Millisecond {
					a.logf(LogLevelInfo, "hotkey debounced - too soon after last activation")
					return
				}
				a.lastHotkeyTime = now

				// Show window on current screen (always repositions based on mouse location)
				a.showWindowOnCurrentScreen()
				runtime.WindowSetAlwaysOnTop(a.ctx, true)

				// Reset always on top after a brief moment
				go func() {
					time.Sleep(100 * time.Millisecond)
					runtime.WindowSetAlwaysOnTop(a.ctx, false)
				}()
			}()

		case <-a.hotkeyShutdown:
			a.logf(LogLevelInfo, "hotkey listener shutting down gracefully")
			return
		}
	}
}

// shutdownHotkey gracefully shuts down the hotkey listener
func (a *App) shutdownHotkey() {
	if a.hotkeyInstance != nil {
		a.logf(LogLevelInfo, "shutting down hotkey...")
		
		// Signal shutdown
		select {
		case a.hotkeyShutdown <- true:
		default:
		}
		
		// Unregister the hotkey
		a.hotkeyInstance.Unregister()
		a.hotkeyInstance = nil
		a.hotkeyStatus = HotkeyStatusDisabled
		a.logf(LogLevelInfo, "hotkey shutdown complete")
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		displayMover:   display.NewMover(),
		audioManager:   audio.NewManager(),
		spacesManager:  spaces.NewManager(),
		numberBuffer:   "",
		hotkeyStatus:   HotkeyStatusDisabled,
		currentMode:    AppModeMain,
		hotkeyShutdown: make(chan bool, 1),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.logf(LogLevelInfo, "Flik starting up...")
	
	// Log runtime environment for debugging
	a.logEnvironmentInfo()
	
	// Show window on current screen
	a.showWindowOnCurrentScreen()
	a.logf(LogLevelInfo, "Window shown and centered on current screen")

	// Delay hotkey setup to allow app to fully initialize
	go func() {
		time.Sleep(500 * time.Millisecond) // Give app time to start
		a.logf(LogLevelInfo, "Initializing hotkey setup...")
		mainthread.Init(func() { a.setupHotkey() })
	}()
}

// OnSecondInstanceLaunch is called when a second instance of the app is launched
func (a *App) OnSecondInstanceLaunch(secondInstanceData options.SecondInstanceData) {
	a.showWindowOnCurrentScreen()
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	// Listen for app activation events from frontend
	runtime.EventsOn(ctx, "app:activate", func(optionalData ...interface{}) {
		a.showWindowOnCurrentScreen()
	})
}

// onShutdown is called when the app is shutting down
func (a *App) onShutdown(ctx context.Context) {
	a.logf(LogLevelInfo, "App shutting down...")
	a.shutdownHotkey()
}

// logEnvironmentInfo logs runtime environment details for debugging
func (a *App) logEnvironmentInfo() {
	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		a.logf(LogLevelError, "failed to get executable path: %v", err)
		execPath = "unknown"
	}
	
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		a.logf(LogLevelError, "failed to get working directory: %v", err)
		wd = "unknown"
	}
	
	// Check if running from Applications
	isFromApplications := filepath.HasPrefix(execPath, "/Applications/")
	
	a.logf(LogLevelInfo, "=== Environment Info ===")
	a.logf(LogLevelInfo, "Executable: %s", execPath)
	a.logf(LogLevelInfo, "Working Dir: %s", wd)
	a.logf(LogLevelInfo, "From Applications: %v", isFromApplications)
	a.logf(LogLevelInfo, "UID: %d", os.Getuid())
	a.logf(LogLevelInfo, "GID: %d", os.Getgid())
	
	// Check key environment variables
	envVars := []string{"HOME", "USER", "TMPDIR", "PATH"}
	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if len(value) > 100 {
			value = value[:100] + "..."
		}
		a.logf(LogLevelInfo, "%s: %s", envVar, value)
	}
	a.logf(LogLevelInfo, "========================")
}

// logPermissionDetails logs detailed permission status for debugging
func (a *App) logPermissionDetails() {
	execPath, _ := os.Executable()
	a.logf(LogLevelInfo, "=== Permission Details ===")
	a.logf(LogLevelInfo, "Bundle ID would be: com.yourcompany.workefficiency")
	a.logf(LogLevelInfo, "Executable path: %s", execPath)
	
	// Try to determine if we're sandboxed
	containerPath := os.Getenv("APP_SANDBOX_CONTAINER_ID")
	if containerPath != "" {
		a.logf(LogLevelInfo, "Running in sandbox: %s", containerPath)
	} else {
		a.logf(LogLevelInfo, "Not detected as sandboxed")
	}
	
	a.logf(LogLevelInfo, "==========================")
}

// ProcessKeyPress handles key press events from the frontend
func (a *App) ProcessKeyPress(key string) {
	// Check if it's a number (for vim-like repetition)
	if num, err := strconv.Atoi(key); err == nil {
		a.numberBuffer += strconv.Itoa(num)
		return
	}

	// Get repetition count
	count := 1
	if a.numberBuffer != "" {
		if c, err := strconv.Atoi(a.numberBuffer); err == nil && c > 0 {
			count = c
		}
		a.numberBuffer = "" // Reset buffer
	}

	// Handle mode switching and commands based on current mode
	switch a.currentMode {
	case AppModeMain:
		a.processMainModeKey(key, count)
	case AppModeWindow:
		a.processWindowModeKey(key, count)
	case AppModeDisplay:
		a.processDisplayModeKey(key, count)
	}
}

// processMainModeKey handles key presses in main mode (W M D)
func (a *App) processMainModeKey(key string, count int) {
	switch key {
	case "w":
		a.currentMode = AppModeWindow
		a.logf(LogLevelInfo, "switched to window mode")
		return // Don't hide window, let user see the mode change
	case "d":
		a.currentMode = AppModeDisplay
		a.logf(LogLevelInfo, "switched to display mode")
		return // Don't hide window, let user see the mode change
	case "m":
		a.audioManager.ToggleMute()
	case "Escape":
		runtime.WindowHide(a.ctx)
		a.numberBuffer = ""
		return
	default:
		a.logf(LogLevelWarn, "invalid key '%s' in main mode", key)
		return
	}
	runtime.WindowHide(a.ctx)
}

// processWindowModeKey handles key presses in window mode (H M L)
func (a *App) processWindowModeKey(key string, count int) {
	switch key {
	case "h":
		// Move left through virtual desktops/spaces
		for i := 0; i < count; i++ {
			if err := a.spacesManager.MoveToSpace("left"); err != nil {
				a.logf(LogLevelError, "failed to move to left space: %v", err)
				break
			}
		}
		// Return to main mode after space navigation
		a.currentMode = AppModeMain
	case "l":
		// Move right through virtual desktops/spaces
		for i := 0; i < count; i++ {
			if err := a.spacesManager.MoveToSpace("right"); err != nil {
				a.logf(LogLevelError, "failed to move to right space: %v", err)
				break
			}
		}
		// Return to main mode after space navigation
		a.currentMode = AppModeMain
	case "m":
		a.audioManager.ToggleMute()
	case "b", "Escape":
		a.currentMode = AppModeMain
		a.logf(LogLevelInfo, "returned to main mode")
		return // Don't hide window, let user see the mode change
	default:
		a.logf(LogLevelWarn, "invalid key '%s' in window mode", key)
		return
	}
	runtime.WindowHide(a.ctx)
}

// processDisplayModeKey handles key presses in display mode (H M L)
func (a *App) processDisplayModeKey(key string, count int) {
	switch key {
	case "h":
		// Move left through physical displays
		for i := 0; i < count; i++ {
			if err := a.displayMover.MoveToDisplay("left"); err != nil {
				a.logf(LogLevelError, "failed to move to left display: %v", err)
				break
			}
		}
		// Return to main mode after display navigation
		a.currentMode = AppModeMain
	case "l":
		// Move right through physical displays
		for i := 0; i < count; i++ {
			if err := a.displayMover.MoveToDisplay("right"); err != nil {
				a.logf(LogLevelError, "failed to move to right display: %v", err)
				break
			}
		}
		// Return to main mode after display navigation
		a.currentMode = AppModeMain
	case "m":
		a.audioManager.ToggleMute()
	case "b", "Escape":
		a.currentMode = AppModeMain
		a.logf(LogLevelInfo, "returned to main mode")
		return // Don't hide window, let user see the mode change
	default:
		a.logf(LogLevelWarn, "invalid key '%s' in display mode", key)
		return
	}
	runtime.WindowHide(a.ctx)
}

// GetStatus returns the current status for the UI
func (a *App) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"numberBuffer":  a.numberBuffer,
		"isMuted":       a.audioManager.IsMuted(),
		"hotkeyStatus":  string(a.hotkeyStatus),
		"currentMode":   string(a.currentMode),
	}
}

// showPermissionError displays a permission error dialog to the user
func (a *App) showPermissionError(message string) {
	if a.ctx != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.WarningDialog,
			Title:   "Permissions Required",
			Message: message,
		})
	}
}

// createMenuBar creates the application menu bar
func (a *App) createMenuBar() *menu.Menu {
	appMenu := menu.NewMenu()

	// File menu
	fileMenu := appMenu.AddSubmenu("Flik")
	fileMenu.AddText("Show Flik", keys.CmdOrCtrl("space"), func(_ *menu.CallbackData) {
		a.showWindow()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(a.ctx)
	})

	// Help menu
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("Check Permissions", nil, func(_ *menu.CallbackData) {
		if permissions.CheckAccessibilityPermissions() {
			runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
				Type:    runtime.InfoDialog,
				Title:   "Permissions Status",
				Message: "✓ Accessibility permissions are granted.\nGlobal hotkey (Ctrl+Space) should work.",
			})
		} else {
			a.showPermissionError("❌ Accessibility permissions are NOT granted.\n\nTo enable global hotkeys:\n1. Open System Preferences\n2. Go to Security & Privacy > Privacy > Accessibility\n3. Click the lock to make changes\n4. Add Flik and enable it\n5. Restart Flik")
		}
	})

	return appMenu
}

// showWindow shows and centers the application window
func (a *App) showWindow() {
	a.showWindowOnCurrentScreen()
}

// showWindowOnCurrentScreen shows and centers the window on the screen with the mouse cursor
func (a *App) showWindowOnCurrentScreen() {
	if a.ctx == nil {
		return
	}

	// Get all displays to find the current one
	displays, err := a.displayMover.GetConfiguration()
	if err != nil {
		a.logf(LogLevelWarn, "failed to get display configuration, using default centering: %v", err)
		// Fallback to default behavior
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowCenter(a.ctx)
		return
	}

	if len(displays) == 0 {
		a.logf(LogLevelWarn, "no displays found, using default centering")
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowCenter(a.ctx)
		return
	}

	// Get current mouse position to determine which screen we're on
	currentDisplay := a.getCurrentDisplay(displays)
	if currentDisplay == nil {
		a.logf(LogLevelWarn, "could not determine current display, using default centering")
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowCenter(a.ctx)
		return
	}

	// Calculate center position of current display
	x, _ := strconv.ParseFloat(currentDisplay.X, 64)
	y, _ := strconv.ParseFloat(currentDisplay.Y, 64)
	width, _ := strconv.ParseFloat(currentDisplay.Width, 64)
	height, _ := strconv.ParseFloat(currentDisplay.Height, 64)
	
	centerX := int(x + width/2)
	centerY := int(y + height/2)
	windowX := centerX - 225 // 225 = half of 450px width
	windowY := centerY - 72  // 72 = half of 145px height
	
	a.logf(LogLevelInfo, "positioning window on display %s at center (%d, %d)", currentDisplay.ID, centerX, centerY)
	
	// Force a clean state: hide, position, then show
	// This ensures the window always moves to the correct position
	runtime.WindowHide(a.ctx)
	runtime.WindowSetPosition(a.ctx, windowX, windowY)
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

// getCurrentDisplay determines which display the mouse cursor is currently on
func (a *App) getCurrentDisplay(displays []display.Display) *display.Display {
	if len(displays) == 1 {
		return &displays[0]
	}
	
	// Get current mouse position
	mouseX, mouseY := a.displayMover.GetCurrentMousePosition()
	a.logf(LogLevelInfo, "current mouse position: (%.0f, %.0f)", mouseX, mouseY)
	
	// Find which display contains the mouse cursor
	for i := range displays {
		x, _ := strconv.ParseFloat(displays[i].X, 64)
		y, _ := strconv.ParseFloat(displays[i].Y, 64)
		width, _ := strconv.ParseFloat(displays[i].Width, 64)
		height, _ := strconv.ParseFloat(displays[i].Height, 64)
		
		// Check if mouse is within this display's bounds
		if mouseX >= x && mouseX <= x+width && mouseY >= y && mouseY <= y+height {
			a.logf(LogLevelInfo, "mouse is on display %s (%s,%s %sx%s)", displays[i].ID, displays[i].X, displays[i].Y, displays[i].Width, displays[i].Height)
			return &displays[i]
		}
	}
	
	// Fallback to primary display (usually 0,0)
	for i := range displays {
		if displays[i].X == "0" && displays[i].Y == "0" {
			a.logf(LogLevelInfo, "fallback to primary display (0,0)")
			return &displays[i]
		}
	}
	
	// Final fallback to first display
	a.logf(LogLevelInfo, "final fallback to first display")
	return &displays[0]
}

// GetCurrentScreenInfo returns information about the current screen for debugging
func (a *App) GetCurrentScreenInfo() map[string]interface{} {
	displays, err := a.displayMover.GetConfiguration()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	
	mouseX, mouseY := a.displayMover.GetCurrentMousePosition()
	currentDisplay := a.getCurrentDisplay(displays)
	
	return map[string]interface{}{
		"mousePosition": map[string]float64{
			"x": mouseX,
			"y": mouseY,
		},
		"displayCount": len(displays),
		"displays":    displays,
		"currentDisplay": currentDisplay,
	}
}

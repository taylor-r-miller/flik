package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"

	"github.com/taylor-r-miller/Flik/internal/audio"
	"github.com/taylor-r-miller/Flik/internal/display"
	"github.com/taylor-r-miller/Flik/internal/permissions"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	numberBuffer string
	displayMover *display.Mover
	audioManager *audio.Manager
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
	// Check accessibility permissions first
	if !permissions.CheckAccessibilityPermissions() {
		a.logf(LogLevelWarn, "accessibility permissions not granted")
		a.showPermissionError("Flik needs accessibility permissions to register global hotkeys.\n\nPlease:\n1. Open System Preferences\n2. Go to Security & Privacy > Privacy > Accessibility\n3. Click the lock to make changes\n4. Add Flik to the list and enable it\n5. Restart Flik")
		
		// Optionally request permissions (shows system dialog)
		permissions.RequestAccessibilityPermissions()
		return
	}

	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeySpace)
	
	// Retry hotkey registration with exponential backoff
	maxRetries := 3
	retryDelay := time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := hk.Register()
		if err == nil {
			a.logf(LogLevelInfo, "global hotkey (Ctrl+Space) registered successfully")
			break
		}
		
		a.logf(LogLevelWarn, "hotkey registration attempt %d failed: %v", attempt+1, err)
		
		if attempt == maxRetries-1 {
			a.logf(LogLevelError, "failed to register hotkey after %d attempts, continuing without hotkey support", maxRetries)
			a.showPermissionError("Global hotkey registration failed despite having permissions. This may be a system issue. Please try restarting the app.")
			return
		}
		
		time.Sleep(retryDelay)
		retryDelay *= 2
	}

	// Hotkey registration successful, start listening
	a.logf(LogLevelInfo, "listening for global hotkey events")
	for {
		<-hk.Keydown()
		a.logf(LogLevelInfo, "global hotkey activated, showing window")

		// Foreground the application window
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.WindowCenter(a.ctx)

		// Reset always on top after a brief moment
		go func() {
			time.Sleep(100 * time.Millisecond)
			runtime.WindowSetAlwaysOnTop(a.ctx, false)
		}()
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		displayMover: display.NewMover(),
		audioManager: audio.NewManager(),
		numberBuffer: "",
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Show window immediately when app starts
	runtime.WindowShow(ctx)
	runtime.WindowCenter(ctx)
	
	
	go mainthread.Init(func() { a.setupHotkey() })
}

// OnSecondInstanceLaunch is called when a second instance of the app is launched
func (a *App) OnSecondInstanceLaunch(secondInstanceData options.SecondInstanceData) {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowCenter(a.ctx)
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	// Listen for app activation events from frontend
	runtime.EventsOn(ctx, "app:activate", func(optionalData ...interface{}) {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowCenter(a.ctx)
	})
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

	// Process the actual command
	switch key {
	case "h": // Move left
		for i := 0; i < count; i++ {
			a.displayMover.MoveToDisplay("left")
		}
	case "l": // Move right
		for i := 0; i < count; i++ {
			a.displayMover.MoveToDisplay("right")
		}
	case "m": // Toggle mute
		a.audioManager.ToggleMute()
	case "Escape":
		// Quit the application
		runtime.Quit(a.ctx)
		a.numberBuffer = "" // Reset on escape
	}
	runtime.WindowHide(a.ctx)
}

// GetStatus returns the current status for the UI
func (a *App) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"numberBuffer": a.numberBuffer,
		"isMuted":      a.audioManager.IsMuted(),
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
	if a.ctx != nil {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.WindowCenter(a.ctx)
		
		// Reset always on top after a brief moment
		go func() {
			time.Sleep(100 * time.Millisecond)
			runtime.WindowSetAlwaysOnTop(a.ctx, false)
		}()
	}
}

package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"

	"github.com/taylor-r-miller/Flik/internal/audio"
	"github.com/taylor-r-miller/Flik/internal/display"
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

func (a *App) setupHotkey() {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl}, hotkey.KeySpace)
	err := hk.Register()
	if err != nil {
		log.Fatalf("hotkey: failed to register hotkey: %v", err)
		return
	}

	log.Printf("hotkey: %v is registered\n", hk)

	for {
		<-hk.Keydown()
		log.Printf("hotkey: %v is down\n", hk)

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

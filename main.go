package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	log.Println("Flik starting...")
	
	// Create an instance of the app structure
	app := NewApp()
	log.Println("App instance created")

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received shutdown signal, cleaning up...")
		app.shutdownHotkey()
		os.Exit(0)
	}()

	// Create application with options
	log.Println("Starting Wails application...")
	err := wails.Run(&options.App{
		Title:  "Flik",
		Width:  450,
		Height: 145,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		Frameless:        true,
		AlwaysOnTop:      false,
		DisableResize:    true,
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.onShutdown,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "flik-unique-id",
			OnSecondInstanceLaunch: app.OnSecondInstanceLaunch,
		},
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               true,
				FullSizeContent:            true,
			},
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Menu:        app.createMenuBar(),
		StartHidden: false,
	})

	if err != nil {
		log.Printf("Flik failed to start: %v", err)
		log.Fatal(err)
	}
	
	log.Println("Flik exited normally")
}

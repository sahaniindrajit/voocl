package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "Voocl",
		Description: "Open-source BYOK voice dictation",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Placeholder window until the tray icon exists (see ROADMAP.md, M1).
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Voocl",
		Width:  480,
		Height: 320,
		URL:    "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

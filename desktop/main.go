package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:             "PDF DB",
		Width:             320,
		Height:            520,
		MinWidth:          320,
		MinHeight:         520,
		MaxWidth:          320,
		MaxHeight:         520,
		DisableResize:     true,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "pdfdb-pocket-desktop",
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				FullSizeContent:            true,
				HideToolbarSeparator:       true,
			},
			Appearance:          mac.NSAppearanceNameDarkAqua,
			WindowIsTranslucent: true,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

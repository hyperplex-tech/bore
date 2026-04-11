//go:build desktop

package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/hyperplex-tech/bore/desktop"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := desktop.NewApp()

	err := wails.Run(&options.App{
		Title:     "Bore \u2014 SSH Tunnel Manager",
		Width:     960,
		Height:    640,
		MinWidth:  800,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: desktop.Assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 46, A: 255},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to start desktop app")
	}
}

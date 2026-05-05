package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "wtw-sticker-picker",
		Description: "Sticker picker popup utility",
		Services: []application.Service{
			application.NewService(&GreetService{}),
			application.NewService(&GlobalHotkeyService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            popupWindowName,
		Title:           "Sticker Picker",
		Width:           popupWidth,
		Height:          popupHeight,
		Hidden:          true,
		DisableResize:   true,
		Frameless:       true,
		AlwaysOnTop:     true,
		HideOnEscape:    true,
		HideOnFocusLost: true,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHidden,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            settingsWindowName,
		Title:           "Settings",
		Width:           settingsWidth,
		Height:          settingsHeight,
		Hidden:          true,
		DisableResize:   true,
		Frameless:       true,
		AlwaysOnTop:     true,
		HideOnEscape:    true,
		HideOnFocusLost: false,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHidden,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/?view=settings",
	})

	tray := app.SystemTray.New()
	tray.SetLabel("WTW")
	tray.SetTooltip("wtw-sticker-picker")
	trayMenu := app.NewMenu()
	trayMenu.Add("Settings").OnClick(func(_ *application.Context) {
		showWindowCentered(app, settingsWindowName, settingsWidth, settingsHeight)
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)
	tray.OnRightClick(func() {
		tray.OpenMenu()
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

package main

import "github.com/wailsapp/wails/v3/pkg/application"

type GreetService struct{}

func (g *GreetService) Greet(name string) string {
	return "Hello " + name + "!"
}

func (g *GreetService) HidePopup() {
	window, ok := application.Get().Window.GetByName(popupWindowName)
	if !ok {
		return
	}
	window.Hide()
}

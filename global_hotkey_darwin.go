//go:build darwin

package main

import "golang.design/x/hotkey"

const GlobalShortcutDescription = "Ctrl + Option + S"

func globalHotkeyModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModCtrl, hotkey.ModOption}
}

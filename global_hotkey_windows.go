//go:build windows

package main

import "golang.design/x/hotkey"

const GlobalShortcutDescription = "Win + Alt + S"

func globalHotkeyModifiers() []hotkey.Modifier {
	return []hotkey.Modifier{hotkey.ModWin, hotkey.ModAlt}
}

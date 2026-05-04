//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

type point struct {
	X int32
	Y int32
}

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procGetCursorPos = user32.NewProc("GetCursorPos")
)

func currentCursorPosition() (int, int, bool) {
	var p point
	ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(p.X), int(p.Y), true
}

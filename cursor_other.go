//go:build !darwin && !windows

package main

func currentCursorPosition() (int, int, bool) {
	return 0, 0, false
}

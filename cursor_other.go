//go:build !darwin

package main

func currentCursorPosition() (int, int, bool) {
	return 0, 0, false
}

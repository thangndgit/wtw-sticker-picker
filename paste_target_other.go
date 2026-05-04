//go:build !darwin && !windows

package main

import "fmt"

func capturePasteTarget() {}

func pasteIntoCapturedTarget() error {
	return fmt.Errorf("paste target is not supported on this platform")
}

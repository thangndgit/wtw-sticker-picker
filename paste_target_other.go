//go:build !darwin && !windows

package main

import "fmt"

func capturePasteTarget() {}

func pasteIntoCapturedTarget() error {
	return fmt.Errorf("paste target is not supported on this platform")
}

func pasteRawStickerIntoCapturedTarget(ext string, raw []byte) error {
	return fmt.Errorf("raw sticker paste target is not supported on this platform")
}

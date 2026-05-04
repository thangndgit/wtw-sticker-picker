//go:build !windows

package main

import (
	"sync"

	"golang.design/x/clipboard"
)

var (
	clipboardInitOnce sync.Once
	clipboardInitErr  error
)

func writeStickerImageToClipboard(pngBytes []byte) error {
	clipboardInitOnce.Do(func() {
		clipboardInitErr = clipboard.Init()
	})
	if clipboardInitErr != nil {
		return clipboardInitErr
	}
	clipboard.Write(clipboard.FmtImage, pngBytes)
	return nil
}

//go:build darwin

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var lastFrontmostBundleID string

func capturePasteTarget() {
	out, err := exec.Command(
		"osascript",
		"-e", `tell application "System Events" to get bundle identifier of first application process whose frontmost is true`,
	).Output()
	if err != nil {
		return
	}
	lastFrontmostBundleID = strings.TrimSpace(string(out))
}

func pasteIntoCapturedTarget() error {
	if lastFrontmostBundleID == "" {
		return runAppleScript(`tell application "System Events" to keystroke "v" using command down`)
	}
	activate := fmt.Sprintf(`tell application id "%s" to activate`, escapeAppleScriptString(lastFrontmostBundleID))
	if err := runAppleScript(activate); err != nil {
		return err
	}
	time.Sleep(70 * time.Millisecond)
	return runAppleScript(`tell application "System Events" to keystroke "v" using command down`)
}

func pasteRawStickerIntoCapturedTarget(ext string, raw []byte) error {
	file, err := os.CreateTemp("", "wtw-sticker-*"+ext)
	if err != nil {
		return fmt.Errorf("create temp sticker file: %w", err)
	}
	tempPath := file.Name()
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("write temp sticker file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close temp sticker file: %w", err)
	}
	quotedPath := escapeAppleScriptString(filepath.Clean(tempPath))
	if err := runAppleScript(fmt.Sprintf(`set the clipboard to (POSIX file "%s")`, quotedPath)); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("copy sticker file to clipboard: %w", err)
	}
	if err := pasteIntoCapturedTarget(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	go func(path string) {
		time.Sleep(20 * time.Second)
		_ = os.Remove(path)
	}(tempPath)
	return nil
}

func runAppleScript(lines ...string) error {
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("osascript timeout")
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}

func escapeAppleScriptString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

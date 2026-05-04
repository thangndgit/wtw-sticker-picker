//go:build darwin

package main

import (
	"bytes"
	"fmt"
	"os/exec"
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

func runAppleScript(lines ...string) error {
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}
	cmd := exec.Command("osascript", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
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

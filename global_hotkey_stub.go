//go:build !darwin && !windows

package main

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// GlobalHotkeyService is only wired on macOS and Windows.
type GlobalHotkeyService struct{}

func (s *GlobalHotkeyService) ServiceStartup(context.Context, application.ServiceOptions) error {
	return nil
}

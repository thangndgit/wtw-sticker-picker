//go:build darwin || windows

package main

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.design/x/hotkey"
)

// GlobalShortcutDescription matches the chord below (Ctrl+Alt+S on Windows).
const GlobalShortcutDescription = "Ctrl + Option + S"

// GlobalHotkeyService registers a system-wide hotkey for the picker popup.
type GlobalHotkeyService struct {
	done chan struct{}
}

func (s *GlobalHotkeyService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	// Registering too early on macOS can crash before the app event loop fully starts.
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModOption}, hotkey.KeyS)

	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		app := application.Get()
		if app == nil {
			return
		}

		started := make(chan struct{}, 1)
		cancel := app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
			select {
			case started <- struct{}{}:
			default:
			}
		})
		defer cancel()

		select {
		case <-ctx.Done():
			return
		case <-started:
		case <-time.After(1200 * time.Millisecond):
			// Fallback in case the event is emitted before listener registration.
		}

		if err := hk.Register(); err != nil {
			return
		}
		defer func() {
			_ = hk.Unregister()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-hk.Keydown():
				if !ok {
					return
				}
				showPopupNearCursor(application.Get(), popupWindowName)
				select {
				case <-ctx.Done():
					return
				case _, ok = <-hk.Keyup():
					if !ok {
						return
					}
				}
			}
		}
	}()
	return nil
}

func (s *GlobalHotkeyService) ServiceShutdown() error {
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
		}
	}
	return nil
}

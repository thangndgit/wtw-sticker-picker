//go:build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const runRegistryValueName = "wtw-sticker-picker"

func isLaunchOnStartupEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open run registry key: %w", err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue(runRegistryValueName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read run registry value: %w", err)
	}
	execPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("resolve executable path: %w", err)
	}
	return value == `"`+execPath+`"`, nil
}

func setLaunchOnStartup(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run registry key: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(runRegistryValueName); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("delete run registry value: %w", err)
		}
		return nil
	}
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := key.SetStringValue(runRegistryValueName, `"`+execPath+`"`); err != nil {
		return fmt.Errorf("set run registry value: %w", err)
	}
	return nil
}

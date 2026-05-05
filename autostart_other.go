//go:build !darwin && !windows

package main

func isLaunchOnStartupEnabled() (bool, error) {
	return false, nil
}

func setLaunchOnStartup(enabled bool) error {
	_ = enabled
	return nil
}

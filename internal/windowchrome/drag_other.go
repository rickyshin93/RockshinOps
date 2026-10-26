//go:build !darwin

package windowchrome

// Other platforms use MyGo's CSS drag regions.
func InstallTopDrag(handle uintptr) {}

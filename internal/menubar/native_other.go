//go:build !darwin

package menubar

import "fmt"

func Native(action string, payload any) (State, error) {
	state := State{Items: []Item{}}
	if action == "state" || action == "stop" {
		return state, nil
	}
	return state, fmt.Errorf("menu bar management is only available on macOS")
}

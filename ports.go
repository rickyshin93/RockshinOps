package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

func validateFavoritePorts(ports []int) error {
	seen := make(map[int]bool)
	for _, port := range ports {
		if port < 1 || port > 65535 || seen[port] {
			return fmt.Errorf("invalid or duplicate favorite port: %d", port)
		}
		seen[port] = true
	}
	return nil
}

// An absent file means no favorites, including on upgrades.
func (s *System) FavoritePorts() ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(s.preferencesPath), "ports.json"))
	if errors.Is(err, os.ErrNotExist) {
		return []int{}, nil
	}
	if err != nil {
		return nil, err
	}
	var ports []int
	if err = json.Unmarshal(data, &ports); err != nil {
		return nil, err
	}
	if ports == nil {
		return nil, fmt.Errorf("favorite ports must be an array")
	}
	return ports, validateFavoritePorts(ports)
}

func (s *System) SaveFavoritePorts(ports []int) ([]int, error) {
	if err := validateFavoritePorts(ports); err != nil {
		return nil, err
	}
	ports = append([]int{}, ports...)
	slices.Sort(ports)
	data, err := json.Marshal(ports)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(filepath.Dir(s.preferencesPath), "ports.json")
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if err = os.WriteFile(path+".tmp", append(data, '\n'), 0600); err != nil {
		return nil, err
	}
	if err = os.Rename(path+".tmp", path); err != nil {
		return nil, err
	}
	return ports, nil
}

func (s *System) CopyPortAddress(address string) error {
	return mygo.Clipboard.Write(transfer.TextData(address))
}

//go:build !darwin

package main

import (
	"fmt"
	"net/url"
)

func systemUpdateProxy(_ *url.URL) (*url.URL, error) {
	return nil, fmt.Errorf("system proxy for updates is currently supported on macOS only")
}

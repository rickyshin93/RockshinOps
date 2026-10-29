//go:build !darwin

package appicon

import "errors"

func render(string, int) ([]byte, error) {
	return nil, errors.New("app icons are not implemented on this platform")
}

func renderComputer() ([]byte, error) {
	return nil, errors.New("computer icons are not implemented on this platform")
}

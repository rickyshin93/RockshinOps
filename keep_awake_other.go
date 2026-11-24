//go:build !darwin

package main

import (
	"errors"
	"time"
)

func takeIdleSleepAssertion(time.Duration, bool) (uint32, error) { return 0, errors.ErrUnsupported }
func releaseIdleSleepAssertion(uint32) error                     { return errors.ErrUnsupported }

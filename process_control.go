package main

import (
	"context"
	"time"

	"rockshinops/internal/metrics"
)

func (s *System) TerminateProcess(pid int, identity string, force bool) error {
	return metrics.TerminateProcess(pid, identity, force)
}

func (s *System) TerminatePortProcess(ctx context.Context, pid int, identity string, port int) error {
	return metrics.TerminatePortProcess(ctx, time.Duration(s.config.ConnectionTimeoutSeconds)*time.Second, pid, identity, port)
}

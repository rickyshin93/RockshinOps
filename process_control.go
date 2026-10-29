package main

import "localops/internal/metrics"

func (s *System) TerminateProcess(pid int, identity string, force bool) error {
	return metrics.TerminateProcess(pid, identity, force)
}

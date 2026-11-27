package main

import (
	"slices"

	"rockshinops/internal/metrics"
)

type trayFrame struct {
	sequence    uint64
	preferences Preferences
	snapshot    metrics.Snapshot
}

func (s *System) trayFrameLocked(snapshot metrics.Snapshot) (trayFrame, func(trayFrame)) {
	s.traySequence++
	prefs := s.preferences
	prefs.MenuBarItems = slices.Clone(prefs.MenuBarItems)
	return trayFrame{sequence: s.traySequence, preferences: prefs, snapshot: snapshot}, s.publishTray
}

// The dispatcher owns presentation order. Producers can release their locks
// before dispatching, even if one is delayed and reaches the UI out of order.
func newTrayPublisher(dispatch func(func()), paint func(trayFrame)) func(trayFrame) {
	var presented uint64
	return func(frame trayFrame) {
		dispatch(func() {
			if frame.sequence > presented {
				presented = frame.sequence
				paint(frame)
			}
		})
	}
}

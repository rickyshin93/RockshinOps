package metrics

import (
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Thermal is the OS thermal pressure level: nominal, moderate, heavy, trapping or sleeping.
type Thermal struct {
	Pressure string `json:"pressure"`
	Source   string `json:"source"`
	Error    string `json:"error"`
}

// PowerAssertion is one active request by a process to keep the system or display awake.
type PowerAssertion struct {
	PID     int        `json:"pid"`
	Process string     `json:"process"`
	App     string     `json:"app"`
	Type    string     `json:"type"`
	Name    string     `json:"name"`
	Details string     `json:"details"`
	Since   *time.Time `json:"since"`
}

type PowerAssertions struct {
	Items     []PowerAssertion `json:"items"`
	Source    string           `json:"source"`
	UpdatedAt time.Time        `json:"updatedAt"`
	Error     string           `json:"error"`
}

// rawAssertion holds the fields read from one IOPMCopyAssertionsByProcess entry.
type rawAssertion struct {
	pid, level                               int
	assertType, name, details, process, path string
	start                                    float64 // CFAbsoluteTime
	startValid                               bool
}

// cfAbsoluteEpoch is 2001-01-01T00:00:00Z, the CFAbsoluteTime reference date, in Unix seconds.
const cfAbsoluteEpoch = 978307200

// thermalPressure maps OSThermalPressureLevel (macOS values 0-4) to a status code.
func thermalPressure(state uint64) (string, bool) {
	levels := []string{"nominal", "moderate", "heavy", "trapping", "sleeping"}
	if state >= uint64(len(levels)) {
		return "", false
	}
	return levels[state], true
}

// powerAssertions keeps held (non-zero level) assertions, oldest first.
func powerAssertions(raw []rawAssertion) []PowerAssertion {
	result := []PowerAssertion{}
	for _, r := range raw {
		if r.level == 0 {
			continue
		}
		a := PowerAssertion{PID: r.pid, Process: r.process, Type: r.assertType, Name: r.name, Details: r.details}
		if a.Process == "" && r.path != "" {
			a.Process = filepath.Base(r.path)
		}
		if index := strings.Index(r.path, ".app/"); index >= 0 {
			a.App = r.path[:index+4]
		}
		if r.startValid {
			since := time.Unix(cfAbsoluteEpoch, 0).Add(time.Duration(r.start * float64(time.Second))).UTC()
			a.Since = &since
		}
		result = append(result, a)
	}
	slices.SortStableFunc(result, func(a, b PowerAssertion) int {
		switch {
		case a.Since == nil && b.Since == nil:
			return a.PID - b.PID
		case a.Since == nil:
			return 1
		case b.Since == nil:
			return -1
		}
		return a.Since.Compare(*b.Since)
	})
	return result
}

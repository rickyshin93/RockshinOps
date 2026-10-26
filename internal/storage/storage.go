package storage

import (
	"fmt"
	"sync"
	"time"

	"localops/internal/localization"
)

// Volume reports the filesystem containing the user's home directory.
// Available is writable capacity, excluding space the OS may later reclaim.
type Volume struct {
	Name      string `json:"name"`
	Mount     string `json:"mount"`
	Format    string `json:"format"`
	Total     uint64 `json:"total"`
	Available uint64 `json:"available"`
}

type Snapshot struct {
	Volume    Volume    `json:"volume"`
	UpdatedAt time.Time `json:"updatedAt"`
	Error     string    `json:"error"`
}

// Monitor serializes native reads and retains only the current result.
type Monitor struct {
	mu      sync.Mutex
	read    func() (Volume, error)
	current Snapshot
}

func NewMonitor(read func() (Volume, error)) *Monitor {
	return &Monitor{read: read, current: Snapshot{Error: localization.T("正在读取磁盘容量")}}
}

func (m *Monitor) Current() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

func (m *Monitor) Refresh() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, err := m.read()
	s := Snapshot{UpdatedAt: time.Now().UTC()}
	if err != nil {
		s.Error = err.Error()
	} else {
		s.Volume = v
	}
	m.current = s
	return s
}

// FormatCapacity uses decimal units, matching the detail window.
func FormatCapacity(bytes uint64) string {
	if bytes >= 1_000_000_000_000 {
		return fmt.Sprintf("%.2f TB", float64(bytes)/1_000_000_000_000)
	}
	if bytes > 0 && bytes < 1_000_000_000 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1_000_000)
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/1_000_000_000)
}

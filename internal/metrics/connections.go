package metrics

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"rockshinops/internal/localization"
)

type ConnectionMetrics struct {
	Received *uint64  `json:"received"`
	Sent     *uint64  `json:"sent"`
	Download *float64 `json:"download"`
	Upload   *float64 `json:"upload"`
	RTT      *float64 `json:"rtt"`
}

type Connection struct {
	Metrics         *ConnectionMetrics `json:"metrics,omitempty"`
	PID             int                `json:"pid"`
	Process         string             `json:"process"`
	ProcessIdentity string             `json:"processIdentity,omitempty"`
	UserID          string             `json:"userId"`
	Protocol        string             `json:"protocol"`
	Family          string             `json:"family"`
	Local           string             `json:"local"`
	Remote          string             `json:"remote"`
	State           string             `json:"state"`
}

type Connections struct {
	Items     []Connection `json:"items"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Warning   string       `json:"warning"`
}

func ReadConnections(ctx context.Context, timeout, interval time.Duration) (Connections, error) {
	return readConnections(ctx, timeout, interval)
}

// BindListeners uses process identities sampled before the socket enumeration.
// The close action rechecks both the socket and the identity against the OS.
func (c Connections) BindListeners(before ProcessList) {
	if before.Error != "" {
		return
	}
	identities := make(map[int]string, len(before.Items))
	for _, process := range before.Items {
		if process.Identity != "0" {
			identities[process.PID] = process.Identity
		}
	}
	for i := range c.Items {
		if c.Items[i].Protocol == "TCP" && c.Items[i].State == "LISTEN" {
			c.Items[i].ProcessIdentity = identities[c.Items[i].PID]
		}
	}
}

func parseConnections(data []byte) ([]Connection, error) {
	items := []Connection{}
	var process, file Connection
	hasFile := false
	flush := func() {
		if hasFile && file.Protocol != "" && file.Local != "" {
			items = append(items, file)
		}
		hasFile = false
	}
	for field := range bytes.SplitSeq(data, []byte{0}) {
		field = bytes.TrimLeft(field, "\n")
		if len(field) < 1 {
			continue
		}
		value := string(field[1:])
		switch field[0] {
		case 'p':
			flush()
			pid, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf(localization.T("lsof 返回了无效 PID：%q"), value)
			}
			process = Connection{PID: pid}
		case 'c':
			process.Process = value
		case 'u':
			process.UserID = value
		case 'f':
			flush()
			file = process
			hasFile = true
		case 't':
			file.Family = value
		case 'P':
			file.Protocol = value
		case 'n':
			file.Local, file.Remote, _ = strings.Cut(value, "->")
		case 'T':
			if state, found := strings.CutPrefix(value, "ST="); found {
				file.State = state
			}
		}
	}
	flush()
	return items, nil
}

package metrics

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type connectionKey struct {
	pid                     int
	protocol, local, remote string
}

type connectionSample struct {
	metrics ConnectionMetrics
	at      time.Time
}

// lsof exposes KAME's embedded IPv6 scope, while nettop uses a zone name.
// Drop the scope for matching; duplicate tuples are excluded rather than guessed.
func connectionEndpoint(endpoint string) string {
	host, port, found := strings.Cut(endpoint, "]:")
	if found {
		host = strings.TrimPrefix(host, "[")
	} else {
		i := strings.LastIndex(endpoint, ":")
		if i < 0 {
			return endpoint
		}
		host, port = endpoint[:i], endpoint[i+1:]
	}
	if host == "*" && port == "*" {
		return ""
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.WithZone("")
		if addr.Is6() && addr.IsLinkLocalUnicast() {
			bytes := addr.As16()
			bytes[2], bytes[3] = 0, 0
			addr = netip.AddrFrom16(bytes)
		}
		host = addr.String()
	}
	return net.JoinHostPort(host, port)
}

func keyForConnection(c Connection) connectionKey {
	return connectionKey{c.PID, c.Protocol, connectionEndpoint(c.Local), connectionEndpoint(c.Remote)}
}

func parseConnectionMetrics(reader io.Reader) (map[connectionKey]ConnectionMetrics, error) {
	r := csv.NewReader(reader)
	r.FieldsPerRecord = -1
	frames := []map[connectionKey]connectionSample{}
	duplicates := map[connectionKey]bool{}
	columns := map[string]int{}
	pid := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if row[0] == "time" {
			columns = map[string]int{}
			for i, name := range row {
				if name != "" {
					columns[name] = i
				}
			}
			for _, name := range []string{"time", "bytes_in", "bytes_out", "rtt_avg"} {
				if _, ok := columns[name]; !ok {
					return nil, fmt.Errorf("nettop: missing column %s", name)
				}
			}
			if len(row) < 2 || row[1] != "" {
				return nil, fmt.Errorf("nettop: missing socket column")
			}
			frames = append(frames, map[connectionKey]connectionSample{})
			duplicates = map[connectionKey]bool{}
			pid = 0
			continue
		}
		if len(frames) == 0 || len(row) < 2 {
			return nil, fmt.Errorf("nettop: missing header or socket")
		}
		label := row[1]
		kind, endpoints, socket := strings.Cut(label, " ")
		if !socket || (kind != "tcp4" && kind != "tcp6" && kind != "udp4" && kind != "udp6") {
			i := strings.LastIndex(label, ".")
			if i < 0 {
				return nil, fmt.Errorf("nettop: invalid process %q", label)
			}
			pid, err = strconv.Atoi(label[i+1:])
			if err != nil {
				return nil, fmt.Errorf("nettop: invalid PID: %w", err)
			}
			continue // Never apply per-process totals to individual sockets.
		}
		if pid <= 0 {
			return nil, fmt.Errorf("nettop: socket without process")
		}
		local, remote, ok := strings.Cut(endpoints, "<->")
		if !ok {
			return nil, fmt.Errorf("nettop: invalid endpoints %q", endpoints)
		}
		if strings.HasSuffix(kind, "6") {
			toEndpoint := func(value string) (string, error) {
				i := strings.LastIndex(value, ".")
				if i < 0 {
					return "", fmt.Errorf("nettop: invalid IPv6 endpoint %q", value)
				}
				return net.JoinHostPort(value[:i], value[i+1:]), nil
			}
			if local, err = toEndpoint(local); err != nil {
				return nil, err
			}
			if remote, err = toEndpoint(remote); err != nil {
				return nil, err
			}
		}
		key := connectionKey{pid, strings.ToUpper(kind[:3]), connectionEndpoint(local), connectionEndpoint(remote)}
		frame := frames[len(frames)-1]
		if duplicates[key] {
			continue
		}
		if _, exists := frame[key]; exists {
			delete(frame, key)
			duplicates[key] = true
			continue
		}
		value := func(name string) (string, error) {
			i := columns[name]
			if i >= len(row) {
				return "", fmt.Errorf("nettop: truncated row")
			}
			return strings.TrimSpace(row[i]), nil
		}
		sample := connectionSample{}
		stamp, err := value("time")
		if err != nil {
			return nil, err
		}
		if sample.at, err = time.Parse("15:04:05.999999", stamp); err != nil {
			return nil, err
		}
		for name, target := range map[string]**uint64{"bytes_in": &sample.metrics.Received, "bytes_out": &sample.metrics.Sent} {
			text, err := value(name)
			if err != nil {
				return nil, err
			}
			if text == "" {
				continue
			}
			n, err := strconv.ParseUint(text, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("nettop: invalid %s: %w", name, err)
			}
			*target = &n
		}
		text, err := value("rtt_avg")
		if err != nil {
			return nil, err
		}
		if text != "" {
			number, unitOK := strings.CutSuffix(text, " ms")
			// nettop truncates the RTT field to nine characters, even in CSV mode.
			if !unitOK && len(text) == 9 {
				number, unitOK = strings.CutSuffix(text, " m")
			}
			ms, err := strconv.ParseFloat(number, 64)
			if !unitOK || err != nil || math.IsNaN(ms) || math.IsInf(ms, 0) || ms < 0 {
				return nil, fmt.Errorf("nettop: invalid RTT %q", text)
			}
			// A zero RTT can mean no RTT sample in this interval.
			if ms > 0 && key.protocol == "TCP" && key.remote != "" {
				sample.metrics.RTT = &ms
			}
		}
		frame[key] = sample
	}
	if len(frames) != 2 {
		return nil, fmt.Errorf("nettop: expected two snapshots, got %d", len(frames))
	}
	result := map[connectionKey]ConnectionMetrics{}
	for key, current := range frames[1] {
		metrics := current.metrics
		if previous, ok := frames[0][key]; ok {
			elapsed := current.at.Sub(previous.at)
			if elapsed < 0 {
				elapsed += 24 * time.Hour
			}
			before := previous.metrics
			if elapsed > 0 && before.Received != nil && before.Sent != nil && metrics.Received != nil && metrics.Sent != nil && *metrics.Received >= *before.Received && *metrics.Sent >= *before.Sent {
				download := float64(*metrics.Received-*before.Received) / elapsed.Seconds()
				upload := float64(*metrics.Sent-*before.Sent) / elapsed.Seconds()
				metrics.Download, metrics.Upload = &download, &upload
			}
		}
		result[key] = metrics
	}
	return result, nil
}

func attachConnectionMetrics(items []Connection, samples map[connectionKey]ConnectionMetrics) {
	counts := map[connectionKey]int{}
	for _, item := range items {
		counts[keyForConnection(item)]++
	}
	for i := range items {
		key := keyForConnection(items[i])
		if metrics, ok := samples[key]; ok && counts[key] == 1 {
			if items[i].State != "ESTABLISHED" {
				metrics.RTT = nil
			}
			items[i].Metrics = &metrics
		}
	}
}

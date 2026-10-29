//go:build linux || windows

package metrics

import (
	"context"
	"net"
	"strconv"
	"time"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

func readConnections(ctx context.Context, timeout, interval time.Duration) (Connections, error) {
	r := Connections{Items: []Connection{}, UpdatedAt: time.Now().UTC()}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	rows, err := psnet.ConnectionsWithContext(ctx, "inet")
	if err != nil {
		return r, err
	}
	names := map[int32]string{}
	for _, row := range rows {
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		name, found := names[row.Pid]
		if !found {
			if p, err := process.NewProcessWithContext(ctx, row.Pid); err == nil {
				name, _ = p.NameWithContext(ctx)
			}
			names[row.Pid] = name
		}
		c := Connection{PID: int(row.Pid), Process: name, Protocol: "TCP", Family: "IPv4", State: row.Status, Local: net.JoinHostPort(row.Laddr.IP, strconv.Itoa(int(row.Laddr.Port)))}
		if row.Type == 2 {
			c.Protocol = "UDP"
		}
		if row.Family != 2 {
			c.Family = "IPv6"
		}
		if row.Raddr.Port != 0 {
			c.Remote = net.JoinHostPort(row.Raddr.IP, strconv.Itoa(int(row.Raddr.Port)))
		}
		if len(row.Uids) > 0 {
			c.UserID = strconv.Itoa(int(row.Uids[0]))
		}
		r.Items = append(r.Items, c)
	}
	return r, nil
}

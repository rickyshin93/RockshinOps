package metrics

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"rockshinops/internal/localization"
	"strconv"
	"strings"
	"sync"
	"time"
)

func readConnections(ctx context.Context, timeout, interval time.Duration) (Connections, error) {
	result := Connections{Items: []Connection{}, UpdatedAt: time.Now().UTC()}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	// nettop's two-sample window can run while lsof enumerates the sockets.
	// Keep separate buffers and always reap the command, including lsof failures.
	var flowOut, flowStderr bytes.Buffer
	flow := exec.CommandContext(ctx, "/usr/bin/nettop", "-n", "-x", "-L", "2", "-s", strconv.Itoa(int(interval/time.Second)), "-J", "time,bytes_in,bytes_out,rtt_avg")
	flow.Stdout, flow.Stderr = &flowOut, &flowStderr
	var flowErr error
	var flowDone sync.WaitGroup
	flowDone.Add(1)
	go func() { defer flowDone.Done(); flowErr = flow.Run() }()
	defer func() { cancel(); flowDone.Wait() }()
	cmd := exec.CommandContext(ctx, "/usr/sbin/lsof", "-nP", "-i", "-F0pcuftPnT")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return result, fmt.Errorf(localization.T("读取连接超时：%w"), ctx.Err())
	}
	if err != nil && !(cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 && stdout.Len() == 0 && stderr.Len() == 0) {
		return result, fmt.Errorf(localization.T("读取连接失败：%w · %s"), err, strings.TrimSpace(stderr.String()))
	}
	result.Items, err = parseConnections(stdout.Bytes())
	result.Warning = strings.TrimSpace(stderr.String())
	if err != nil {
		return result, err
	}
	flowDone.Wait()
	if err = flowErr; err == nil {
		var samples map[connectionKey]ConnectionMetrics
		samples, err = parseConnectionMetrics(&flowOut)
		if err == nil {
			attachConnectionMetrics(result.Items, samples)
		}
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err != nil {
		result.Warning = strings.TrimSpace(result.Warning + "\n" + fmt.Sprintf(localization.T("连接流量采集失败：%s"), err.Error()))
	}
	result.Warning = strings.TrimSpace(result.Warning + "\n" + flowStderr.String())
	result.UpdatedAt = time.Now().UTC()
	return result, nil
}

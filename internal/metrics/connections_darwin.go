package metrics

import (
	"bytes"
	"context"
	"fmt"
	"localops/internal/localization"
	"os/exec"
	"strings"
	"time"
)

func readConnections(ctx context.Context, timeout time.Duration) (Connections, error) {
	result := Connections{Items: []Connection{}, UpdatedAt: time.Now().UTC()}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
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
	return result, err
}

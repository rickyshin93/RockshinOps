//go:build darwin || linux

package metrics

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"rockshinops/internal/localization"
)

func TestTerminationChild(t *testing.T) {
	mode := os.Args[len(os.Args)-1]
	if mode != "quit-child" && mode != "ignore-child" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	fmt.Println("ready")
	for range signals {
		if mode == "quit-child" {
			os.Exit(0)
		}
		fmt.Println("ignored")
	}
}

func TestTerminateProcess(t *testing.T) {
	for _, mode := range []string{"quit-child", "ignore-child"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminationChild$", "--", mode)
			pipe, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer cmd.Process.Kill()
			reader := bufio.NewReader(pipe)
			if line, err := reader.ReadString('\n'); err != nil || line != "ready\n" {
				t.Fatalf("child not ready: %q %v", line, err)
			}
			items, _, err := readProcesses()
			if err != nil {
				t.Fatal(err)
			}
			identity := ""
			for _, item := range items {
				if item.PID == cmd.Process.Pid {
					identity = strconv.FormatUint(item.start, 10)
				}
			}
			if identity == "" {
				t.Fatal("test child absent from native collector")
			}
			if err := TerminateProcess(cmd.Process.Pid, identity+"1", true); err == nil {
				t.Fatal("stale identity accepted")
			}
			if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
				t.Fatalf("stale request affected child: %v", err)
			}
			if mode == "quit-child" {
				err := TerminatePortProcess(ctx, 3*time.Second, cmd.Process.Pid, identity, 3000)
				if err == nil || err.Error() != localization.T("监听端口已变化，请刷新后重试。") {
					t.Fatalf("unrelated port did not block termination: %v", err)
				}
			}
			if err := TerminateProcess(cmd.Process.Pid, identity, false); err != nil {
				t.Fatal(err)
			}
			if mode == "ignore-child" {
				if line, err := reader.ReadString('\n'); err != nil || line != "ignored\n" {
					t.Fatalf("SIGTERM not received: %q %v", line, err)
				}
				if err := TerminateProcess(cmd.Process.Pid, identity, true); err != nil {
					t.Fatal(err)
				}
			}
			err = cmd.Wait()
			if mode == "quit-child" && err != nil {
				t.Fatalf("graceful handler did not exit cleanly: %v", err)
			}
			if mode == "ignore-child" {
				status := cmd.ProcessState.Sys().(syscall.WaitStatus)
				if !status.Signaled() || status.Signal() != syscall.SIGKILL {
					t.Fatalf("not killed: %v", err)
				}
			}
			if err := TerminateProcess(cmd.Process.Pid, identity, true); err == nil {
				t.Fatal("exited process reported success")
			}
		})
	}
}

func TestTerminateProcessRejectsUnsafeTargets(t *testing.T) {
	for _, pid := range []int{-1, 0, 1, os.Getpid()} {
		if err := TerminateProcess(pid, "123", true); err == nil {
			t.Fatalf("accepted pid %d", pid)
		}
	}
	if err := TerminateProcess(12345, "", false); err == nil {
		t.Fatal("accepted missing identity")
	}
}

func TestPortActionRequiresTheExactTCPListener(t *testing.T) {
	items := []Connection{
		{PID: 42, Protocol: "TCP", State: "LISTEN", Local: "[::1]:3000"},
		{PID: 42, Protocol: "UDP", State: "LISTEN", Local: "*:4000"},
		{PID: 42, Protocol: "TCP", State: "ESTABLISHED", Local: "*:5000"},
		{PID: 43, Protocol: "TCP", State: "LISTEN", Local: "*:6000"},
	}
	connections := Connections{Items: items}
	connections.BindListeners(ProcessList{Items: []Process{{PID: 42, Identity: "111"}, {PID: 43, Identity: "0"}}})
	if items[0].ProcessIdentity != "111" || items[1].ProcessIdentity != "" || items[2].ProcessIdentity != "" || items[3].ProcessIdentity != "" {
		t.Fatalf("listener ownership must use a valid preceding process sample: %+v", items)
	}
	if !ownsListeningPort(items, 42, 3000) {
		t.Fatal("exact IPv6 listener was missed")
	}
	for _, port := range []int{300, 4000, 5000, 6000} {
		if ownsListeningPort(items, 42, port) {
			t.Errorf("wrong listener matched PID 42 on port %d", port)
		}
	}
}

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

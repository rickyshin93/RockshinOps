package metrics

import (
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

func TestNativeConnectionTransferMetrics(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client, err := net.Dial("tcp4", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go io.Copy(io.Discard, server)
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		data := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if _, err := client.Write(data); err != nil {
					return
				}
			}
		}
	}()
	snapshot, err := ReadConnections(context.Background(), 10*time.Second, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Warning != "" {
		t.Fatal(snapshot.Warning)
	}
	for _, row := range snapshot.Items {
		if row.PID == os.Getpid() && row.Local == client.LocalAddr().String() && row.Remote == client.RemoteAddr().String() {
			m := row.Metrics
			if m == nil || m.Sent == nil || *m.Sent == 0 || m.Upload == nil || *m.Upload <= 0 {
				t.Fatalf("known local transfer has no traffic: %+v", m)
			}
			t.Logf("native socket sent=%d upload=%.1f B/s", *m.Sent, *m.Upload)
			return
		}
	}
	t.Fatal("known local connection absent")
}

func BenchmarkNativeConnectionCollection(b *testing.B) {
	for b.Loop() {
		result, err := ReadConnections(context.Background(), 10*time.Second, time.Second)
		if err != nil || result.Warning != "" {
			b.Fatalf("collection: %v %s", err, result.Warning)
		}
	}
}

func TestNativeConnectionCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(50*time.Millisecond, cancel)
	defer timer.Stop()
	start := time.Now()
	result, err := ReadConnections(ctx, 10*time.Second, time.Second)
	if err == nil && result.Warning == "" {
		t.Fatal("canceled collection reported success")
	}
	if elapsed := time.Since(start); elapsed >= 2*time.Second {
		t.Fatalf("cancellation waited for sampling: %v", elapsed)
	}
}

//go:build linux || windows

package metrics

import "testing"

func TestNvidiaResponse(t *testing.T) {
	gpu := parseNvidiaGPU("NVIDIA Test, 37, 512\n")
	if !gpu.Available || gpu.Usage != 37 || gpu.Memory != 512*1024*1024 {
		t.Fatalf("wrong GPU counters: %+v", gpu)
	}
	for _, value := range []string{"NVIDIA Test, N/A, 512\n", "NVIDIA Test, NaN, 512\n", "NVIDIA Test, 37, N/A\n", "unexpected"} {
		if gpu := parseNvidiaGPU(value); gpu.Available || gpu.Error == "" {
			t.Fatalf("unavailable counter accepted: %+v", gpu)
		}
	}
}

package metrics

import "testing"

func TestConnectionRecordsKeepIPv6AndProcessOwnership(t *testing.T) {
	items, err := parseConnections([]byte("p42\x00cExample Helper\x00u501\x00\nf7\x00tIPv6\x00PTCP\x00n[::1]:8000->[::1]:5000\x00TST=ESTABLISHED\x00\nf8\x00tIPv4\x00PUDP\x00n*:5353\x00\n"))
	if err != nil || len(items) != 2 {
		t.Fatalf("%+v %v", items, err)
	}
	if items[0].Remote != "[::1]:5000" || items[0].Process != "Example Helper" || items[1].PID != 42 || items[1].State != "" {
		t.Fatalf("incorrect records: %+v", items)
	}
	if _, err := parseConnections([]byte("pinvalid\x00")); err == nil {
		t.Fatal("accepted invalid process")
	}
}

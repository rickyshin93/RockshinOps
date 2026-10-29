package metrics

import (
	"strings"
	"testing"
)

func BenchmarkParseConnections(b *testing.B) {
	data := []byte(strings.Repeat("p123\x00cProcess\x00u501\x00f10\x00tIPv4\x00PTCP\x00n127.0.0.1:3456->127.0.0.1:443\x00TST=ESTABLISHED\x00\n", 1500))
	b.ResetTimer()
	for b.Loop() {
		rows, err := parseConnections(data)
		if err != nil || len(rows) != 1500 {
			b.Fatalf("parse: %v / %d rows", err, len(rows))
		}
	}
}

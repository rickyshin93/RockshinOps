package metrics

import (
	"strings"
	"testing"
)

const trafficHeader = "time,,bytes_in,bytes_out,rtt_avg,\n"

func TestConnectionMetricsRTTFormats(t *testing.T) {
	for _, test := range []struct {
		text  string
		want  float64
		valid bool
	}{
		{"6.75 ms", 6.75, true},
		{"999.99 ms", 999.99, true},
		{"1000.00 m", 1000, true},
		{"1019.84 m", 1019.84, true},
		{"9999.99 m", 9999.99, true},
		{"6.75 m", 0, false},
		{"1019.84 s", 0, false},
		{"1019.84", 0, false},
		{"NaN ms", 0, false},
		{"+Inf ms", 0, false},
		{"-1.00 ms", 0, false},
	} {
		t.Run(test.text, func(t *testing.T) {
			row := "tcp4 127.0.0.1:5010<->127.0.0.1:80,"
			input := trafficHeader + "12:00:00.0,App.42,,,,\n12:00:00.0," + row + "100,50,6.75 ms,\n" +
				trafficHeader + "12:00:01.0,App.42,,,,\n12:00:01.0," + row + "300,100," + test.text + ",\n"
			samples, err := parseConnectionMetrics(strings.NewReader(input))
			if !test.valid {
				if err == nil {
					t.Fatal("accepted invalid RTT")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			items := []Connection{{PID: 42, Protocol: "TCP", Local: "127.0.0.1:5010", Remote: "127.0.0.1:80", State: "ESTABLISHED"}}
			attachConnectionMetrics(items, samples)
			m := items[0].Metrics
			if m == nil || m.RTT == nil || *m.RTT != test.want || m.Download == nil || *m.Download != 200 || m.Upload == nil || *m.Upload != 50 {
				t.Fatalf("RTT or traffic lost: %+v", m)
			}
		})
	}
}

func TestConnectionMetricsSkipsQUICFlowRows(t *testing.T) {
	// macOS 15 CI exposes QUIC flows alongside TCP/UDP sockets. They cannot
	// be matched to the lsof TCP/UDP rows, but must not discard other metrics.
	for _, flow := range []string{
		"quic4 192.168.64.19:50488<->17.248.192.28:443",
		"quic6 fe80::1%en0.50488<->fe80::2%en0.443",
	} {
		t.Run(flow[:5], func(t *testing.T) {
			input := trafficHeader + "12:00:00.0,App.42,9999,9999,,\n" +
				"12:00:00.0," + flow + ",9000,9000,5.00 ms,\n" +
				"12:00:00.0,tcp4 127.0.0.1:5010<->127.0.0.1:80,100,50,5.00 ms,\n" +
				trafficHeader + "12:00:01.0,App.42,9999,9999,,\n" +
				"12:00:01.0," + flow + ",9500,9500,6.00 ms,\n" +
				"12:00:01.0,tcp4 127.0.0.1:5010<->127.0.0.1:80,300,100,6.00 ms,\n"
			samples, err := parseConnectionMetrics(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			key := connectionKey{42, "TCP", "127.0.0.1:5010", "127.0.0.1:80"}
			metrics, ok := samples[key]
			if !ok || len(samples) != 1 || metrics.Download == nil || *metrics.Download != 200 || metrics.Upload == nil || *metrics.Upload != 50 {
				t.Fatalf("QUIC flow changed TCP ownership or counters: %+v", samples)
			}
		})
	}
}

func TestConnectionMetricsRatesAndOwnership(t *testing.T) {
	input := trafficHeader + "12:00:00.0,App.Helper.42,9999999,8888888,,\n" +
		"12:00:00.0,tcp4 127.0.0.1:5010<->127.0.0.1:80,100,50,5.25 ms,\n" +
		"12:00:00.0,udp4 *:5353<->*:*,0,0,,\n" + trafficHeader +
		"12:00:02.0,App.Helper.42,9999999,8888888,,\n" +
		"12:00:02.0,tcp4 127.0.0.1:5010<->127.0.0.1:80,500,150,6.75 ms,\n" +
		"12:00:02.0,udp4 *:5353<->*:*,0,0,,\n"
	samples, err := parseConnectionMetrics(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	items := []Connection{
		{PID: 42, Protocol: "TCP", Local: "127.0.0.1:5010", Remote: "127.0.0.1:80", State: "ESTABLISHED"},
		{PID: 42, Protocol: "UDP", Local: "*:5353"},
		{PID: 99, Protocol: "TCP", Local: "127.0.0.1:5010", Remote: "127.0.0.1:80"},
		{PID: 42, Protocol: "TCP", Local: "*:80", State: "LISTEN"},
	}
	attachConnectionMetrics(items, samples)
	m := items[0].Metrics
	if m == nil || *m.Received != 500 || *m.Sent != 150 || *m.Download != 200 || *m.Upload != 50 || *m.RTT != 6.75 {
		t.Fatalf("wrong socket counters: %+v", m)
	}
	u := items[1].Metrics
	if u == nil || *u.Download != 0 || *u.Received != 0 || u.RTT != nil {
		t.Fatalf("zero is a measurement, UDP RTT is not: %+v", u)
	}
	if items[2].Metrics != nil || items[3].Metrics != nil {
		t.Fatal("attached traffic to another PID or listener")
	}
}

func TestConnectionMetricsBaselineResetAndMissing(t *testing.T) {
	for _, test := range []struct{ name, before, after string }{
		{"new socket", "", "100,200,0.00 ms,"},
		{"counter reset", "100,200,0.00 ms,", "1,2,0.00 ms,"},
		{"unavailable", ",,,", ",,,"},
	} {
		t.Run(test.name, func(t *testing.T) {
			row := "tcp4 127.0.0.1:5010<->127.0.0.1:80,"
			input := trafficHeader + "12:00:00.0,App.42,,,,\n"
			if test.before != "" {
				input += "12:00:00.0," + row + test.before + "\n"
			}
			input += trafficHeader + "12:00:01.0,App.42,,,,\n12:00:01.0," + row + test.after + "\n"
			samples, err := parseConnectionMetrics(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range samples {
				if m.Download != nil || m.Upload != nil || m.RTT != nil {
					t.Fatalf("invented rates/RTT: %+v", m)
				}
			}
		})
	}
}

func TestConnectionMetricsIPv6AndMidnight(t *testing.T) {
	row := "tcp6 fe80::1%en0.5010<->fe80::2%en0.80,"
	input := trafficHeader + "23:59:59.5,App.42,,,,\n23:59:59.5," + row + "100,100,2.00 ms,\n" + trafficHeader + "00:00:00.5,App.42,,,,\n00:00:00.5," + row + "200,300,2.00 ms,\n"
	samples, err := parseConnectionMetrics(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	items := []Connection{{PID: 42, Protocol: "TCP", Local: "[fe80:e::1]:5010", Remote: "[fe80:e::2]:80", State: "ESTABLISHED"}}
	attachConnectionMetrics(items, samples)
	if m := items[0].Metrics; m == nil || *m.Download != 100 || *m.Upload != 200 {
		t.Fatalf("IPv6/time rollover: %+v", m)
	}
	duplicates := append(items, items[0])
	duplicates[0].Metrics = nil
	duplicates[1].Metrics = nil
	attachConnectionMetrics(duplicates, samples)
	if duplicates[0].Metrics != nil || duplicates[1].Metrics != nil {
		t.Fatal("ambiguous lsof tuple was assigned traffic")
	}
	input = strings.ReplaceAll(input, row+"200,300,2.00 ms,\n", row+"200,300,2.00 ms,\n00:00:00.5,"+row+"200,300,2.00 ms,\n")
	samples, err = parseConnectionMetrics(strings.NewReader(input))
	if err != nil || len(samples) != 0 {
		t.Fatalf("ambiguous nettop tuple: %+v %v", samples, err)
	}
}

func TestConnectionMetricsRejectMalformedOutput(t *testing.T) {
	for _, input := range []string{"", trafficHeader, "time,,bytes_in,\n", trafficHeader + "bad,App.42,,,,\nbad,tcp4 x<->y,1,2,3 ms,\n", trafficHeader + "12:00:00.0,App.42,,,,\n12:00:00.0,tcp4 x<->y,-1,2,3 ms,\n"} {
		if _, err := parseConnectionMetrics(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid CSV: %q", input)
		}
	}
}

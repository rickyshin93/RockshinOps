package fileindex

import (
	"strings"
	"testing"
)

func score(q, name string) int {
	return Score([]byte(strings.ToLower(q)), []byte(strings.ToLower(name)), []byte(name))
}

func TestScoreOrder(t *testing.T) {
	// Each query lists names from best to worst match; "" names must not match.
	for _, c := range []struct {
		query string
		names []string
	}{
		{"第一题", []string{"第一题", "第一题复核成绩.xlsx", "第一题复核成绩.xlsx.inspect.ndjson"}},
		{"复核成绩", []string{"复核成绩.xlsx", "第一题复核成绩.xlsx", "第一题复核成绩.xlsx.inspect.ndjson"}},
		{"status", []string{"STATUS.md", "STATUS-archive.md", "app_status.go", "lastatus.txt"}},
		{"sta", []string{"STATUS.md", "my-stats.csv", "mostat.txt", "sXtXa.txt"}},
		{"vsc", []string{"Visual Studio Code", "vision-scanner"}},
		{"edit", []string{"TextEdit", "credit.txt"}},
		{"actmon", []string{"Activity Monitor"}},
	} {
		prev := 1 << 30
		for _, n := range c.names {
			s := score(c.query, n)
			if s <= 0 || s >= prev {
				t.Errorf("%q: %q scored %d after %d", c.query, n, s, prev)
			}
			prev = s
		}
	}
	for _, c := range [][2]string{{"zzz", "STATUS.md"}, {"复核成绩表", "第一题复核成绩.xlsx"}, {"ab", "ba"}} {
		if s := score(c[0], c[1]); s != 0 {
			t.Errorf("%q matched %q with %d", c[0], c[1], s)
		}
	}
}

func TestScoreTextMatchesPinyin(t *testing.T) {
	for _, c := range [][2]string{
		{"diyi", "第一题复核成绩.xlsx"},   // full pinyin
		{"fhcj", "第一题复核成绩.xlsx"},   // initials
		{"fuhecj", "第一题复核成绩.xlsx"}, // full and initials mixed
		{"第yi", "第一题复核成绩.xlsx"},   // characters and pinyin mixed
		{"dyt", "第一题复核成绩.xlsx"},
		{"jsq", "计算器"},
		{"第一", "第一题复核成绩.xlsx"},
	} {
		if ScoreText([]byte(c[0]), c[1]) <= 0 {
			t.Errorf("%q does not match %q", c[0], c[1])
		}
	}
	// Letters merely in order across syllables are not a pinyin match.
	for _, q := range []string{"zzz", "deep", "dt", "iyi"} {
		if s := ScoreText([]byte(q), "第一题复核成绩.xlsx.inspect.ndjson"); s != 0 {
			t.Errorf("%q matched with %d", q, s)
		}
	}
	// Syllable starts count as word starts, so "yiti" beats a plain substring.
	if a, b := ScoreText([]byte("yiti"), "第一题"), ScoreText([]byte("yiti"), "byitis"); a <= b {
		t.Errorf("syllable start %d should beat substring %d", a, b)
	}
	// Typing the characters themselves beats typing their pinyin.
	if a, b := ScoreText([]byte("di"), "di.txt"), ScoreText([]byte("di"), "第.txt"); a <= b {
		t.Errorf("direct %d should beat pinyin %d", a, b)
	}
}

package fileindex

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// pinyinPenalty ranks a match through pinyin just below the same match on
// the characters themselves.
const pinyinPenalty = 20

// syllable is one character of a name: a Chinese character with its pinyin,
// or any other character as itself, lower-cased.
type syllable struct {
	char, pinyin string // pinyin is "" for other characters
}

var plain = pinyin.Args{Style: pinyin.Normal}

// spell splits a name into syllables, or returns nil when it has no Chinese
// characters. Characters with several readings use the most common one.
func spell(name string) []syllable {
	if !strings.ContainsFunc(name, isHan) {
		return nil
	}
	out := make([]syllable, 0, len(name))
	for _, r := range name {
		s := syllable{char: string(unicode.ToLower(r))}
		if isHan(r) {
			if p := pinyin.SinglePinyin(r, plain); len(p) > 0 {
				s.pinyin = p[0]
			}
		}
		out = append(out, s)
	}
	return out
}

func isHan(r rune) bool { return unicode.Is(unicode.Han, r) }

// spells reports where q, read from syllable k on, ends: each Chinese
// character may be typed as itself, in full pinyin or as a prefix of it
// (at least its first letter); other characters as themselves.
func spells(q string, syl []syllable, k int) (int, bool) {
	if q == "" {
		return k, true
	}
	if k == len(syl) {
		return 0, false
	}
	s := syl[k]
	if strings.HasPrefix(q, s.char) {
		if end, ok := spells(q[len(s.char):], syl, k+1); ok {
			return end, true
		}
	}
	for l := min(len(s.pinyin), len(q)); l >= 1; l-- {
		if q[:l] == s.pinyin[:l] {
			if end, ok := spells(q[l:], syl, k+1); ok {
				return end, true
			}
		}
	}
	return 0, false
}

// scorePinyin ranks q spelling the name from its first character (exact or
// prefix) or from a later Chinese character (word start).
func scorePinyin(q string, syl []syllable) int {
	if len(q) > 64 {
		return 0
	}
	length := min(len(syl), 99)
	for k, s := range syl {
		if k > 0 && s.pinyin == "" {
			continue
		}
		end, ok := spells(q, syl, k)
		switch {
		case !ok:
			continue
		case k == 0 && end == len(syl):
			return 1000 - pinyinPenalty
		case k == 0:
			return 900 - length - pinyinPenalty
		}
		return 700 - length - pinyinPenalty
	}
	return 0
}

// scoreSpelled is Score on a name, or on its pinyin when that ranks higher.
func scoreSpelled(q, lower, name []byte, syl []syllable) int {
	best := Score(q, lower, name)
	if syl != nil {
		best = max(best, scorePinyin(string(q), syl))
	}
	return best
}

// ScoreText scores the lower-case query q against a name, its characters
// or their pinyin, for callers without an index (such as app names).
func ScoreText(q []byte, name string) int {
	return scoreSpelled(q, []byte(strings.ToLower(name)), []byte(name), spell(name))
}

package fileindex

import (
	"bytes"
	"unicode/utf8"
)

// Score ranks how well the lower-case query q matches a name, given as its
// lower-case form and as shown: exact, prefix, word start, initials,
// substring, then letters in order. Shorter names win within a tier; 0 is
// no match. It does not allocate, as it runs for every indexed name.
func Score(q, lower, name []byte) int {
	if len(q) == 0 || !inOrder(q, lower) {
		return 0
	}
	length := min(utf8.RuneCount(lower), 99)
	switch {
	case bytes.Equal(lower, q):
		return 1000
	case bytes.HasPrefix(lower, q):
		return 900 - length
	}
	// Word starts follow a separator or a lower-to-upper case change (ASCII
	// only); their letters in turn are the initials, which q may begin.
	k, initials := 1, q[0] == lower[0]
	for i := 1; i < len(name) && i < len(lower); i++ {
		if !wordStart(name, i) {
			continue
		}
		if bytes.HasPrefix(lower[i:], q) {
			return 700 - length
		}
		if initials && k < len(q) {
			if lower[i] == q[k] {
				k++
			} else {
				initials = false
			}
		}
	}
	if initials && k == len(q) {
		return 600 - length
	}
	if bytes.Contains(lower, q) {
		return 500 - length
	}
	return 300 - length
}

func wordStart(name []byte, i int) bool {
	c, prev := name[i], name[i-1]
	if !isAlnum(c) {
		return false
	}
	return !isAlnum(prev) && prev < utf8.RuneSelf || isLower(prev) && isUpper(c)
}

func isLower(c byte) bool { return 'a' <= c && c <= 'z' }
func isUpper(c byte) bool { return 'A' <= c && c <= 'Z' }
func isAlnum(c byte) bool { return isLower(c) || isUpper(c) || '0' <= c && c <= '9' }

// inOrder reports whether the bytes of q appear in s in order, ignoring
// spaces in q; for UTF-8 this is what letters in order need, and it is the
// cheap test every name takes first.
func inOrder(q, s []byte) bool {
	j := 0
	for _, c := range q {
		if c == ' ' {
			continue
		}
		k := bytes.IndexByte(s[j:], c)
		if k < 0 {
			return false
		}
		j += k + 1
	}
	return true
}

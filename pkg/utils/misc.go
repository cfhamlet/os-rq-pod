package utils

import "unicode/utf8"

// AbsInt64 TODO
func AbsInt64(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

// Reverse TODO
func Reverse(s string) string {
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}

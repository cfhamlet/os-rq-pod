package utils

import (
	"fmt"
	"sync"
	"unicode/utf8"
)

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

// Text TODO
func Text(obj interface{}) string {
	return fmt.Sprintf("%s", obj)
}

// Merge TODO
func Merge(cs ...<-chan error) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan error) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

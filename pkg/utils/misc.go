package utils

// AbsInt64 TODO
func AbsInt64(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

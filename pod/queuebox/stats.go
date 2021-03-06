package queuebox

import (
	"sync/atomic"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/prep/average"
)

// Stats TODO
type Stats struct {
	requestNum      int64
	requestInputWS  *average.SlidingWindow
	requestOutputWS *average.SlidingWindow
}

// WindowTotal TODO
func WindowTotal(window *average.SlidingWindow, second time.Duration) int64 {
	t, _ := window.Total(second * time.Second)
	return t
}

// Stats TODO
func (stats *Stats) Stats() sth.Result {
	result := sth.Result{
		"total": stats.RequestNum(),
		"speed": sth.Result{
			"input_5s":  WindowTotal(stats.requestInputWS, 5),
			"output_5s": WindowTotal(stats.requestOutputWS, 5),
		},
	}
	return result
}

// IncrRequestNum TODO
func (stats *Stats) IncrRequestNum(d int64) int64 {
	return atomic.AddInt64(&(stats.requestNum), d)
}

// RequestNum TODO
func (stats *Stats) RequestNum() int64 {
	return atomic.LoadInt64(&(stats.requestNum))
}

// UpdateInput TODO
func (stats *Stats) UpdateInput(d int64) {
	stats.IncrRequestNum(d)
	stats.requestInputWS.Add(d)
}

// UpdateOutput TODO
func (stats *Stats) UpdateOutput(d int64) {
	stats.IncrRequestNum(0 - d)
	stats.requestOutputWS.Add(d)
}

// MustNewMinuteWindow TODO
func MustNewMinuteWindow() *average.SlidingWindow {
	return average.MustNew(time.Minute, time.Second)
}

// NewStats TODO
func NewStats() *Stats {
	return &Stats{
		0,
		MustNewMinuteWindow(),
		MustNewMinuteWindow(),
	}
}

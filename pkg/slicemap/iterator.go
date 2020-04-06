package slicemap

import (
	"math/rand"
	"time"
)

// IterFunc TODO
type IterFunc func(Item) bool

// Iterator TODO
type Iterator interface {
	Iter(IterFunc)
}

// BaseIter TODO
type BaseIter struct {
	*Map
}

// Iter TODO
func (iter *BaseIter) Iter(f IterFunc) {
	iter.RLock()
	defer iter.RUnlock()
	for i := 0; i < iter.maxIdx; i++ {
		if !f(iter.items[i]) {
			break
		}
	}
}

// NewBaseIter TODO
func NewBaseIter(m *Map) *BaseIter {
	return &BaseIter{m}
}

// ReverseIter TODO
type ReverseIter struct {
	*Map
}

// Iter TODO
func (iter *ReverseIter) Iter(f IterFunc) {
	iter.RLock()
	defer iter.RUnlock()
	for i := iter.maxIdx - 1; i >= 0; i-- {
		if !f(iter.items[i]) {
			break
		}
	}
}

// NewReverseIter TODO
func NewReverseIter(m *Map) *ReverseIter {
	return &ReverseIter{m}
}

// SubIter TODO
type SubIter struct {
	*Map
	start int
	n     int
}

// NewSubIter TODO
func NewSubIter(m *Map, start, n int) *SubIter {
	return &SubIter{m, start, n}
}

// Iter TODO
func (iter *SubIter) Iter(f IterFunc) {
	iter.RLock()
	defer iter.RUnlock()

	l := iter.size()
	if iter.start > l {
		return
	}
	end := iter.start + iter.n
	if end > l {
		end = l
	}

	for i := iter.start; i < end; i++ {
		if !f(iter.items[i]) {
			break
		}
	}
}

// RandomKIter TODO
type RandomKIter struct {
	*Map
	r *rand.Rand
	k int
}

// NewRandomKIter TODO
func NewRandomKIter(m *Map, k int) *RandomKIter {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &RandomKIter{m, r, k}
}

// Iter TODO
func (iter *RandomKIter) Iter(f IterFunc) {
	iter.RLock()
	defer iter.RUnlock()
	shuffled := iter.r.Perm(iter.size())
	for i := 0; i < iter.maxIdx; {
		if !f(iter.items[shuffled[i]]) {
			break
		}
		i++
		if i >= iter.k {
			break
		}
	}
}

// CycleIter TODO
type CycleIter struct {
	*Map
	cur int
}

// NewCycleIter TODO
func NewCycleIter(m *Map, start int) *CycleIter {
	return &CycleIter{m, start}
}

// Iter TODO
func (iter *CycleIter) Iter(f IterFunc) {
	iter.RLock()
	defer iter.RUnlock()
	l := iter.size()
	if l <= 0 {
		return
	}
	iter.cur = iter.cur % l

	for i := 0; i < l; i++ {
		b := f(iter.items[iter.cur])
		iter.cur++
		if iter.cur >= l {
			iter.cur = 0
		}
		if !b {
			break
		}
	}
}

// CycleStepIter TODO
type CycleStepIter struct {
	*Map
	cur   int
	steps int
}

// NewCycleStepIter TODO
func NewCycleStepIter(m *Map, start, steps int) *CycleStepIter {
	return &CycleStepIter{m, start, steps}
}

// SetSteps TODO
func (iter *CycleStepIter) SetSteps(steps int) {
	iter.steps = steps
}

// Iter TODO
func (iter *CycleStepIter) Iter(f IterFunc) {
	iter.RLock()
	defer iter.RUnlock()
	l := iter.size()
	if l <= 0 {
		return
	}
	iter.cur = iter.cur % l

	for i := 0; i < iter.steps; i++ {
		b := f(iter.items[iter.cur])
		iter.cur++
		if iter.cur >= l {
			iter.cur = 0
		}
		if !b {
			break
		}
	}
}

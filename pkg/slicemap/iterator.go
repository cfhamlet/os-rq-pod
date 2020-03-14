package slicemap

import (
	"math/rand"
	"time"
)

// Iterator TODO
type Iterator interface {
	Iter(func(Item))
}

// FastIter TODO
type FastIter struct {
	m *Map
}

// Iter TODO
func (iter *FastIter) Iter(f func(Item)) {
	for _, item := range iter.m.items {
		if item == nil {
			break
		}
		f(item)
	}
}

// NewFastIter TODO
func NewFastIter(m *Map) *FastIter {
	return &FastIter{m}
}

// DeleteSafeIter TODO
type DeleteSafeIter struct {
	m *Map
}

// NewDeleteSafeIter TODO
func NewDeleteSafeIter(m *Map) *DeleteSafeIter {
	return &DeleteSafeIter{m}
}

// Iter TODO
func (iter *DeleteSafeIter) Iter(f func(Item)) {
	m := iter.m
	i := 0
	for _, item := range m.items {
		if item == nil {
			break
		}
		f(item)
		if i < len(m.items) {
			newItem := m.items[i]
			for newItem != nil && newItem.ItemID() != item.ItemID() {
				f(newItem)
				item = newItem
				if i < len(m.items) {
					newItem = m.items[i]
				}
			}
		}
		i++
	}
}

// SubIter TODO
type SubIter struct {
	m     *Map
	start int
	n     int
}

// NewSubIter TODO
func NewSubIter(m *Map, start, n int) *SubIter {
	return &SubIter{m, start, n}
}

// Iter TODO
func (iter *SubIter) Iter(f func(Item)) {
	for _, item := range iter.m.items[iter.start : iter.start+iter.n] {
		if item == nil {
			break
		}
		f(item)
	}
}

// RandomKIter TODO
type RandomKIter struct {
	m *Map
	r *rand.Rand
	k int
}

// NewRandomKIter TODO
func NewRandomKIter(m *Map, k int) *RandomKIter {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &RandomKIter{m, r, k}
}

// Iter TODO
func (iter *RandomKIter) Iter(f func(Item)) {
	i := 0
	shuffled := iter.r.Perm(iter.m.maxIdx)
	for _, idx := range shuffled {
		f(iter.m.items[idx])
		i++
		if i >= iter.k {
			break
		}
	}
}

// CycleIter TODO
type CycleIter struct {
	m       *Map
	start   int
	cur     int
	steps   int
	started bool
}

// NewCycleIter TODO
func NewCycleIter(m *Map, start, steps int) *CycleIter {
	return &CycleIter{m, start, start, steps, false}
}

// SetSteps TODO
func (iter *CycleIter) SetSteps(steps int) {
	iter.steps = steps
}

// Iter TODO
func (iter *CycleIter) Iter(f func(Item)) {
	if iter.Done() {
		return
	} else if !iter.started {
		iter.start = iter.start % iter.m.Size()
		iter.cur = iter.start
		iter.started = true
	}

	for i := 0; i < iter.steps; i++ {
		f(iter.m.items[iter.cur])
		iter.cur++
		if iter.cur >= iter.m.Size() {
			iter.cur = 0
		}
		if iter.cur == iter.start {
			break
		}
	}
}

// Done TODO
func (iter *CycleIter) Done() bool {
	if iter.m.Size() <= 0 {
		return true
	}
	if !iter.started {
		return false
	}

	if iter.cur != iter.start {
		return false
	}

	return true
}

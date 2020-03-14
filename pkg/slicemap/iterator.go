package slicemap

import (
	"math/rand"
	"time"
)

// Iterator TODO
type Iterator interface {
	Iter(func(Item))
	Break()
}

// FastIter TODO
type FastIter struct {
	m *Map
	b bool
}

// Iter TODO
func (iter *FastIter) Iter(f func(Item)) {
	for _, item := range iter.m.items {
		if item == nil {
			break
		}
		f(item)
		if iter.b {
			iter.b = false
			break
		}
	}
}

// Break TODO
func (iter *FastIter) Break() {
	iter.b = true
}

// NewFastIter TODO
func NewFastIter(m *Map) *FastIter {
	return &FastIter{m, false}
}

// DeleteSafeIter TODO
type DeleteSafeIter struct {
	m *Map
	b bool
}

// Break TODO
func (iter *DeleteSafeIter) Break() {
	iter.b = true
}

// NewDeleteSafeIter TODO
func NewDeleteSafeIter(m *Map) *DeleteSafeIter {
	return &DeleteSafeIter{m, false}
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
		if iter.b {
			iter.b = false
			break
		}
		if i < len(m.items) {
			newItem := m.items[i]
			for newItem != nil && newItem.ItemID() != item.ItemID() {
				f(newItem)
				if iter.b {
					iter.b = false
					break
				}
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
	b     bool
}

// NewSubIter TODO
func NewSubIter(m *Map, start, n int) *SubIter {
	return &SubIter{m, start, n, false}
}

// Break TODO
func (iter *SubIter) Break() {
	iter.b = true
}

// Iter TODO
func (iter *SubIter) Iter(f func(Item)) {
	for _, item := range iter.m.items[iter.start : iter.start+iter.n] {
		if item == nil {
			break
		}
		f(item)
		if iter.b {
			iter.b = false
			break
		}
	}
}

// RandomKIter TODO
type RandomKIter struct {
	m *Map
	r *rand.Rand
	k int
	b bool
}

// NewRandomKIter TODO
func NewRandomKIter(m *Map, k int) *RandomKIter {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &RandomKIter{m, r, k, false}
}

// Break TODO
func (iter *RandomKIter) Break() {
	iter.b = true
}

// Iter TODO
func (iter *RandomKIter) Iter(f func(Item)) {
	i := 0
	shuffled := iter.r.Perm(iter.m.maxIdx)
	for _, idx := range shuffled {
		item := iter.m.items[idx]
		if item == nil {
			break
		}
		f(item)
		if iter.b {
			iter.b = false
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
	m     *Map
	cur   int
	steps int
	b     bool
}

// NewCycleIter TODO
func NewCycleIter(m *Map, start, steps int) *CycleIter {
	return &CycleIter{m, start, steps, false}
}

// SetSteps TODO
func (iter *CycleIter) SetSteps(steps int) {
	iter.steps = steps
}

// Iter TODO
func (iter *CycleIter) Iter(f func(Item)) {
	l := iter.m.Size()
	if l <= 0 {
		return
	}
	iter.cur = iter.cur % l

	for i := 0; i < iter.steps; i++ {
		item := iter.m.items[iter.cur]
		if item == nil {
			break
		}
		f(item)
		iter.cur++
		if iter.cur >= l {
			iter.cur = 0
		}
		if iter.b {
			iter.b = false
			break
		}
	}
}

// Break TODO
func (iter *CycleIter) Break() {
	iter.b = true
}

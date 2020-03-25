package slicemap

import "sync"

const minShrinkSize = 1024

// Item TODO
type Item interface {
	ItemID() uint64
}

// Map TODO
type Map struct {
	items  []Item
	idxMap map[uint64]int
	maxIdx int
	*sync.RWMutex
}

// New TODO
func New() *Map {
	return &Map{
		items:   make([]Item, 0),
		idxMap:  make(map[uint64]int),
		maxIdx:  0,
		RWMutex: &sync.RWMutex{},
	}
}

// Add TODO
func (m *Map) Add(item Item) bool {
	m.Lock()
	defer m.Unlock()
	id := item.ItemID()
	if idx, ok := m.idxMap[id]; ok {
		m.items[idx] = item
		return false
	}

	m.idxMap[id] = m.maxIdx
	if m.maxIdx < len(m.items) {
		m.items[m.maxIdx] = item
	} else {
		m.items = append(m.items, item)
	}
	m.maxIdx++
	return true
}

// Get TODO
func (m *Map) Get(id uint64) Item {
	m.RLock()
	defer m.RUnlock()
	if idx, ok := m.idxMap[id]; ok {
		return m.items[idx]
	}
	return nil
}

// Size TODO
func (m *Map) Size() int {
	m.RLock()
	defer m.RUnlock()
	return m.maxIdx
}

func (m *Map) shrink() {
	if m.maxIdx < len(m.items) {
		m.items = m.items[0:m.maxIdx]
	}
}

// Shrink TODO
func (m *Map) Shrink() {
	m.Lock()
	defer m.Unlock()
	m.shrink()
}

func (m *Map) delete(id uint64) bool {
	if idx, ok := m.idxMap[id]; ok {
		delete(m.idxMap, id)
		if idx == m.maxIdx-1 {
			m.items[idx] = nil
			m.maxIdx--
		} else {
			m.maxIdx--
			item := m.items[m.maxIdx]
			m.items[idx] = item
			m.idxMap[item.ItemID()] = idx
			m.items[m.maxIdx] = nil
		}
	} else {
		return false
	}

	// Shrink to prevent slice increasing with no limit.
	overCount := len(m.items) - m.maxIdx
	if m.maxIdx > minShrinkSize && overCount > 0 &&
		(float64(overCount)/float64(m.maxIdx) > 0.1) {
		m.shrink()
	}
	return true
}

// Delete TODO
func (m *Map) Delete(item Item) bool {
	m.Lock()
	defer m.Unlock()
	return m.delete(item.ItemID())
}

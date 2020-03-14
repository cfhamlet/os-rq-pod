package slicemap

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
}

// NewMap TODO
func NewMap() *Map {
	return &Map{
		items:  make([]Item, 0),
		idxMap: make(map[uint64]int),
		maxIdx: 0,
	}
}

// Add TODO
func (m *Map) Add(item Item) {
	id := item.ItemID()
	if idx, ok := m.idxMap[id]; ok {
		m.items[idx] = item
		return
	}

	m.idxMap[id] = m.maxIdx
	if m.maxIdx < len(m.items) {
		m.items[m.maxIdx] = item
	} else {
		m.items = append(m.items, item)
	}
	m.maxIdx++
}

// Get TODO
func (m *Map) Get(id int64) Item {
	if idx, ok := m.idxMap[id]; ok {
		return m.items[idx]
	}
	return nil
}

// Size TODO
func (m *Map) Size() int {
	return m.maxIdx
}

func (m *Map) shrink() {
	if m.maxIdx < len(m.items) {
		m.items = m.items[0:m.maxIdx]
	}
}

// Shrink TODO
func (m *Map) Shrink() {
	m.shrink()
}

func (m *Map) delete(id int64) {
	if curIdx, ok := m.idxMap[id]; ok {
		delete(m.idxMap, id)
		if curIdx == m.maxIdx-1 {
			m.items[curIdx] = nil
			m.maxIdx--
		} else {
			m.maxIdx--
			item := m.items[m.maxIdx]
			m.items[curIdx] = item
			m.idxMap[item.ItemID()] = curIdx
			m.items[m.maxIdx] = nil
		}
	} else {
		return
	}

	// Shrink to prevent slice increasing with no limit.
	overCount := len(m.items) - m.maxIdx
	if m.maxIdx > minShrinkSize && overCount > 0 &&
		(float64(overCount)/float64(m.maxIdx) > 0.1) {
		m.shrink()
	}
}

// Delete TODO
func (m *Map) Delete(item Item) {
	m.delete(item.ItemID())
}

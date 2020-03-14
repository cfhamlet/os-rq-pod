package slicemap

const minShrinkSize = 1024

// Item TODO
type Item interface {
	ItemID() int
}

// Map TODO
type Map struct {
	items  []Item
	idxMap map[int]int
	maxIdx int
}

// NewMap TODO
func NewMap() *Map {
	return &Map{
		items:  make([]Item, 0),
		idxMap: make(map[int]int),
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
func (m *Map) Get(id int) Item {
	if idx, ok := m.idxMap[id]; ok {
		return m.items[idx]
	}
	return nil
}

// Len TODO
func (m *Map) Len() int {
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

func (m *Map) delete(id int) {
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
	rdt := len(m.items) - m.maxIdx
	if m.maxIdx > minShrinkSize && rdt > 0 &&
		(float32(rdt)/float32(m.maxIdx) > 0.1) {
		m.shrink()
	}
}

// Delete TODO
func (m *Map) Delete(item Item) {
	m.delete(item.ItemID())
}

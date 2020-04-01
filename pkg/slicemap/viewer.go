package slicemap

import "sync"

// ViewFunc TODO
type ViewFunc func(Item)

// Viewer TODO
type Viewer struct {
	*Map
	cLock *sync.RWMutex
}

// NewViewer TODO
func NewViewer(m *Map) *Viewer {
	if m == nil {
		m = New()
	}
	return &Viewer{m, &sync.RWMutex{}}
}

// View TODO
func (viewer *Viewer) View(id uint64, f ViewFunc) {
	viewer.RLock()
	defer viewer.RUnlock()
	item := viewer.get(id)
	f(item)
}

// GetOrAdd TODO
func (viewer *Viewer) GetOrAdd(id uint64, f func(Item) Item) {
	viewer.RLock()
	defer viewer.RUnlock()

	item := viewer.get(id)
	if item == nil {
		viewer.cLock.Lock()
		defer viewer.cLock.Unlock()
		item := viewer.get(id)
		new := f(item)
		if item == nil {
			if new != nil {
				viewer.add(new)
			}
		}
	} else {
		f(item)
	}
}

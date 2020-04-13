package slicemap

// ViewFunc TODO
type ViewFunc func(Item)

// Viewer TODO
type Viewer struct {
	*Map
}

// NewViewer TODO
func NewViewer(m *Map) *Viewer {
	if m == nil {
		m = New()
	}
	return &Viewer{m}
}

// View TODO
func (viewer *Viewer) View(id uint64, f ViewFunc) {
	viewer.RLock()
	defer viewer.RUnlock()
	item := viewer.get(id)
	f(item)
}

// GetOrAdd TODO
func (viewer *Viewer) GetOrAdd(id uint64, f func(Item) Item) bool {
	viewer.Lock()
	defer viewer.Unlock()

	item := viewer.get(id)
	if item == nil {
		new := f(item)
		if new != nil {
			viewer.add(new)
			return true
		}
	} else {
		f(item)
	}
	return false
}

// GetAndDelete TODO
func (viewer *Viewer) GetAndDelete(id uint64, f func(Item) bool) bool {
	viewer.Lock()
	defer viewer.Unlock()
	item := viewer.get(id)
	if item != nil {
		if f(item) {
			return viewer.delete(id)
		}
	}
	return false
}

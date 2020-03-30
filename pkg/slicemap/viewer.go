package slicemap

// ExistViewer TODO
type ExistViewer struct {
	*Map
}

type ViewFunc func(Item)

// View TODO
func (viewer *ExistViewer) View(id uint64, f ViewFunc) {
	viewer.RLock()
	defer viewer.RUnlock()
	if idx, ok := viewer.idxMap[id]; ok {
		f(viewer.items[idx])
	}
}

// MustViewer TODO
type MustViewer struct {
	*Map
}

// View TODO
func (viewer *MustViewer) View(id uint64, f ViewFunc) {
	viewer.RLock()
	defer viewer.RUnlock()
	if idx, ok := viewer.idxMap[id]; ok {
		f(viewer.items[idx])
	} else {
		f(nil)
	}
}

// NewMustViewer TODO
func NewMustViewer(m *Map) *MustViewer {
	return &MustViewer{m}
}

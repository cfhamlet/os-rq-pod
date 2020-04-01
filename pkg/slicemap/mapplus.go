package slicemap

import "sync"

// MapPlus TODO
type MapPlus struct {
	*Map
	cLock *sync.RWMutex
}

// GetOrAdd TODO
func (mp *MapPlus) GetOrAdd(id uint64, f func(Item) Item) {
	mp.RLock()
	defer mp.RUnlock()

	item := mp.get(id)
	if item == nil {
		mp.cLock.Lock()
		defer mp.cLock.Unlock()
		item := mp.get(id)
		new := f(item)
		if item == nil {
			if new != nil {
				mp.add(new)
			}
		}
	} else {
		f(item)
	}
}

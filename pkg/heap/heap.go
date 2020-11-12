package heap

import (
	cheap "container/heap"
	"sync"
)

type HeapLess func(a interface{}, b interface{}) bool

type heap struct{
	array []interface{}
	less HeapLess
}

// Len TODO
func (h heap) Len() int { 
	return len(h.array) 
}

// Less TODO
func (h heap) Less(i, j int) bool {
	return h.less(h.array[i], h.array[j])
}

// Swap TODO
func (h heap) Swap(i, j int){
	
	h.array[i], h.array[j] = h.array[j], h.array[i]
}

// Push TODO
func (h *heap) Push(x interface{}) {
	h.array = append(h.array, x)
}

// Pop TODO
func (h *heap) Pop() interface{} {
	old := h.array
	n := len(old)
	x := old[n-1]
	h.array = old[0 : n-1]
	return x
}

// Top TODO
func (h heap) Top() interface{} {

	if len(h.array) == 0 {
		return nil
	}
	return h.array[0]
}

// Heap is a heap of int
type Heap struct{
	heap heap
	*sync.RWMutex
}

// Len TODO
func (h Heap) Len() int { 
	h.RLock()
	defer h.RUnlock()
	return h.heap.Len() 
}

// Push TODO
func (h *Heap) Push(x interface{}) {
	h.Lock()
	defer h.Unlock()
	cheap.Push(&h.heap, x)
}

// Pop TODO
func (h *Heap) Pop() interface{} {
	h.Lock()
	defer h.Unlock()
	if h.heap.Len() == 0 {
		return nil
	}
	return cheap.Pop(&h.heap)
}

// Top TODO
func (h Heap) Top() interface{} {
	h.RLock()
	defer h.RUnlock()
	return h.heap.Top()
}

// NewHeap TODO
func NewHeap(f HeapLess) *Heap{
	return &Heap{heap{make([]interface{}, 0), f}, &sync.RWMutex{}}
}

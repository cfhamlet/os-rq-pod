package utils

import "sync"

// BulkLock TODO
type BulkLock struct {
	lockers []*sync.RWMutex
}

// NewBulkLock TODO
func NewBulkLock(size int) *BulkLock {
	lockers := []*sync.RWMutex{}
	for i := 0; i < size; i++ {
		lockers = append(lockers, &sync.RWMutex{})
	}
	return &BulkLock{lockers}
}

func (lock *BulkLock) locker(i uint64) *sync.RWMutex {
	return lock.lockers[int(i>>32)%len(lock.lockers)]
}

// Lock TODO
func (lock *BulkLock) Lock(i uint64) {
	lock.locker(i).Lock()
}

// Unlock TODO
func (lock *BulkLock) Unlock(i uint64) {
	lock.locker(i).Unlock()
}

// RLock TODO
func (lock *BulkLock) RLock(i uint64) {
	lock.locker(i).RLock()
}

// RUnlock TODO
func (lock *BulkLock) RUnlock(i uint64) {
	lock.locker(i).RUnlock()
}
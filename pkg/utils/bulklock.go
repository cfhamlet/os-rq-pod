package utils

// BulkLock TODO
type BulkLock struct {
	lockers []*StageLock
}

// NewBulkLock TODO
func NewBulkLock(size int) *BulkLock {
	lockers := []*StageLock{}
	for i := 0; i < size; i++ {
		lockers = append(lockers, &StageLock{})
	}
	return &BulkLock{lockers}
}

func (lock *BulkLock) locker(i uint64) *StageLock {
	return lock.lockers[int(i>>32)%len(lock.lockers)]
}

// Lock TODO
func (lock *BulkLock) Lock(i uint64) {
	lock.locker(i).LLlock()
}

// Unlock TODO
func (lock *BulkLock) Unlock(i uint64) {
	lock.locker(i).LLunlock()
}

// RLock TODO
func (lock *BulkLock) RLock(i uint64) {
	lock.locker(i).RRLock()
}

// RUnlock TODO
func (lock *BulkLock) RUnlock(i uint64) {
	lock.locker(i).RRunlock()
}

// RR2RL TODO
func (lock *BulkLock) RR2RL(i uint64) {
	lock.locker(i).RR2RL()
}

// RL2RR TODO
func (lock *BulkLock) RL2RR(i uint64) {
	lock.locker(i).RL2RR()
}

// BatchRR2RL TODO
func (lock *BulkLock) BatchRR2RL(array []uint64) {
	t := map[*StageLock]struct{}{}
	for i := range array{
		t[lock.locker(array[i])]=struct{}{}
		if _, ok:=t[lock.locker(array[i])];!ok{
			l:=lock.locker(array[i])
			t[l]=struct{}{}
			l.RR2RL()
		}
	}
}

// BatchRL2RR TODO
func (lock *BulkLock) BatchRL2RR(array []uint64) {
	t := map[*StageLock]struct{}{}
	for i := range array{
		t[lock.locker(array[i])]=struct{}{}
		if _, ok:=t[lock.locker(array[i])];!ok{
			l:=lock.locker(array[i])
			t[l]=struct{}{}
			l.RL2RR()
		}
	}
}

// RLockAll TODO
func (lock *BulkLock) RLockAll(i uint64) {
	for locker := range lock.lockers{
		lock.lockers[locker].RRLock()
	}
}

// RUnlockAll TODO
func (lock *BulkLock) RUnlockAll(i uint64) {
	for locker := range lock.lockers{
		lock.lockers[locker].RRunlock()
	}
}

// BulkLockLen TODO
func (lock *BulkLock) BulkLockLen() uint64 {
	return uint64(len(lock.lockers))
}
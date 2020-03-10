package pod

// QueueOperation TODO
type QueueOperation interface {
	Operate(id QueueID) (Result, error)
}

// FuncWithQueueID TODO
type FuncWithQueueID func(QueueID) (Result, error)

// CallByQueueID TODO
type CallByQueueID struct {
	pod *Pod
	f   FuncWithQueueID
}

// Operate TODO
func (opt *CallByQueueID) Operate(id QueueID) (result Result, err error) {
	return opt.f(id)
}

// NewCallByQueueIDOperation TODO
func NewCallByQueueIDOperation(pod *Pod, f FuncWithQueueID) *CallByQueueID {
	return &CallByQueueID{pod, f}
}

// ViewQueue TODO
type ViewQueue struct {
	R     *Pod
	Start int64
	End   int64
}

// Operate TODO
func (opt *ViewQueue) Operate(id QueueID) (Result, error) {
	return opt.R.ViewQueue(id, opt.Start, opt.End)
}

// NewViewQueueOperation TODO
func NewViewQueueOperation(pod *Pod, start, end int64) *ViewQueue {
	return &ViewQueue{pod, start, end}
}

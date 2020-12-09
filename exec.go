package commons

import (
	"sync"
)

type Executor = func(int, ExecutionContext) error

type Verify = func(interface{}) bool

type ExecutionContext interface {
	End(data interface{})
	Ended() bool
}

func Run(exec Executor, verify Verify, n int) (chan interface{}, chan error) {
	r := newExecutionContext(verify)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer func() {
				// recover writing to a closed channel
				if r := recover(); r != nil {
					return
				}
			}()
			r.checkError(exec(i, r))
		}(i)
	}
	return r.out, r.err
}

type executionContext struct {
	m      *sync.Mutex
	stopped bool

	verify Verify

	out chan interface{}
	err chan error
}

func newExecutionContext(verify Verify) *executionContext {
	var m sync.Mutex
	sm := executionContext{&m, false, verify, make(chan interface{}, 1),make(chan error, 1)}
	return &sm
}

func (r *executionContext) End(data interface{}) {
	r.m.Lock()
	defer r.m.Unlock()

	r.stopped = r.verify(data)
	if r.stopped {
		r.out <- data
	}
}

func (r *executionContext) Ended() bool {
	r.m.Lock()
	defer r.m.Unlock()

	return r.stopped
}

func (r *executionContext) checkError(e error) {
	r.m.Lock()
	defer r.m.Unlock()

	if e != nil {
		r.stopped = true
		r.err <- e
	}
}
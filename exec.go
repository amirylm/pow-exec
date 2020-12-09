package commons

import (
	"sync"
)

// Execute is taking care of the actual execution
type Execute = func(int, ExecutionContext) error

// Verify checks the result of the execution
type Verify = func(interface{}) bool

// ExecutionContext is an interface for managing the flow from multiple go-routines
type ExecutionContext interface {
	// End is called when an executor is finished
	// will verify the output data
	// won't end execution unless the output is verified
	End(data interface{})
	// Ended is a function that checks whether the execution is done
	Ended() bool
}

// Run executes some task, with the given exec and verify, using n go-routines
func Run(exec Execute, verify Verify, n int) (chan interface{}, chan error) {
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

// executionContext is an implementation of ExecutionContext
type executionContext struct {
	m       *sync.Mutex
	stopped bool

	verify Verify

	out chan interface{}
	err chan error
}

func newExecutionContext(verify Verify) *executionContext {
	var m sync.Mutex
	sm := executionContext{&m, false, verify, make(chan interface{}, 1), make(chan error, 1)}
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

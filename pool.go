package routines

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"

	"go.uber.org/atomic"
)

var errPoolClosed = errors.New("the pool has closed")

type callbackFn func(errMsg, stack string)

type Pool struct {
	numWorkers int
	workers    []*worker
	processor  processor

	closed atomic.Bool
	mu     sync.Mutex
	wg     sync.WaitGroup

	panicCallbackFn callbackFn
	requestCh       chan request
	respawnCh       chan int
	shutdownCh      chan struct{}
}

type response struct {
	ret interface{}
	err error
}

type request struct {
	ctx        context.Context
	input      interface{}
	responseCh chan response
}

type processor interface {
	Process(ctx context.Context, input interface{}) (interface{}, error)
}

// NewPool would create a routine pool
func NewPool(numWorkers int, proc processor) (*Pool, error) {
	if numWorkers <= 0 {
		return nil, errors.New("thread number should be > 0")
	}

	p := &Pool{
		numWorkers: numWorkers,
		workers:    make([]*worker, numWorkers),
		processor:  proc,

		requestCh:  make(chan request),
		respawnCh:  make(chan int),
		shutdownCh: make(chan struct{}),
	}
	p.closed.Store(true)
	return p, nil
}

// Start the routine pool and spawn worker routines
func (p *Pool) Start() error {
	if !p.closed.CAS(true, false) {
		return errors.New("the pool was running")
	}

	p.spawnWorkers()

	p.wg.Add(1)
	go p.loop()
	return nil
}

func (p *Pool) SetPanicCallback(fn callbackFn) {
	p.mu.Lock()
	p.panicCallbackFn = fn
	p.mu.Unlock()
}

func (p *Pool) loop() {
	defer func() {
		p.wg.Done()
		if err := recover(); err != nil {
			log.Panicf("Found the loop panic, err %v with stack: %s", err, string(debug.Stack()))
		}
	}()

	for {
		select {
		case index := <-p.respawnCh:
			select {
			case <-p.shutdownCh:
				return
			default:
			}

			p.wg.Add(1)
			go p.spawnWorker(index)
		case <-p.shutdownCh:
			return
		}
	}
}

// Wait would block and wait for routines exited
func (p *Pool) Wait() {
	p.wg.Wait()
	// it's ok that leaves the channel forever open here to make the data race detector happy,
	// GC would help us to collect it when never used again.
	// close(p.respawnCh)
	// close(p.requestCh)
}

func (p *Pool) spawnWorker(index int) {
	defer func() {
		p.wg.Done()

		if err := recover(); err != nil {
			if fn := p.panicCallbackFn; fn != nil {
				fn(fmt.Sprintf("%v", err), string(debug.Stack()))
			}
			select {
			case p.respawnCh <- index:
			// do nothing, just wait for respawning a new one
			case <-p.shutdownCh:
				return
			}
		}
	}()

	p.mu.Lock()
	worker := newWorker(p)
	p.workers[index] = worker
	p.mu.Unlock()

	worker.start()
}

func (p *Pool) spawnWorkers() {
	p.wg.Add(p.numWorkers)
	for i := 0; i < p.numWorkers; i++ {
		go p.spawnWorker(i)
	}
}

// Process would pass the task to worker routines
func (p *Pool) Process(ctx context.Context, i interface{}) (interface{}, error) {
	if p.closed.Load() {
		return nil, errPoolClosed
	}
	responseCh := make(chan response)
	req := request{ctx: ctx, input: i, responseCh: responseCh}
	select {
	case p.requestCh <- req:
		rsp := <-req.responseCh
		close(responseCh)
		return rsp.ret, rsp.err
	case <-p.shutdownCh:
		return nil, errPoolClosed
	}
}

// AsyncProcess behaves like the Process except that
// don't wait for the response, and the user can use
// the future to fetch the response later.
func (p *Pool) AsyncProcess(ctx context.Context, i interface{}) (*Future, error) {
	if p.closed.Load() {
		return nil, errPoolClosed
	}

	future := newFuture()
	req := request{ctx: ctx, input: i, responseCh: future.responseCh}
	select {
	case p.requestCh <- req:
		return future, nil
	case <-p.shutdownCh:
		return nil, errPoolClosed
	}
}

// Stop would stop the pool and worker routines
func (p *Pool) Stop() error {
	if !p.closed.CAS(false, true) {
		return errors.New("the pool has closed")
	}
	close(p.shutdownCh)

	p.mu.Lock()
	for i := 0; i < p.numWorkers; i++ {
		if p.workers[i] != nil { // pool may be stopped before spawning all routines
			p.workers[i].stop()
		}
	}
	p.mu.Unlock()
	return nil
}

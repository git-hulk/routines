package routines

import (
	"context"
	"sync"
)

type Future struct {
	val interface{}
	err error

	responseCh chan response
	rwmu       sync.RWMutex
	ready      bool
}

func newFuture() *Future {
	return &Future{responseCh: make(chan response, 1)}
}

func (future *Future) setResponse(rsp *response) {
	future.ready = true
	future.val = rsp.ret
	future.err = rsp.err
}

// Get was used to fetch the response from future,
// it would be blocked until the response was ready
func (future *Future) Get(ctx context.Context) (interface{}, error) {
	future.rwmu.RLock()
	if future.ready {
		future.rwmu.RUnlock()
		return future.val, future.err
	}
	future.rwmu.RUnlock()

	future.rwmu.Lock()
	defer future.rwmu.Unlock()

	if future.ready {
		return future.val, future.err
	}
	select {
	case rsp := <-future.responseCh:
		future.setResponse(&rsp)
		return rsp.ret, rsp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

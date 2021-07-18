package routines

import (
	"fmt"
)

type worker struct {
	processor processor
	requestCh chan request
	owner     *Pool

	shutdown chan struct{}
}

func newWorker(owner *Pool) *worker {
	return &worker{
		owner:     owner,
		processor: owner.processor,
		requestCh: owner.requestCh,

		shutdown: make(chan struct{}),
	}
}

func (w *worker) start() {
	for {
		select {
		case request := <-w.requestCh:
			w.process(&request)
		case <-w.shutdown:
			return
		case <-w.owner.shutdownCh:
			return
		}
	}
}

func (w *worker) process(req *request) {
	defer func() {
		if err := recover(); err != nil {
			// the processor would block and wait the response,
			// so we should notify the processor even the worker was panic
			if req.responseCh != nil {
				req.responseCh <- response{ret: nil, err: fmt.Errorf("found panic: %v", err)}
			}
			// raise the panic to pool and it would respawn a new one
			panic(err)
		}
	}()

	rsp, err := w.processor.Process(req.ctx, req.input)
	if req.responseCh != nil {
		req.responseCh <- response{ret: rsp, err: err}
	}

}

func (w *worker) stop() {
	close(w.shutdown)
}

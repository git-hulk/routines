package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/git-hulk/routines"

	"go.uber.org/atomic"
)

type incrProcessor struct{}

func (p *incrProcessor) Process(ctx context.Context, i interface{}) (interface{}, error) {
	v := i.(*atomic.Int32)
	v.Inc()
	return v, nil
}

func main() {
	// the pool would spawn 3 routines to process jobs and respawn
	// a new one when panic
	pool, err := routines.NewPool(3, &incrProcessor{})
	if err != nil {
		panic(err)
	}
	// Start the routine pool
	_ = pool.Start()

	// Send the counter task to process in other routines
	var wg sync.WaitGroup
	v := atomic.NewInt32(0)
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = pool.Process(context.Background(), v)
			}
		}()
	}
	wg.Wait()

	fmt.Println("Total: ", v.Load())
	// Stop the routine pool and wait for all routines exited
	_ = pool.Stop()
	pool.Wait()
}

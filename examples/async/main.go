package main

import (
	"context"
	"fmt"

	"github.com/git-hulk/routines"

	"go.uber.org/atomic"
)

type incrProcessor struct{}

func (p *incrProcessor) Process(ctx context.Context, i interface{}) (interface{}, error) {
	v := i.(*atomic.Int32)
	v.Inc()
	return v.Load(), nil
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
	futures := make([]*routines.Future, 10)
	v := atomic.NewInt32(0)
	for i := 0; i < 10; i++ {
		futures[i], _ = pool.AsyncProcess(context.Background(), v)
	}
	for _, future := range futures {
		val, err := future.Get(context.Background())
		fmt.Println(val, err)
	}

	fmt.Println("Total: ", v.Load())
	// Stop the routine pool and wait for all routines exited
	_ = pool.Stop()
	pool.Wait()
}

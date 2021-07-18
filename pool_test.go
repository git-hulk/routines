package routines

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.uber.org/atomic"
)

type incrProcessor struct{}

func (p *incrProcessor) Process(ctx context.Context, i interface{}) (interface{}, error) {
	v := i.(*atomic.Int32)
	v.Inc()
	return v, nil
}

type panicProcessor struct{}

func (p *panicProcessor) Process(ctx context.Context, i interface{}) (interface{}, error) {
	v := i.(*atomic.Int32)
	if v.Inc()%3 == 0 {
		panic("expected to panic")
	}
	return v, nil
}

func TestNewPool(t *testing.T) {
	for _, proc := range []processor{&incrProcessor{}, &panicProcessor{}} {
		pool, err := NewPool(3, proc)
		require.Nil(t, err)
		_ = pool.Start()
		time.Sleep(10 * time.Millisecond)

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
		_ = pool.Stop()
		pool.Wait()
		require.Equal(t, int32(100), v.Load())
	}
}

type slowProcessor struct{}

func (*slowProcessor) Process(ctx context.Context, i interface{}) (interface{}, error) {
	time.Sleep(time.Millisecond * 500)
	return nil, nil
}

func TestClosePool(t *testing.T) {
	pool, err := NewPool(3, &slowProcessor{})
	require.Nil(t, err)
	_ = pool.Start()
	cnt := 10
	finishedGoroutineCount := atomic.NewInt32(0)
	for i := 0; i < cnt; i++ {
		go func() {
			_, _ = pool.Process(context.Background(), nil)
			finishedGoroutineCount.Inc()
		}()
	}
	_ = pool.Stop()
	pool.Wait()
	time.Sleep(time.Second)
	require.Equal(t, int32(cnt), finishedGoroutineCount.Load())
}

type asyncProcessor struct{}

func (*asyncProcessor) Process(ctx context.Context, i interface{}) (interface{}, error) {
	v := i.(*atomic.Int32)
	return v.Inc(), nil
}

func TestAsyncProcess_Future(t *testing.T) {
	pool, err := NewPool(3, &asyncProcessor{})
	require.Nil(t, err)

	_ = pool.Start()
	n := 10
	cnt := atomic.NewInt32(0)
	for i := 0; i < n; i++ {
		future, err := pool.AsyncProcess(context.Background(), cnt)
		require.Nil(t, err)
		v, _ := future.Get(context.Background())
		require.Equal(t, int32(i+1), v)

		// make sure reenter the `Get` function call was ok
		v, _ = future.Get(context.Background())
		require.Equal(t, int32(i+1), v)
	}
	_ = pool.Stop()
	pool.Wait()
}

func TestAsyncProcess_No_Future(t *testing.T) {
	pool, err := NewPool(3, &asyncProcessor{})
	require.Nil(t, err)

	_ = pool.Start()
	n := 10
	cnt := atomic.NewInt32(0)
	for i := 0; i < n; i++ {
		_, err := pool.AsyncProcess(context.Background(), cnt)
		require.Nil(t, err)
	}

	time.Sleep(10 * time.Millisecond)
	require.Equal(t, int32(10), cnt.Load())
	_ = pool.Stop()
	pool.Wait()
}

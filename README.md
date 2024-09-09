## Routines
[![Go Report Card](https://goreportcard.com/badge/github.com/git-hulk/routines)](https://goreportcard.com/report/github.com/git-hulk/routines) [![GitHub release](https://img.shields.io/github/tag/git-hulk/routines.svg?label=release)](https://github.com/git-hulk/routines/releases) [![GitHub release date](https://img.shields.io/github/release-date/git-hulk/routines.svg)](https://github.com/git-hulk/routines/releases) [![LICENSE](https://img.shields.io/github/license/git-hulk/routines.svg)](https://github.com/git-hulk/routines/blob/master/LICENSE) [![GoDoc](https://img.shields.io/badge/Godoc-reference-blue.svg)](https://godoc.org/github.com/git-hulk/routines) [![codecov](https://codecov.io/gh/git-hulk/routines/branch/master/graph/badge.svg?token=1O559OI069)](https://codecov.io/gh/git-hulk/routines) 

Routines was a fixed number thread pool to process the user task, and it would respawn a corresponding new thread when panic. It supports the sync process mode that would wait for the response, as well as the async mode, which would not wait for the response. The most commonly used scenario was to control the max concurrency when processing jobs.

## Example 1: Process Tasks
For the full example, see: [examples/counter](examples/counter/main.go)
 
```Go
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
```

## Example 2: Async Process Tasks

For the full example, see: [examples/counter](examples/async/main.go)

```Go
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

```

## License
Routines is under the MIT license. See the [LICENSE](https://github.com/git-hulk/routines/blob/master/LICENSE) file for details.

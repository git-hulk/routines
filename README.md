## Routines

Routines was a fixed number thread pool to process the user task, and it would respawn a corresponding new thread when panic. It supports the sync process mode that would wait for the response, as well as the sync mode, which would not wait for the response. The most commonly used scenario was to control the max concurrency when processing jobs.

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

## Example 1: Async Process Tasks

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

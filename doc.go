// Package graterm provides capabilities to create a [Terminator] instance, register ordered termination Hooks,
// and block application execution until one of the registered [os.Signal] events occurs.
//
// Termination hooks registered with the same [Order] will be executed concurrently.
//
// It is possible to set individual timeouts for each registered termination hook and global termination timeout for the whole application.
//
// Example code for generic application components:
//
// 	func main() {
// 		// create new Terminator instance:
// 		terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
//
// 		// Register some hooks:
// 		terminator.WithOrder(1).
// 			WithName("HOOK#1").
// 			Register(1*time.Second, func(ctx context.Context) {
// 				log.Println("terminating HOOK#1...")
// 				defer log.Println("...HOOK#1 terminated")
// 			})
//
// 		terminator.WithOrder(1).
// 			WithName("HOOK#2").
// 			Register(1*time.Second, func(ctx context.Context) {
// 				log.Println("terminating HOOK#2...")
// 				defer log.Println("...HOOK#2 terminated")
// 			})
//
// 		terminator.WithOrder(2).
// 			WithName("HOOK#3").
// 			Register(1*time.Second, func(ctx context.Context) {
// 				log.Println("terminating HOOK#3...")
// 				defer log.Println("...HOOK#3 terminated")
//
// 				const sleepTime = 3 * time.Second
// 				select {
// 				case <-time.After(sleepTime):
// 					log.Printf("HOOK#3 sleep time %v is over\n", sleepTime)
// 				case <-ctx.Done():
// 					log.Printf("HOOK#3 Context is Done because of: %+v\n", ctx.Err())
// 				}
// 			})
//
// 		// Wait for os.Signal to occur, then terminate application with maximum timeout of 20 seconds:
// 		if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
// 			log.Printf("graceful termination period was timed out: %+v", err)
// 		}
// 	}
//
// Example code for HTTP server integration:
//
// 	func main() {
// 		terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
//
// 		// .....................
//
// 		httpServer := &http.Server{
// 			Addr:    ":8080",
// 			Handler: http.DefaultServeMux,
// 		}
// 		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 			fmt.Fprintf(w, "hello, world!")
// 		})
//
// 		go func() {
// 			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
// 				log.Printf("terminated HTTP Server: %+v\n", err)
// 			}
// 		}()
//
// 		terminator.WithOrder(1).
// 			WithName("HTTPServer").
// 			Register(10*time.Second, func(ctx context.Context) {
// 				if err := httpServer.Shutdown(ctx); err != nil {
// 					log.Printf("shutdown HTTP Server: %+v\n", err)
// 				}
// 			})
//
// 		if err := terminator.Wait(appCtx, 30*time.Second); err != nil {
// 			log.Printf("graceful termination period is timed out: %+v\n", err)
// 		}
// 	}
package graterm

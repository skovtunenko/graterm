/*
Package graterm provides capabilities to create a [Terminator] instance, register ordered termination Hooks,
and block application execution until one of the registered [os.Signal] events occurs.

Termination hooks registered with the same [Order] will be executed concurrently.

It is possible to set individual timeouts for each registered termination hook and global termination timeout for the whole application.

Optionally a Hook may have a name (using Hook.WithName). It might be handy only if the Logger injected into Terminator instance to
log internal termination lifecycle events.

# Examples

Example code for generic application components:

	func main() {
		// Define termination Orders:
		const (
			HTTPServerTerminationOrder graterm.Order = 1
			DBTerminationOrder         graterm.Order = 2
		)

		// create new Terminator instance:
		terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		terminator.SetLogger(log.Default()) // Optional step

		// Register HTTP Server termination hook:
		terminator.WithOrder(HTTPServerTerminationOrder).
			WithName("HTTP Server"). // setting a Name is optional and will be useful only if logger instance provided
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating HTTP Server...")
				defer log.Println("...HTTP Server terminated")
			})

		// Register nameless DB termination hook:
		terminator.WithOrder(DBTerminationOrder).
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating Database...")
				defer log.Println("...Database terminated")

				const sleepTime = 3 * time.Second
				select {
				case <-time.After(sleepTime):
					log.Printf("Database termination sleep time %v is over\n", sleepTime)
				case <-ctx.Done():
					log.Printf("Database termination Context is Done because of: %+v\n", ctx.Err())
				}
			})

		// Wait for os.Signal to occur, then terminate application with maximum timeout of 20 seconds:
		if err := terminator.Wait(appCtx, 20 * time.Second); err != nil {
			log.Printf("graceful termination period was timed out: %+v", err)
		}
	}

Example code for HTTP server integration:

	func main() {
		// Define Order for HTTP Server termination:
		const HTTPServerTerminationOrder graterm.Order = 1

		// create new Terminator instance:
		terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		terminator.SetLogger(log.Default()) // Optional step

		// Create an HTTP Server and add one simple handler into it:
		httpServer := &http.Server{
			Addr:              ":8080",
			Handler:           http.DefaultServeMux,
		}
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "hello, world!")
		})

		// Start HTTP server in a separate goroutine:
		go func() {
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("terminated HTTP Server: %+v\n", err)
			}
		}()

		// Register HTTP Server termination hook:
		terminator.WithOrder(HTTPServerTerminationOrder).
			WithName("HTTPServer"). // setting a Name is optional and will be useful only if logger instance provided
			Register(10*time.Second, func(ctx context.Context) {
				if err := httpServer.Shutdown(ctx); err != nil {
					log.Printf("shutdown HTTP Server: %+v\n", err)
				}
			})

		// Wait for os.Signal to occur, then terminate application with maximum timeout of 30 seconds:
		if err := terminator.Wait(appCtx, 30 * time.Second); err != nil {
			log.Printf("graceful termination period is timed out: %+v\n", err)
		}
	}
*/
package graterm

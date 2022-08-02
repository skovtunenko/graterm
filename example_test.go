package graterm_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/skovtunenko/graterm"
)

func ExampleNewWithSignals() {
	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// register hooks...

	// Wire other components ...

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
	if err := terminator.Wait(appCtx, 40*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

func ExampleTerminator_SetLogger() {
	// create new Terminator instance:
	terminator, _ := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Set custom logger implementation instead of default NOOP one:
	terminator.SetLogger(log.Default())
}

func ExampleTerminator_Wait() {
	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// register hooks...

	// Wire other components ...

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
	if err := terminator.Wait(appCtx, 40*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

func ExampleTerminator_WithOrder_1() {
	// Define Orders:
	const (
		HTTPServerTerminationOrder graterm.Order = 1
		MessagingTerminationOrder  graterm.Order = 1
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

	// Register nameless Messaging termination hook:
	terminator.WithOrder(MessagingTerminationOrder).
		Register(1*time.Second, func(ctx context.Context) {
			log.Println("terminating Messaging...")
			defer log.Println("...Messaging terminated")
		})

	// Register Database termination hook:
	terminator.WithOrder(DBTerminationOrder).
		WithName("DB"). // setting a Name is optional and will be useful only if logger instance provided
		Register(1*time.Second, func(ctx context.Context) {
			log.Println("terminating DB...")
			defer log.Println("...DB terminated")

			const sleepTime = 3 * time.Second
			select {
			case <-time.After(sleepTime):
				log.Printf("DB termination sleep time %v is over\n", sleepTime)
			case <-ctx.Done():
				log.Printf("DB termination Context is Done because of: %+v\n", ctx.Err())
			}
		})

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 20 seconds:
	if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

func ExampleTerminator_WithOrder_2() {
	// Define Order for HTTP Server termination:
	const HTTPServerTerminationOrder graterm.Order = 1

	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	terminator.SetLogger(log.Default()) // Optional step

	// Create an HTTP Server and add one simple handler into it:
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
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
	if err := terminator.Wait(appCtx, 30*time.Second); err != nil {
		log.Printf("graceful termination period is timed out: %+v\n", err)
	}
}

func ExampleHook_Register() {
	// Define Order:
	const (
		HTTPServerTerminationOrder graterm.Order = 1
	)

	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Register some hooks:
	terminator.WithOrder(HTTPServerTerminationOrder).
		Register(1*time.Second, func(ctx context.Context) {
			log.Println("terminating HTTP Server...")
			defer log.Println("...HTTP Server terminated")
		})

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 20 seconds:
	if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

func ExampleHook_WithName() {
	// Define Order:
	const (
		HTTPServerTerminationOrder graterm.Order = 1
	)

	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	terminator.SetLogger(log.Default()) // Optional step

	// Register some hooks:
	terminator.WithOrder(HTTPServerTerminationOrder).
		WithName("HTTP Server"). // Define (optional) Hook name
		Register(1*time.Second, func(ctx context.Context) {
			log.Println("terminating HTTP Server...")
			defer log.Println("...HTTP Server terminated")
		})

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 20 seconds:
	if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

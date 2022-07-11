package graterm_test

import (
	"context"
	"log"
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

func ExampleTerminator_WithOrder() {
	// Define Orders:
	const (
		HTTPServerTerminationOrder graterm.Order = 1
		MessagingTerminationOrder  graterm.Order = 1
		DBTerminationOrder         graterm.Order = 2
	)

	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Register some hooks:
	{
		terminator.WithOrder(HTTPServerTerminationOrder).
			WithName("HTTP Server").
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating HTTP Server...")
				defer log.Println("...HTTP Server terminated")
			})

		terminator.WithOrder(MessagingTerminationOrder).
			WithName("Messaging").
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating Messaging...")
				defer log.Println("...Messaging terminated")
			})

		terminator.WithOrder(DBTerminationOrder).
			WithName("DB").
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
}

func ExampleHook_Register() {
	// Define Orders:
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
	// Define Orders:
	const (
		HTTPServerTerminationOrder graterm.Order = 1
	)

	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

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

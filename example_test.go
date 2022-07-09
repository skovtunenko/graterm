package graterm_test

import (
	"context"
	"github.com/skovtunenko/graterm"
	"log"
	"syscall"
	"time"
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
	// create new Terminator instance:
	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Register some hooks:
	{
		terminator.WithOrder(1).
			WithName("HOOK#1").
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating HOOK#1...")
				defer log.Println("...HOOK#1 terminated")
			})

		terminator.WithOrder(1).
			WithName("HOOK#2").
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating HOOK#2...")
				defer log.Println("...HOOK#2 terminated")
			})

		terminator.WithOrder(2).
			WithName("HOOK#3").
			Register(1*time.Second, func(ctx context.Context) {
				log.Println("terminating HOOK#3...")
				defer log.Println("...HOOK#3 terminated")

				const sleepTime = 3 * time.Second
				select {
				case <-time.After(sleepTime):
					log.Printf("HOOK#3 sleep time %v is over\n", sleepTime)
				case <-ctx.Done():
					log.Printf("HOOK#3 Context is Done because of: %+v\n", ctx.Err())
				}
			})

		// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
		if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
			log.Printf("graceful termination period was timed out: %+v", err)
		}
	}
}

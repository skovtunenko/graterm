package graterm_test

import (
	"context"
	"github.com/skovtunenko/graterm"
	"log"
	"syscall"
	"time"
)

func ExampleNewWithSignals() {
	// create new Stopper instance:
	stopper, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// register hooks...

	// Wire other components ...

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
	if err := stopper.Wait(appCtx, 40*time.Second); err != nil {
		log.Printf("graceful termination period is timed out: %+v", err)
	}
}

func ExampleStopper_Wait() {
	// create new Stopper instance:
	stopper, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// register hooks...

	// Wire other components ...

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
	if err := stopper.Wait(appCtx, 40*time.Second); err != nil {
		log.Printf("graceful termination period is timed out: %+v", err)
	}
}

func ExampleStopper_Register() {
	// create new Stopper instance:
	stopper, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Register some hooks:
	{
		stopper.Register(1, "HOOK#1", 1*time.Second, func(ctx context.Context) {
			log.Println("terminating HOOK#1...")
			defer log.Println("...HOOK#1 terminated")
		})

		stopper.Register(1, "HOOK#2", 1*time.Second, func(ctx context.Context) {
			log.Println("terminating HOOK#2...")
			defer log.Println("...HOOK#2 terminated")
		})

		stopper.Register(2, "HOOK#3", 1*time.Second, func(ctx context.Context) {
			log.Println("terminating HOOK#3...")
			defer log.Println("...HOOK#3 terminated")

			sleepTime := 3 * time.Second
			t := time.NewTimer(sleepTime)
			select {
			case <-t.C:
				log.Printf("HOOK#3 sleep time %v is over\n", sleepTime)
			case <-ctx.Done():
				log.Printf("HOOK#3 Context is Done because of: %+v\n", ctx.Err())
			}
		})

		// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
		if err := stopper.Wait(appCtx, 20*time.Second); err != nil {
			log.Printf("graceful termination period is timed out: %+v", err)
		}
	}
}

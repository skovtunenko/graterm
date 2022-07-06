package graterm_test

import (
	"context"
	"fmt"
	"github.com/skovtunenko/graterm"
	"log"
	"net/http"
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
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

func ExampleStopper_SetLogger() {
	// create new Stopper instance:
	stopper, _ := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Set custom logger implementation instead of default NOOP one:
	stopper.SetLogger(log.Default())
}

func ExampleStopper_Wait() {
	// create new Stopper instance:
	stopper, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// register hooks...

	// Wire other components ...

	// Wait for os.Signal to occur, then terminate application with maximum timeout of 40 seconds:
	if err := stopper.Wait(appCtx, 40*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
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
			log.Printf("graceful termination period was timed out: %+v", err)
		}
	}
}

func ExampleStopper_ServerAsyncStarterFunc() {
	stopper, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Register some hooks:
	stopper.Register(3, "HOOK", 1*time.Second, func(ctx context.Context) {
		log.Println("terminating HOOK...")
		defer log.Println("...HOOK terminated")
	})

	const hostPort = ":8080"
	server := &http.Server{
		Addr:    hostPort,
		Handler: http.DefaultServeMux,
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, world!\n")
	})

	asyncStarterFunc := stopper.ServerAsyncStarterFunc(appCtx, server)
	// ... here it might be some actions before we need to start the server.....
	asyncStarterFunc()
	log.Printf("application started on: %q\n", hostPort)

	if err := stopper.Wait(appCtx, 40*time.Second); err != nil {
		log.Printf("graceful termination period was timed out: %+v", err)
	}
}

# graterm

![Lint](https://github.com/skovtunenko/graterm/actions/workflows/golangci-lint.yml/badge.svg?branch=master)
![Tests](https://github.com/skovtunenko/graterm/actions/workflows/test.yml/badge.svg?branch=master)
[![codecov](https://codecov.io/gh/skovtunenko/graterm/branch/master/graph/badge.svg)](https://codecov.io/gh/skovtunenko/graterm)
[![Go Report Card](https://goreportcard.com/badge/github.com/skovtunenko/graterm)](https://goreportcard.com/report/github.com/skovtunenko/graterm)
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/skovtunenko/graterm/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/skovtunenko/graterm?status.svg)](https://godoc.org/github.com/skovtunenko/graterm)
[![Release](https://img.shields.io/github/release/skovtunenko/graterm.svg?style=flat-square)](https://github.com/skovtunenko/graterm/releases/latest)

Provides primitives to perform ordered **GRA**ceful **TERM**ination (aka shutdown) in Go application.

# Description

Library provides fluent methods to register ordered application termination (aka shutdown) hooks,
and block the main goroutine until the registered os.Signal will occur. 

Termination hooks registered with the same order will be executed concurrently.

It is possible to set individual timeouts for each registered termination hook and global termination timeout for the whole application.

# Example

Each public function has example attached to it. Here is the simple one:

```go
package main

import (
	"context"
	"log"
	"syscall"
	"time"
	
	"github.com/skovtunenko/graterm"
)

func main() {
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

		// Wait for os.Signal to occur, then terminate application with maximum timeout of 20 seconds:
		if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
			log.Printf("graceful termination period was timed out: %+v", err)
		}
	}
}
```

Integration with HTTP server
-----------

The library doesn't have out of the box support to start/terminate the HTTP server, but that's easy to handle:

```go
terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)

// .....................

httpServer := &http.Server{
    Addr:    hostPort,
    Handler: http.DefaultServeMux,
}
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "hello, world!")
})

go func() {
    if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.Printf("terminated HTTP Server: %+v\n", err)
    }
}()

terminator.WithOrder(HTTPServerOrder).
    WithName("HTTPServer").
    Register(httpServerTerminationTimeout, func(ctx context.Context) {
        if err := httpServer.Shutdown(ctx); err != nil { 
            log.Printf("shutdown HTTP Server: %+v\n", err)
        }
    })

if err := terminator.Wait(appCtx, globalTerminationTimeout); err != nil {
    log.Printf("graceful termination period is timed out: %+v\n", err)
}
```

The full-fledged example located here: https://github.com/skovtunenko/graterm/blob/main/internal/example/example.go

Testing
-----------
Unit-tests with code coverage:
```bash
make test
```

Run linter:
```bash
make code-quality
```

LICENSE
-----------
MIT

AUTHORS
-----------
Sergiy Kovtunenko <@skovtunenko>
Oleksandr Halushchak <ohalushchak@exadel.com>
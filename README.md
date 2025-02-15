# graterm

[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
![Lint](https://github.com/skovtunenko/graterm/actions/workflows/golangci-lint.yml/badge.svg?branch=main)
![Tests](https://github.com/skovtunenko/graterm/actions/workflows/test.yml/badge.svg?branch=main)
[![codecov](https://codecov.io/gh/skovtunenko/graterm/branch/main/graph/badge.svg)](https://codecov.io/gh/skovtunenko/graterm)
[![Go Report Card](https://goreportcard.com/badge/github.com/skovtunenko/graterm)](https://goreportcard.com/report/github.com/skovtunenko/graterm)
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/skovtunenko/graterm/blob/main/LICENSE)
[![GoDoc](https://godoc.org/github.com/skovtunenko/graterm?status.svg)](https://godoc.org/github.com/skovtunenko/graterm)
[![Release](https://img.shields.io/github/release/skovtunenko/graterm.svg?style=flat-square)](https://github.com/skovtunenko/graterm/releases/latest)

Provides primitives to perform ordered **GRA**ceful **TERM**ination (aka shutdown) in Go application.

# ⚡ ️️Description

Library provides fluent methods to register ordered application termination (aka shutdown) [hooks](https://pkg.go.dev/github.com/skovtunenko/graterm#Hook),
and block the main goroutine until the registered `os.Signal` will occur. 

Termination [hooks](https://pkg.go.dev/github.com/skovtunenko/graterm#Hook) registered with the 
same [Order](https://pkg.go.dev/github.com/skovtunenko/graterm#Order) will be executed concurrently.

It is possible to set individual timeouts for each registered termination [hook](https://pkg.go.dev/github.com/skovtunenko/graterm#Hook) 
and global termination timeout for the whole application.

# 🎯 Features

* Dependency only on a standard Go library (except tests).
* Component-agnostic (can be adapted to any 3rd party technology).
* Clean and tested code: 100% test coverage, including goroutine leak tests.
* Rich set of examples.

# ⚙️ Usage

Get the library:
```bash
go get -u github.com/skovtunenko/graterm
```

Import the library into the project:
```go
import (
    "github.com/skovtunenko/graterm"
)
```

Create a new instance of [Terminator](https://pkg.go.dev/github.com/skovtunenko/graterm#Terminator) and get an application context 
that will be cancelled when one of the registered `os.Signal`s will occur:
```go
// create new Terminator instance:
terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
terminator.SetLogger(log.Default()) // Optionally set the custom logger implementation instead of default NOOP one
```

Optionally define [Order](https://pkg.go.dev/github.com/skovtunenko/graterm#Order) of components to be terminated at the end:
```go
const (
    HTTPServerTerminationOrder graterm.Order = 1
    MessagingTerminationOrder  graterm.Order = 1
    DBTerminationOrder         graterm.Order = 2
    // ..........
)
```

Register some termination [Hooks](https://pkg.go.dev/github.com/skovtunenko/graterm#Hook) with priorities:
```go
terminator.WithOrder(HTTPServerTerminationOrder).
    WithName("HTTP Server"). // setting a Name is optional and will be useful only if logger instance provided
    Register(1*time.Second, func(ctx context.Context) {
        if err := httpServer.Shutdown(ctx); err != nil {
            log.Printf("shutdown HTTP Server: %+v\n", err)
        }
    })
```

Block main goroutine until the application receives one of the registered `os.Signal`s:
```go
if err := terminator.Wait(appCtx, 20*time.Second); err != nil {
    log.Printf("graceful termination period was timed out: %+v", err)
}
```

# 👀 Versioning

The library follows SemVer policy. With the release of **v1.0.0** the _public API is stable_. 

# 📚 Example

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
```

💡 Integration with HTTP server
-----------

The library doesn't have out of the box support to start/terminate the HTTP server, but that's easy to handle:

```go
package main

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
    if err := terminator.Wait(appCtx, 30*time.Second); err != nil {
        log.Printf("graceful termination period is timed out: %+v\n", err)
    }
}
```

The full-fledged example located here: [example.go](https://github.com/skovtunenko/graterm/blob/main/internal/example/example.go)

📖 Testing
-----------
Unit-tests with code coverage:
```bash
make test
```

Run linter:
```bash
make code-quality
```

⚠️ LICENSE
-----------
[MIT](https://github.com/skovtunenko/graterm/blob/main/LICENSE)

🕶️ AUTHORS
-----------

* [Sergiy Kovtunenko](https://github.com/skovtunenko)
* [Oleksandr Halushchak](ohalushchak@exadel.com)
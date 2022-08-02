package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/skovtunenko/graterm"
	"log"
	"net/http"
	"syscall"
	"time"
)

// Instructions:
// - run the application, reach out to http://localhost:8080/
// - terminate the application (CTRL+C)
// - investigate the log output

const hostPort = ":8080"

const (
	globalTerminationTimeout = 40 * time.Second

	httpServerTerminationTimeout  = 5 * time.Second
	httpServerControllerSleepTime = 15 * time.Second

	messagingTerminationTimeout = 1 * time.Second

	fastDBTerminationTimeout = 1 * time.Second

	slowDBTerminationTimeout = 3 * time.Second
	slowDBSleepTime          = 5 * time.Second // Termination of SlowDB can't be finished in time
)

const (
	HTTPServerTerminationOrder graterm.Order = 0
	MessagingTerminationOrder  graterm.Order = 1
	FastDBTerminationOrder     graterm.Order = 2
	SlowDBTerminationOrder     graterm.Order = 2
)

func main() {
	logger := log.Default()

	logger.Println("Application started...")

	terminator, appCtx := graterm.NewWithSignals(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	terminator.SetLogger(logger)

	// Wire application components:
	slowDB := NewSlowDB(terminator, logger)
	fastDB := NewFastDB(terminator, logger)
	messaging := NewMessaging(terminator, logger)
	srv := NewServer(terminator, logger)
	app := NewApplication(logger, srv, messaging, slowDB, fastDB)
	app.Run()

	if err := terminator.Wait(appCtx, globalTerminationTimeout); err != nil {
		logger.Printf("graceful termination period is timed out: %+v", err)
	}
	logger.Println("Application ended.")
}

type Application struct {
	Log       *log.Logger
	Server    *Server
	Messaging *Messaging
	SlowDB    *SlowDB
	FastDB    *FastDB
}

func NewApplication(log *log.Logger, server *Server, messaging *Messaging, slowDB *SlowDB, fastDB *FastDB) *Application {
	return &Application{Log: log, Server: server, Messaging: messaging, SlowDB: slowDB, FastDB: fastDB}
}

func (a *Application) Run() {
	a.SlowDB.Init()
	a.FastDB.Init()
	a.Messaging.Init()
	a.Server.Init()
}

type Server struct {
	logger     *log.Logger
	terminator *graterm.Terminator
}

func NewServer(terminator *graterm.Terminator, logger *log.Logger) *Server {
	return &Server{terminator: terminator, logger: logger}
}

func (s *Server) Init() {
	defer s.logger.Println("HTTP Server initialized")

	httpServer := &http.Server{
		ReadHeaderTimeout: 60 * time.Second, // fix for potential Slowloris Attack
		Addr:              hostPort,
		Handler:           http.DefaultServeMux,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		select {
		case <-time.After(httpServerControllerSleepTime): // Simulate long-running in-flight HTTP request processing
			s.logger.Println("HTTP Server controller finished")
		case <-ctx.Done():
			s.logger.Printf("HTTP Server controller interrupted because of: %+v\n", ctx.Err())
		}
		fmt.Fprintf(w, "hello, world!\nSlept for %v seconds\n", httpServerControllerSleepTime)
	})

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("terminated HTTP Server: %+v", err)
		}
	}()

	s.terminator.WithOrder(HTTPServerTerminationOrder).
		WithName("HTTPServer").
		Register(httpServerTerminationTimeout, func(ctx context.Context) {
			s.logger.Println("terminating HTTP Server component...")
			defer s.logger.Println("...HTTP Server component terminated")

			if err := httpServer.Shutdown(ctx); err != nil { // stop terminating HTTP Server if the time for that is over.
				s.logger.Printf("shutdown HTTP Server: %+v", err)
			}
		})

	s.logger.Printf("HTTP Server started on: %q\n", hostPort)
}

type Messaging struct {
	logger     *log.Logger
	terminator *graterm.Terminator
}

func NewMessaging(terminator *graterm.Terminator, logger *log.Logger) *Messaging {
	return &Messaging{terminator: terminator, logger: logger}
}

func (m *Messaging) Init() {
	defer m.logger.Println("Messaging initialized")
	m.terminator.WithOrder(MessagingTerminationOrder).
		WithName("Messaging").
		Register(messagingTerminationTimeout, func(ctx context.Context) {
			m.logger.Println("terminating Messaging component...")
			defer m.logger.Println("...Messaging component terminated")
		})
}

type FastDB struct {
	logger     *log.Logger
	terminator *graterm.Terminator
}

func NewFastDB(terminator *graterm.Terminator, logger *log.Logger) *FastDB {
	return &FastDB{terminator: terminator, logger: logger}
}

func (d *FastDB) Init() {
	defer d.logger.Println("FastDB initialized")
	d.terminator.WithOrder(FastDBTerminationOrder).
		WithName("FastDB").
		Register(fastDBTerminationTimeout, func(ctx context.Context) {
			d.logger.Println("terminating FastDB component...")
			defer d.logger.Println("...FastDB component terminated")

			panic(errors.New("BOOM!"))
		})
}

type SlowDB struct {
	logger     *log.Logger
	terminator *graterm.Terminator
}

func NewSlowDB(terminator *graterm.Terminator, logger *log.Logger) *SlowDB {
	return &SlowDB{terminator: terminator, logger: logger}
}

func (d *SlowDB) Init() {
	defer d.logger.Println("SlowDB initialized")
	d.terminator.WithOrder(SlowDBTerminationOrder).
		WithName("SlowDB").
		Register(slowDBTerminationTimeout, func(ctx context.Context) {
			d.logger.Println("terminating SlowDB component...")
			defer d.logger.Println("...SlowDB component terminated")
			select {
			case <-time.After(slowDBSleepTime):
				d.logger.Println("SlowDB cleanup finished")
			case <-ctx.Done():
				d.logger.Printf("SlowDB termination interrupted because of: %+v\n", ctx.Err())
			}
		})
}

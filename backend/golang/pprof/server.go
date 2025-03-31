package pprof

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"
)

const (
	pprofURL        = "/debug/pprof/"
	cmdlineURL      = "/debug/pprof/cmdline"
	profileURL      = "/debug/pprof/profile"
	symbolURL       = "/debug/pprof/symbol"
	traceURL        = "/debug/pprof/trace"
	goroutineURL    = "/debug/pprof/goroutine"
	heapURL         = "/debug/pprof/heap"
	threadcreateURL = "/debug/pprof/threadcreate"
	blockURL        = "/debug/pprof/block"
)

type Server struct {
	address           string
	readHeaderTimeout time.Duration
	httpServer        *http.Server
}

func NewServer(cfg Config) *Server {
	return &Server{
		address:           fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		readHeaderTimeout: cfg.ReadHeaderTimeout,
	}
}

func (s *Server) Run(_ context.Context) error {
	router := http.NewServeMux()
	router.HandleFunc(pprofURL, pprof.Index)
	router.HandleFunc(cmdlineURL, pprof.Cmdline)
	router.HandleFunc(profileURL, pprof.Profile)
	router.HandleFunc(symbolURL, pprof.Symbol)
	router.HandleFunc(traceURL, pprof.Trace)
	router.Handle(goroutineURL, pprof.Handler("goroutine"))
	router.Handle(heapURL, pprof.Handler("heap"))
	router.Handle(threadcreateURL, pprof.Handler("threadcreate"))
	router.Handle(blockURL, pprof.Handler("block"))

	s.httpServer = &http.Server{
		Addr:              s.address,
		Handler:           router,
		ReadHeaderTimeout: s.readHeaderTimeout,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Close() error {
	return s.httpServer.Close()
}

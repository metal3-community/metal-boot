//go:build ironic_wsgi

package wsgi

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

// Server runs a WSGI application inside an embedded CPython interpreter and
// exposes it as an http.Handler.
type Server struct {
	cfg Config
	log *slog.Logger

	jobs  chan *requestJob
	ready chan struct{}

	runCtx        context.Context
	workers       []*pythonWorker
	active        atomic.Int32
	lastRequestID atomic.Int64
}

// requestJob carries one HTTP request through the worker pool. Exactly one
// side (a worker, or the handler goroutine on timeout/disconnect) wins the
// grabbed flag and is responsible for writing the response.
type requestJob struct {
	w       http.ResponseWriter
	req     *http.Request
	r       requestReader
	grabbed atomic.Bool
	done    chan struct{}
}

// New creates an embedded WSGI server. The interpreter is not started until
// Run is called.
func New(log *slog.Logger, cfg Config) (*Server, error) {
	cfg = cfg.withDefaults()
	return &Server{
		cfg:   cfg,
		log:   log,
		jobs:  make(chan *requestJob, cfg.QueueSize),
		ready: make(chan struct{}),
	}, nil
}

// Run initializes the Python interpreter on this goroutine's (locked) OS
// thread, starts the worker pool, and blocks until ctx is cancelled, then
// finalizes the interpreter. The goroutine calling Run is the interpreter's
// main thread; Py_Finalize must run on it.
func (s *Server) Run(ctx context.Context) error {
	runtime.LockOSThread()

	s.log.Info("initializing embedded Python interpreter",
		"module", s.cfg.Module, "workers", s.cfg.Workers)
	if err := initPython(s.cfg); err != nil {
		return err
	}

	s.runCtx = ctx
	for i := range s.cfg.Workers {
		w := &pythonWorker{number: i + 1}
		s.workers = append(s.workers, w)
		go w.run(s)
	}
	close(s.ready)
	s.log.Info("embedded WSGI application ready", "module", s.cfg.Module)

	<-ctx.Done()

	// Let in-flight requests drain briefly before finalizing.
	drainDeadline := time.Now().Add(s.cfg.RequestTimeout)
	for s.active.Load() > 0 && time.Now().Before(drainDeadline) {
		time.Sleep(100 * time.Millisecond)
	}

	// Py_Finalize runs atexit handlers and can hang if the application
	// started uncooperative threads; escape-hatch after a timeout since the
	// process is shutting down anyway.
	finalized := make(chan struct{})
	go func() {
		select {
		case <-finalized:
		case <-time.After(15 * time.Second):
			s.log.Error("python interpreter finalize timed out, exiting")
			os.Exit(0)
		}
	}()
	deinitPython()
	close(finalized)
	s.log.Info("embedded Python interpreter finalized")
	return nil
}

// Handler returns the http.Handler bridging into the WSGI application.
func (s *Server) Handler() http.Handler { return s }

// ServeHTTP implements http.Handler following wsgo's Serve: buffer the
// request body before occupying a Python worker, enqueue the job, then wait
// for completion, client disconnect, or timeout.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	select {
	case <-s.ready:
	default:
		http.Error(w, "WSGI application is starting", http.StatusServiceUnavailable)
		return
	}

	if s.allWorkersStuck() {
		s.log.Error("all Python workers are stuck")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	var reader requestReader
	switch req.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		bufLen := s.cfg.RequestBufferLen
		if cl := int(req.ContentLength); cl >= 0 && cl < bufLen {
			bufLen = cl
		}
		reader = newBufferingReader(req.Body, bufLen)
	default:
		reader = newBufferingReader(req.Body, 0)
	}

	job := &requestJob{
		w:    w,
		req:  req,
		r:    reader,
		done: make(chan struct{}),
	}

	select {
	case s.jobs <- job:
	default:
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	timeout := time.NewTimer(s.cfg.RequestTimeout + 5*time.Second)
	defer timeout.Stop()

	select {
	case <-job.done:
	case <-req.Context().Done():
		if job.grabbed.Swap(true) {
			// Already being serviced; wait so the worker isn't writing to a
			// handler that has returned.
			<-job.done
		}
	case <-timeout.C:
		if !job.grabbed.Swap(true) {
			http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
			return
		}
		<-job.done
	}
}

func (s *Server) allWorkersStuck() bool {
	if len(s.workers) == 0 {
		return false
	}
	for _, w := range s.workers {
		if !w.stuck.Load() {
			return false
		}
	}
	return true
}

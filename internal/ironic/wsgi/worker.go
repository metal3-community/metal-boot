//go:build ironic_wsgi

package wsgi

/*
#include <Python.h>
#include <pythread.h>
#include <stdlib.h>
*/
import "C"

import (
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

// pythonWorker executes WSGI calls on a dedicated OS thread. The thread's
// PyThreadState is created on first use and reused for every subsequent task
// (PyEval_RestoreThread/PyEval_SaveThread), as in wsgo.
type pythonWorker struct {
	number      int
	stuck       atomic.Bool
	gilState    C.PyGILState_STATE
	threadState *C.PyThreadState
}

func (w *pythonWorker) run(s *Server) {
	// The GIL requires each thread state to stay on one OS thread.
	runtime.LockOSThread()

	for {
		select {
		case <-s.runCtx.Done():
			return
		case job := <-s.jobs:
			if job.grabbed.Swap(true) {
				// Timed out or client disconnected before we got here.
				continue
			}
			s.active.Add(1)
			w.runPythonTask(s, func() {
				s.handleJob(job)
			})
			s.active.Add(-1)
			close(job.done)
		}
	}
}

// runPythonTask acquires the GIL (reusing this thread's saved thread state),
// runs task, and arranges for a TimeoutError to be raised asynchronously in
// the Python thread if the task exceeds the request timeout.
func (w *pythonWorker) runPythonTask(s *Server, task func()) {
	if w.threadState != nil {
		C.PyEval_RestoreThread(w.threadState)
	} else {
		// First task on this thread: create its thread state.
		w.gilState = C.PyGILState_Ensure()
	}

	pydone := make(chan struct{})
	threadID := C.PyThread_get_thread_ident()

	if s.cfg.RequestTimeout > 0 {
		go func() {
			select {
			case <-pydone:
				return
			case <-time.After(s.cfg.RequestTimeout):
				s.log.Error("WSGI request timed out, interrupting worker",
					"worker", w.number)
				w.stuck.Store(true)
			}

			runtime.LockOSThread()
			gs := C.PyGILState_Ensure()
			if C.PyThreadState_SetAsyncExc(threadID, C.PyExc_TimeoutError) != 1 {
				s.log.Error("failed to interrupt stuck Python worker",
					"worker", w.number)
			}
			C.PyGILState_Release(gs)
			runtime.UnlockOSThread()
		}()
	}

	task()

	if w.stuck.Load() {
		w.cleanupStuckWorker()
		w.stuck.Store(false)
	}
	close(pydone)

	// Keep the thread state for the next task (PyEval_SaveThread rather than
	// PyGILState_Release).
	w.threadState = C.PyEval_SaveThread()
}

// handleJob runs one WSGI call. Must be called with the GIL held (from
// runPythonTask).
func (s *Server) handleJob(job *requestJob) {
	requestID := s.lastRequestID.Add(1)

	addRequestReader(requestID, job.r)
	defer removeRequestReader(requestID)

	ret := callApplication(requestID, job.req, s.cfg)

	badGateway := func() {
		http.Error(job.w, "Bad Gateway", http.StatusBadGateway)
	}

	if ret == nil {
		C.PyErr_Print()
		badGateway()
		return
	}
	defer C.Py_DecRef(ret)

	rs := takeResponseStart(requestID)
	if rs.status == 0 {
		s.log.Error("WSGI application did not call start_response",
			"path", job.req.URL.Path)
		badGateway()
		return
	}

	for k, vv := range rs.headers {
		for _, v := range vv {
			job.w.Header().Add(k, v)
		}
	}
	job.w.WriteHeader(rs.status)

	if err := readResponseToWriter(ret, job.w); err != nil {
		s.log.Warn("failed to write WSGI response",
			"path", job.req.URL.Path, "error", err)
	}
}

// cleanupStuckWorker releases locks that an asynchronously-raised exception
// may have leaked (notably the logging module's), to avoid deadlocking later
// requests. Must be called with the GIL held. Ported from wsgo.
func (w *pythonWorker) cleanupStuckWorker() {
	cmd := C.CString(`
import logging

try:
    for _ in range(50):
        logging._lock.release()
        print("Released leaked logging._lock!")
except Exception:
    pass

for wr in reversed(logging._handlerList[:]):
    try:
        h = wr()
        if h.lock:
            for _ in range(50):
                h.release()
                print("Released leaked %s lock!" % h)
    except Exception:
        pass
`)
	defer C.free(unsafe.Pointer(cmd))
	C.PyRun_SimpleStringFlags(cmd, nil)
}

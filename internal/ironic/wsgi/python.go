//go:build ironic_wsgi

package wsgi

/*
#cgo pkg-config: python3-embed

#include <Python.h>
#include <stdlib.h>

// Implemented in Go (exports.go).
extern void mb_wsgi_start_response(long request_id, char* status, int status_len, char** header_parts, int* header_part_lengths, int headers_size);
extern PyObject *mb_wsgi_read_request(long request_id, long to_read);
extern PyObject *mb_wsgi_read_request_line(long request_id);

// PyIter_Check is a macro in some Python versions; concretize it for cgo.
static int mb_pyiter_check(PyObject *o) {
	return PyIter_Check(o);
}

// start_response trampoline. METH_FASTCALL; self is a PyLong holding the
// request id, so one PyCFunction object binds one in-flight request.
static PyObject *mb_py_start_response(PyObject *self, PyObject *const *args, Py_ssize_t nargs) {
	if (nargs >= 2) {
		Py_ssize_t status_len;
		const char *status = PyUnicode_AsUTF8AndSize(args[0], &status_len);
		if (status == NULL) {
			PyErr_Print();
			Py_RETURN_NONE;
		}

		long request_id = PyLong_AsLong(self);

		Py_ssize_t headers_size = PyList_Size(args[1]);
		if (headers_size < 0) {
			PyErr_Print();
			Py_RETURN_NONE;
		}

		const char *header_parts[headers_size * 2];
		int header_part_lengths[headers_size * 2];

		for (Py_ssize_t i = 0; i < headers_size; i++) {
			PyObject *tup = PyList_GetItem(args[1], i);
			if (tup == NULL || !PyTuple_Check(tup) || PyTuple_Size(tup) != 2) {
				PyErr_Clear();
				Py_RETURN_NONE;
			}

			Py_ssize_t key_len, val_len;
			const char *key = PyUnicode_AsUTF8AndSize(PyTuple_GetItem(tup, 0), &key_len);
			const char *val = PyUnicode_AsUTF8AndSize(PyTuple_GetItem(tup, 1), &val_len);
			if (key == NULL || val == NULL) {
				PyErr_Print();
				Py_RETURN_NONE;
			}

			header_parts[i * 2] = key;
			header_part_lengths[i * 2] = (int)key_len;
			header_parts[1 + i * 2] = val;
			header_part_lengths[1 + i * 2] = (int)val_len;
		}

		mb_wsgi_start_response(request_id, (char *)status, (int)status_len,
			(char **)header_parts, header_part_lengths, (int)headers_size);
	}

	Py_RETURN_NONE;
}

static PyMethodDef mb_start_response_def = {
	"start_response",
	(PyCFunction)(void (*)(void))mb_py_start_response,
	METH_FASTCALL,
	NULL,
};

// mb_new_start_response returns a new start_response callable bound to a
// request id.
PyObject *mb_new_start_response(long request_id) {
	PyObject *self = PyLong_FromLong(request_id);
	if (self == NULL) {
		return NULL;
	}
	PyObject *fn = PyCFunction_NewEx(&mb_start_response_def, self, NULL);
	Py_DecRef(self);
	return fn;
}

// wsgi.input file-like object; reads delegate to the Go request reader.
typedef struct {
	PyObject_HEAD
	long request_id;
} mb_WsgiInput;

static void mb_wsgi_input_dealloc(PyObject *self) {
	PyObject_Del(self);
}

static PyObject *mb_wsgi_input_read(PyObject *self, PyObject *args) {
	long to_read = 0;
	if (!PyArg_ParseTuple(args, "|l:read", &to_read)) {
		return NULL;
	}
	return mb_wsgi_read_request(((mb_WsgiInput *)self)->request_id, to_read);
}

static PyObject *mb_wsgi_input_readline(PyObject *self, PyObject *args) {
	long limit = -1;
	if (!PyArg_ParseTuple(args, "|l:readline", &limit)) {
		return NULL;
	}
	return mb_wsgi_read_request_line(((mb_WsgiInput *)self)->request_id);
}

static PyObject *mb_wsgi_input_iter(PyObject *self) {
	Py_INCREF(self);
	return self;
}

static PyObject *mb_wsgi_input_next(PyObject *self) {
	PyObject *line = mb_wsgi_read_request_line(((mb_WsgiInput *)self)->request_id);
	if (line == NULL) {
		return NULL;
	}
	if (PyBytes_Size(line) == 0) {
		Py_DecRef(line);
		PyErr_SetNone(PyExc_StopIteration);
		return NULL;
	}
	return line;
}

static PyMethodDef mb_wsgi_input_methods[] = {
	{"read", mb_wsgi_input_read, METH_VARARGS, NULL},
	{"readline", mb_wsgi_input_readline, METH_VARARGS, NULL},
	{NULL, NULL, 0, NULL},
};

static PyTypeObject mb_wsgi_input_type = {
	PyVarObject_HEAD_INIT(NULL, 0)
	.tp_name = "metal_boot.WsgiInput",
	.tp_basicsize = sizeof(mb_WsgiInput),
	.tp_dealloc = mb_wsgi_input_dealloc,
	.tp_flags = Py_TPFLAGS_DEFAULT,
	.tp_doc = PyDoc_STR("WSGI input stream backed by the Go request body"),
	.tp_iter = mb_wsgi_input_iter,
	.tp_iternext = mb_wsgi_input_next,
	.tp_methods = mb_wsgi_input_methods,
};

PyObject *mb_create_wsgi_input(long request_id) {
	mb_WsgiInput *inp = PyObject_New(mb_WsgiInput, &mb_wsgi_input_type);
	if (inp == NULL) {
		return NULL;
	}
	inp->request_id = request_id;
	return (PyObject *)inp;
}

int mb_ready_types(void) {
	return PyType_Ready(&mb_wsgi_input_type);
}

// mb_initialise_python initializes the interpreter with unbuffered stdio and
// without Python signal handlers (Go owns signal handling).
void mb_initialise_python(void) {
	PyStatus status;
	PyConfig config;
	PyConfig_InitPythonConfig(&config);
	config.buffered_stdio = 0;
	config.install_signal_handlers = 0;

	status = Py_InitializeFromConfig(&config);
	if (PyStatus_Exception(status)) {
		PyConfig_Clear(&config);
		Py_ExitStatusException(status);
	}
	PyConfig_Clear(&config);
}
*/
import "C"

import (
	"errors"
	"io"
	"net/http"
	"runtime"
	"unsafe"
)

// appFunc holds a strong reference to the WSGI application callable. A
// process embeds exactly one interpreter, so this is package state, as in
// wsgo.
var appFunc *C.PyObject

// initPython must be called on a locked OS thread; that thread becomes the
// interpreter's main thread. On success the GIL is released and the caller
// may start worker threads.
func initPython(cfg Config) error {
	C.mb_initialise_python()

	code := C.CString(cfg.bootstrapCode())
	defer C.free(unsafe.Pointer(code))
	if C.PyRun_SimpleStringFlags(code, nil) != 0 {
		return errors.New(
			"python WSGI bootstrap failed (traceback on stderr): module " + cfg.Module,
		)
	}

	mainName := C.CString("__main__")
	defer C.free(unsafe.Pointer(mainName))
	mainMod := C.PyImport_AddModule(mainName) // borrowed
	if mainMod == nil {
		C.PyErr_Print()
		return errors.New("could not access __main__ module")
	}
	mainDict := C.PyModule_GetDict(mainMod) // borrowed

	appName := C.CString("application")
	defer C.free(unsafe.Pointer(appName))
	appFunc = C.PyDict_GetItemString(mainDict, appName) // borrowed
	if appFunc == nil {
		return errors.New("bootstrap did not define an 'application' callable")
	}
	C.Py_IncRef(appFunc)

	if C.mb_ready_types() != 0 {
		C.PyErr_Print()
		return errors.New("failed to ready WsgiInput type")
	}

	// Release the GIL; workers acquire it per task.
	C.PyEval_SaveThread()
	return nil
}

// deinitPython finalizes the interpreter. Must run on the thread that called
// initPython.
func deinitPython() {
	C.PyGILState_Ensure()
	C.Py_Finalize()
}

// callApplication invokes application(environ, start_response), returning a
// new reference to the WSGI response iterable (or nil with a Python error
// set). Must be called with the GIL held.
func callApplication(requestID int64, req *http.Request, cfg Config) *C.PyObject {
	args := C.PyTuple_New(2)
	if args == nil {
		return nil
	}
	C.PyTuple_SetItem(args, 0, createWsgiEnviron(requestID, req, cfg)) // steals
	C.PyTuple_SetItem(args, 1, C.mb_new_start_response(C.long(requestID)))

	ret := C.PyObject_CallObject(appFunc, args)
	C.Py_DecRef(args)
	return ret
}

// readResponseToWriter iterates the WSGI response, writing each chunk to w
// with the GIL released around the write, then calls close() on the
// response if present. Must be called with the GIL held.
func readResponseToWriter(response *C.PyObject, w io.Writer) error {
	iter := C.PyObject_GetIter(response)
	if iter == nil {
		C.PyErr_Print()
		return errors.New("WSGI response is not iterable")
	}
	defer C.Py_DecRef(iter)

	if C.mb_pyiter_check(iter) == 0 {
		return errors.New("WSGI response iterator is invalid")
	}

	var writeErr error
	for {
		item := C.PyIter_Next(iter)
		if item == nil {
			if C.PyErr_Occurred() != nil {
				C.PyErr_Print()
				writeErr = errors.New("WSGI response iteration raised")
			}
			break
		}

		var buf *C.char
		var size C.Py_ssize_t
		if C.PyBytes_AsStringAndSize(item, &buf, &size) == -1 {
			C.PyErr_Print()
			C.Py_DecRef(item)
			writeErr = errors.New("WSGI response chunk is not bytes")
			break
		}

		if size > 0 {
			// Copy so the write can happen without the GIL.
			chunk := C.GoBytes(unsafe.Pointer(buf), C.int(size))

			gilState := C.PyThreadState_Get()
			C.PyEval_SaveThread()
			n, err := w.Write(chunk)
			C.PyEval_RestoreThread(gilState)

			if err != nil {
				C.Py_DecRef(item)
				writeErr = err
				break
			}
			if n != int(size) {
				C.Py_DecRef(item)
				writeErr = io.ErrShortWrite
				break
			}
		}
		C.Py_DecRef(item)
	}

	// WSGI requires close() to be called on the response if present.
	closeName := C.CString("close")
	defer C.free(unsafe.Pointer(closeName))
	if C.PyObject_HasAttrString(response, closeName) == 1 {
		closeFn := C.PyObject_GetAttrString(response, closeName)
		if closeFn != nil {
			ret := C.PyObject_CallObject(closeFn, nil)
			if ret == nil {
				C.PyErr_Print()
			} else {
				C.Py_DecRef(ret)
			}
			C.Py_DecRef(closeFn)
		}
	}

	return writeErr
}

// doReadRequest services wsgi.input read()/readline() calls from Python,
// releasing the GIL for the duration of the (potentially blocking) Go read.
func doReadRequest(requestID C.long, toRead C.long, readLine bool) *C.PyObject {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	gilState := C.PyThreadState_Get()
	C.PyEval_SaveThread()

	// read() with no size means "the whole body"; cap a single slurp.
	bufLen := 32 << 20
	if toRead > 0 {
		bufLen = int(toRead)
	}
	buf := make([]byte, bufLen)

	rr := getRequestReader(int64(requestID))

	var n int
	if rr != nil {
		if readLine {
			n, _ = rr.Readline(buf)
		} else {
			n, _ = io.ReadFull(rr, buf)
		}
	}

	var cstr *C.char
	if n > 0 {
		cstr = (*C.char)(unsafe.Pointer(&buf[0]))
	}

	C.PyEval_RestoreThread(gilState)

	return C.PyBytes_FromStringAndSize(cstr, C.Py_ssize_t(n))
}

//go:build ironic_wsgi

package wsgi

/*
#include <Python.h>

extern PyObject *mb_create_wsgi_input(long request_id);
*/
import "C"

import (
	"net"
	"net/http"
	"strings"
	"unsafe"
)

// createWsgiEnviron builds the WSGI environ dict for a request. Returns a new
// reference. Must be called with the GIL held. Ported from wsgo's
// CreateWsgiEnvironment.
func createWsgiEnviron(requestID int64, req *http.Request, cfg Config) *C.PyObject {
	environ := C.PyDict_New()

	pyDictSetString(environ, "REQUEST_METHOD", req.Method)
	pyDictSetString(environ, "SCRIPT_NAME", "")
	pyDictSetString(environ, "PATH_INFO", req.URL.Path)
	pyDictSetString(environ, "QUERY_STRING", req.URL.RawQuery)
	pyDictSetString(environ, "CONTENT_TYPE", req.Header.Get("Content-Type"))
	pyDictSetString(environ, "CONTENT_LENGTH", req.Header.Get("Content-Length"))
	pyDictSetString(environ, "SERVER_NAME", cfg.ServerName)
	pyDictSetString(environ, "SERVER_PORT", cfg.ServerPort)
	pyDictSetString(environ, "SERVER_PROTOCOL", req.Proto)
	pyDictSetString(environ, "HTTP_HOST", req.Host)

	if host, port, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		pyDictSetString(environ, "REMOTE_ADDR", host)
		pyDictSetString(environ, "REMOTE_PORT", port)
	}

	scheme := req.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
		if req.TLS != nil {
			scheme = "https"
		}
	}
	pyDictSetString(environ, "wsgi.url_scheme", scheme)

	pyDictSetObject(environ, "wsgi.multiprocess", C.Py_False)
	pyDictSetObject(environ, "wsgi.multithread", C.Py_True)
	pyDictSetObject(environ, "wsgi.run_once", C.Py_False)

	version := C.PyTuple_New(2)
	C.PyTuple_SetItem(version, 0, C.PyLong_FromLong(1)) // steals
	C.PyTuple_SetItem(version, 1, C.PyLong_FromLong(0)) // steals
	pyDictSetObject(environ, "wsgi.version", version)
	C.Py_DecRef(version)

	// HTTP_* headers, CGI-style (from Go's net/http/cgi).
	for k, v := range req.Header {
		k = strings.Map(upperCaseAndUnderscore, k)
		if k == "PROXY" {
			// golang.org/issue/16405
			continue
		}
		joinStr := ", "
		if k == "COOKIE" {
			joinStr = "; "
		}
		pyDictSetString(environ, "HTTP_"+k, strings.Join(v, joinStr))
	}

	input := C.mb_create_wsgi_input(C.long(requestID))
	pyDictSetObject(environ, "wsgi.input", input)
	C.Py_DecRef(input)

	stderrName := C.CString("stderr")
	stderr := C.PySys_GetObject(stderrName) // borrowed
	C.free(unsafe.Pointer(stderrName))
	pyDictSetObject(environ, "wsgi.errors", stderr)

	return environ
}

func upperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		return '_'
	}
	return r
}

// pyDictSetString sets a latin-1 string value. WSGI uses latin-1 strings even
// when the octets represent another encoding (codepoints 0-255 map to bytes).
func pyDictSetString(dict *C.PyObject, key, value string) {
	keyStr := C.CString(key)
	defer C.free(unsafe.Pointer(keyStr))
	keyObj := C.PyUnicode_DecodeLatin1(keyStr, C.Py_ssize_t(len(key)), nil)
	defer C.Py_DecRef(keyObj)

	valueStr := C.CString(value)
	defer C.free(unsafe.Pointer(valueStr))
	valueObj := C.PyUnicode_DecodeLatin1(valueStr, C.Py_ssize_t(len(value)), nil)
	defer C.Py_DecRef(valueObj)

	if keyObj != nil && valueObj != nil {
		C.PyDict_SetItem(dict, keyObj, valueObj)
	}
}

func pyDictSetObject(dict *C.PyObject, key string, obj *C.PyObject) {
	keyStr := C.CString(key)
	defer C.free(unsafe.Pointer(keyStr))
	keyObj := C.PyUnicode_DecodeLatin1(keyStr, C.Py_ssize_t(len(key)), nil)
	defer C.Py_DecRef(keyObj)

	if keyObj != nil && obj != nil {
		C.PyDict_SetItem(dict, keyObj, obj)
	}
}

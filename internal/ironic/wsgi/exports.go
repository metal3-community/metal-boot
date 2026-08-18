//go:build ironic_wsgi

// This file holds the cgo-exported callbacks the C shim in python.go calls
// back into. Per cgo rules its preamble may only contain declarations.

package wsgi

/*
#include <Python.h>
*/
import "C"

import (
	"strconv"
	"sync"
	"unsafe"
)

// responseStart captures the arguments of a WSGI start_response call.
type responseStart struct {
	status        int
	statusMessage string
	headers       map[string][]string
}

var (
	responseStarts      = map[int64]responseStart{}
	responseStartsMutex sync.Mutex

	requestReaders      = map[int64]requestReader{}
	requestReadersMutex sync.Mutex
)

func addRequestReader(requestID int64, r requestReader) {
	requestReadersMutex.Lock()
	requestReaders[requestID] = r
	requestReadersMutex.Unlock()
}

func removeRequestReader(requestID int64) {
	requestReadersMutex.Lock()
	delete(requestReaders, requestID)
	requestReadersMutex.Unlock()
}

func getRequestReader(requestID int64) requestReader {
	requestReadersMutex.Lock()
	defer requestReadersMutex.Unlock()
	return requestReaders[requestID]
}

// takeResponseStart removes and returns the recorded start_response data for
// a request. A zero status means start_response was never called.
func takeResponseStart(requestID int64) responseStart {
	responseStartsMutex.Lock()
	defer responseStartsMutex.Unlock()
	rs := responseStarts[requestID]
	delete(responseStarts, requestID)
	return rs
}

//export mb_wsgi_start_response
func mb_wsgi_start_response(
	requestID C.long,
	status *C.char,
	statusLen C.int,
	headerParts **C.char,
	headerPartLengths *C.int,
	headersSize C.int,
) {
	// Status lines look like "200 OK"; a bare "200" is tolerated.
	statusLine := C.GoStringN(status, statusLen)
	if len(statusLine) < 3 || (len(statusLine) > 3 && statusLine[3] != ' ') {
		return
	}
	statusCode, err := strconv.Atoi(statusLine[0:3])
	if err != nil {
		return
	}
	message := ""
	if len(statusLine) > 4 {
		message = statusLine[4:]
	}

	rs := responseStart{
		status:        statusCode,
		statusMessage: message,
		headers:       make(map[string][]string, int(headersSize)),
	}

	parts := unsafe.Slice(headerParts, int(headersSize)*2)
	lengths := unsafe.Slice(headerPartLengths, int(headersSize)*2)
	for i := 0; i < int(headersSize); i++ {
		k := C.GoStringN(parts[i*2], lengths[i*2])
		v := C.GoStringN(parts[1+i*2], lengths[1+i*2])
		rs.headers[k] = append(rs.headers[k], v)
	}

	responseStartsMutex.Lock()
	responseStarts[int64(requestID)] = rs
	responseStartsMutex.Unlock()
}

//export mb_wsgi_read_request
func mb_wsgi_read_request(requestID C.long, toRead C.long) *C.PyObject {
	return doReadRequest(requestID, toRead, false)
}

//export mb_wsgi_read_request_line
func mb_wsgi_read_request_line(requestID C.long) *C.PyObject {
	return doReadRequest(requestID, 0, true)
}

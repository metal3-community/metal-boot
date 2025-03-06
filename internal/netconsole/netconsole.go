package netconsole

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

func ListenAndServe(network, addr string, handle func(addr net.Addr, l Log)) *Server {
	pc, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

	// Accept as many messages as indicated by input slice.
	var logsWG sync.WaitGroup

	// Gather valid logs into the slice, but decrement the waitgroup on
	// all messages so that we can ensure deterministic test output.
	var logs []Log
	s := &Server{
		handle: func(_ net.Addr, l Log) {
			defer logsWG.Done()
			logs = append(logs, l)
		},
		drop: func(_ net.Addr, _ []byte) {
			defer logsWG.Done()
		},
	}

	// Ensure serve goroutine is cleaned up.
	var serveWG sync.WaitGroup
	serveWG.Add(1)

	ctx, _ := context.WithCancel(context.Background())

	go func() {
		defer serveWG.Done()

		err = s.serve(ctx, pc)
		if err != nil {
			panic(fmt.Sprintf("failed to serve: %v", err))
		}

	}()

	if handle == nil {
		// By default, do nothing with processed logs.
		handle = func(_ net.Addr, _ Log) {}
	}

	return &Server{
		network: network,
		addr:    addr,
		handle:  handle,
		// By default, do nothing with dropped logs.
		drop: func(_ net.Addr, _ []byte) {},
	}
}

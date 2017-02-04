package logging

import (
	"fmt"
	"net"
	"time"

	"github.com/op/go-logging"
)

const (
	size = 1024
	path = "/tmp/ctop.sock"
)

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

type CTopLogger struct {
	*logging.Logger
	backend *logging.MemoryBackend
	done    chan bool
}

func New(serverEnabled string) *CTopLogger {
	log := &CTopLogger{
		logging.MustGetLogger("ctop"),
		logging.NewMemoryBackend(size),
		make(chan bool),
	}

	backendFmt := logging.NewBackendFormatter(log.backend, format)
	logging.SetBackend(backendFmt)
	log.Info("logger initialized")

	if serverEnabled == "1" {
		log.Serve()
	}

	return log
}

func (log *CTopLogger) Exit() {
	log.done <- true
}

func (log *CTopLogger) Serve() {
	ln, err := net.Listen("unix", path)
	if err != nil {
		panic(err)
	}

	go func() {
		switch {
		case <-log.done:
			ln.Close()
			return
		default:
			//
		}
	}()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			go log.handler(conn)
		}
	}()

	log.Info("logging server started")
}

func (log *CTopLogger) handler(conn net.Conn) {
	defer conn.Close()
	for msg := range log.tail() {
		msg = fmt.Sprintf("%s\n", msg)
		conn.Write([]byte(msg))
	}
}

func (log *CTopLogger) tail() chan string {
	stream := make(chan string)

	node := log.backend.Head()
	go func() {
		for {
			stream <- node.Record.Formatted(0)
			for {
				nnode := node.Next()
				if nnode != nil {
					node = nnode
					break
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return stream
}
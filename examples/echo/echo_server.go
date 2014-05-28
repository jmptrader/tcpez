package main

import (
	"flag"
	"github.com/paperlesspost/agency/tcpez"
)

type EchoHandler struct{}

func (h *EchoHandler) Respond(req []byte, span *tcpez.Span) (response []byte, err error) {
	return req, nil
}

func main() {
	flag.Parse()
	l := tcpez.NewServer(":2000", new(EchoHandler))
	l.Start()
}

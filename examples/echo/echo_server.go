package main

import (
	"github.com/paperlesspost/tcpez"
)

type EchoHandler struct{}

func (h *EchoHandler) Respond(req []byte, span *tcpez.Span) (response []byte, err error) {
	return req, nil
}

func main() {
	l, _ := tcpez.NewServer(":2000", new(EchoHandler))
	l.Start()
}

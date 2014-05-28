package main

import (
	"flag"
	"github.com/paperlesspost/agency/tcpez"
	"time"
)

func main() {
	flag.Parse()
	c := tcpez.NewClient([]string{":2000"}, 1, 1)
	for {
		c.SendRecv([]byte("PING"))
		time.Sleep(1000)
	}
}

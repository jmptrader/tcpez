package main

import (
	"github.com/paperlesspost/tcpez"
	"time"
)

func main() {
	c, _ := tcpez.NewClient([]string{":2000"}, 1, 1)
	for {
		c.SendRecv([]byte("PING"))
		time.Sleep(1000)
	}
}

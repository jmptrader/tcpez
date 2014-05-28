// reqrep_client is an example of a client to a tcpez ProtoServer
//
// Usage
//
//      go run reqrep_client.go --logtostderr
//
package main

import (
	reqrep "./reqrep"
	proto "code.google.com/p/goprotobuf/proto"
	"flag"
	"github.com/golang/glog"
	"github.com/paperlesspost/agency/tcpez"
	"time"
)

// Client is just a tcpez.Client with its own methods
type Client struct {
	*tcpez.Client
}

func NewClient(addresses []string, initPool, maxPool int) (c *Client) {
	return &Client{tcpez.NewClient(addresses, initPool, maxPool)}
}

// Do is the magical method on client that does the request and response (un)marshalling 
func (c *Client) Do(request *reqrep.Request) (response *reqrep.Response, err error) {
        // marshall the Request into a byte slice
	req, err := proto.Marshal(request)
	if err != nil {
		return
	}
        // Send that over the wire and get a byte slice of the response
	resp, err := c.SendRecv(req)
	if err != nil {
		return
	}
        // unmarshall the response byte slice into a Response struct
	response = new(reqrep.Response)
	err = proto.Unmarshal(resp, response)
	if err != nil {
		return
	}
	return
}

func main() {
        // We need to call flag.Parse() to invoke the flags on the glog logger that's
        // included in tcpez server
	flag.Parse()
        // Initialize the client with a 1 connection pool, that's fine since were not doing anything conccurently
        // but in real production situations we'd want to tune this 
	c := NewClient([]string{":2000"}, 1, 1)
	for {
                // Create a request struct and then send it
		response, err := c.Do(&reqrep.Request{Command: proto.String("SUCCEED"), Args: proto.String("nil")})
		if err == nil {
			glog.Infof("[reqrep] %s %s", response.GetStatus(), response.GetMessage())
		}
		time.Sleep(1000)
	}
}

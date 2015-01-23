// reqrep server is an example of using the tcpez protoserver
//
// Usage
//
//      go run reqrep_server.go &
//      go run reqrep_client.go
//
package main

import (
	reqrep "./reqrep"
	proto "code.google.com/p/goprotobuf/proto"
	"github.com/paperlesspost/tcpez"
)

// NewProtoServer wraps the tcpez.NewProtoServer defining the handler functions required for
// setting up the server, It takes a tcp address to bind to and a StatsRecorder to stats to
func NewProtoServer(address string, stats tcpez.StatsRecorder) *tcpez.Server {
	// The requestFunc is invoked when turning the raw request on the line
	// into the "Request" struct that you've defined for your application
	requestFunc := tcpez.ProtoInitializerFunc(func() proto.Message {
		return new(reqrep.Request)
	})
	// The responseFunc is invoked when turning the raw request on the line
	// into the "Response" struct that you've defined for your application
	responseFunc := tcpez.ProtoInitializerFunc(func() proto.Message {
		return new(reqrep.Response)
	})

	// The ProtoHandlerFunc defines how to respond to a request object. tcpez takes care of the
	// proto (un)marshalling as long as this func responsds with a proto.Message
	handlerFunc := tcpez.ProtoHandlerFunc(func(req proto.Message, res proto.Message, span *tcpez.Span) {
		// We need to do a type assertion here which turns the proto.Message interface into
		// a struct of our request type that we've defined in our proto schema
		request := req.(*reqrep.Request)
		response := res.(*reqrep.Response)
		// This is dumb logic, but your actual request handling logic would go here
		if request.GetCommand() == "SUCCEED" {
			response.Status = proto.String("OK")
			response.Message = proto.String("Request Succeeded")
		} else {
			response.Status = proto.String("ERR")
			response.Message = proto.String("Request Failed")
		}
	})
	// Initialize the actual server
	server, _ := tcpez.NewProtoServer(address, requestFunc, responseFunc, handlerFunc)
	server.Stats = stats
	return server
}

func main() {
	s := NewProtoServer(":2000", new(tcpez.DebugStatsRecorder))
	s.Start()
}

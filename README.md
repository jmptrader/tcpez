# tcpez

(pronounced tee-cee-pee-zee)

tcpez is a simple protocol and client server implementation for a request/response TCP Server using Go (#golang). 

## Why

When working on a simple tcp server application, we identified a common pattern we had seen in the wild (requests and responses encoded as protocol buffers with a simple length header). We decided to create a simple wrapper, one level up from a simple net.TCPListener that would allow us to encapsulate a number of useful abstractions into a reusable package. Our concerns/goals:

- We didn't want to think about setting up listeners and the common request handling loop boilerplate.
- We knew we wanted to keep it constrained to simple request/response (and not other more complex topologies or patterns). The beauty of this is that to the client, this can appear very simple and work well for other environments where there isnt a great concurrency story (ie Ruby, JS, etc). On the server side, however, go provides us with great abilities to spread this work across as many cores or goroutines as we want.
- We wanted to build in the ability to track and log details about the request (ie `Span`'s)
- We wanted the and protocol to be very simple, but be able to build more complex layers on top of it (`tcpez.Server` is dead simple, `tcpez.ProtoServer` adds a layer of data encoding). 
- We wanted to create simple client implementations that could load balance and spread requests across many servers.

The `ProtoServer` implementation was extracted and is in use in PaperlessPost's core services including our image processing pipeline.

## Server

A server is initialized with a 'RequestHandler' which is any object that satisfies the interface:

    type RequestHandler interface {
            Respond([]byte, *Span) ([]byte, error)
    }

The simplest server parses the request bytes, determines the response, and sends it back as bytes. The tcpez.Server handles the connections and protocol parsing/encoding.

Here is the simplest possible EchoSever:

    type EchoHandler struct{}

    func (h *EchoHandler) Respond(req []byte, span *tcpez.Span) (response []byte, err error) {
            return req, nil
    }

    func main() {
            flag.Parse()
            l := tcpez.NewServer(":2000", new(EchoHandler))
            l.Start()
    }

There is also a `ProtoServer` which is a small abstraction on top of `tcpez.Server` to handle requests and responses encoded in arbitrary protocol buffer schemas. This is the implementation that we use primarily in our production systems.

## Client

The reference client implementation is in Go and is equally simple to the server. A client is initialized with an array of addresses which are added to a connection pool. You can then call `SendRecv` on the client object which will Send the bytes to a random server in the pool and then block waiting for the response.


	c := tcpez.NewClient([]string{"localhost:2000"}, 1, 1)
	resp, err := c.SendRecv([]byte("PING"))
	resp //=> []byte("PONG")
	
The client also comes with a sample pipeline implementation which allows for sending multiple requests at once:

	// Initialize a pipeline
	pipe := c.Pipeline()
	// Sends store in an internal buffer
	pipe.Send([]byte("PING1"))
	pipe.Send([]byte("PING2"))
	// Flush must be called to deliver the buffer 
	// to the connection
	responses, err := pipe.Flush()
	responses //=> [[]byte("PONG1"),[]byte("PONG2")]
	// Responses are returned as an array in order of
	// the requests

## Protocol

The protocol for tcpez is extremely simple. A message is defined as

`|4 bit int32 length header|length number of bytes|`

(Note: `|` are just visual delimiters)

Request: `|4|PING|`

Response: `|4|PONG|`

If the length header is a negative number, this is a pipelined request and the absolute value of the header is the number of messages being sent on the wire. 

Request: `|-2|5|PING1|5|PING2|`

Response: `|-2|5|PONG1|5|PONG2|`

## Logging/Stats

tcpez uses [glog](https://github.com/golang/glog) for leveled logging internally. You *must* call `flag.Parse()` in your application's main function to enable the logging and its command line options. tcpez uses a simple implementation of what we call Span's that are used to track metadata and subroutine durations during a request. Every request has a unique span (with a unique id) that is initialized and passed through the request handler. At the end of the request, any metadata and timings added to this span are logged as JSON and flushed to a `StatsRecorder`. There is an optional `StatsdStatsRecorder` that will flush this data to an instance of Statsd for graphing, etc.

## About

tcpez was created by Aaron Quint (quirkey) at Paperless Post (http://www.paperlesspost.com) and is licensed under the MIT license.



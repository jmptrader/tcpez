// Server is the base tcpez server. It sets up a tcp listener on an address,
// and given a RequestHandler, it parses the tcpez protocol format and turns
// it into individual request/responses. Each Connection is handled on a
// seperate goroutine and pipelined requests are first parsed then farmed
// to seperate goroutines. Pipelined requests from the client are handled
// seamlessly this way, each seperate request is passed to its own RequestHandler
// with its own Span.
package tcpez

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/op/go-logging"
	"io"
	"net"
	"sync"
	"time"
)

var log = logging.MustGetLogger("tcpez")
var LogFormat = logging.MustStringFormatter("%{time:2006-01-02T15:04:05.999999999Z07:00} %{level} [%{module}] %{message}")

func init() {
	logging.SetLevel(logging.INFO, "tcpez")
	logging.SetFormatter(LogFormat)
}

// Server is the base struct that wraps the tcp listener and allows
// for setting the RequestHandler that takes each request and returns
// an response
type Server struct {
	// Address is the string address that the tcp listener is
	// bound to
	Address string
	// Handler is a value that responds to the RequestHandler
	// interface.
	Handler RequestHandler
	// StatsRecorder is the value that delivers stats to a collection
	// agent. This is the DebugStatsRecorder by default (does nothing)
	// but can be swapped to send to StatsD. This is passed to each
	// Span created and passed to the RequestHandler
	Stats StatsRecorder
	// UUIDGenerator generates UUIDs for each Span, by default this uses
	// a simple hash function but can be swapped out for something more
	// complex (a vector-clock style UUID generator for example)
	UUIDGenerator UUIDGenerator

	// The underlying TCP listener, access if you need to set timeouts, etc
	Conn *net.TCPListener

	isClosed    bool
	connId      int
	clientConns map[int]net.Conn
}

// RequestHandler is the basic interface for setting up the request handling
// logic of a tcpez server. The server handles all the request parsing and setup
// as well as the response encoding. All you have to do to create a working server
// is create an object that has a Respond() method that takes a byte slice (which
// is the request) and a Span (which allows you to track timings and meta data
// through the request) and then it returns a byte slice which is the response.
//
//        type MyHandler struct
//
//        func (h *MyHandler) Respond(req []byte, span *tcpez.Span) (res []byte, err error) {
//              if string(req) == "PING" {
//                  span.Attr("command", "PING")
//                  return []byte{"PONG"}, nil
//              }
//        }
//
type RequestHandler interface {
	Respond([]byte, *Span) ([]byte, error)
}

// NewServer is the tcpez server intializer. It only requires two parameters,
// an address to bind to (same format as net.ListenTCP) and a RequestHandler
// which serves the requests.
func NewServer(address string, handler RequestHandler) (s *Server, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}

	return &Server{Address: address, Conn: l, Handler: handler, Stats: new(DebugStatsRecorder), UUIDGenerator: DefaultUUIDGenerator, clientConns: make(map[int]net.Conn)}, nil
}

// Start starts the Connection handling and request processing loop.
// This is a blocking operation and can be started in a goroutine.
func (s *Server) Start() {
	log.Debug("Listening on %s", s.Conn.Addr().String())
	for {
		if s.isClosed == true {
			break
		}
		clientConn, err := s.Conn.Accept()
		if err != nil {
			log.Warning(err.Error())
			break
		}
		s.connId++
		go s.handle(clientConn, s.connId)
	}
	log.Debug("Closing %s", s.Conn.Addr().String())
}

func (s *Server) NumConnections() int {
	return len(s.clientConns)
}

// Close closes the server listener to any more Connections
func (s *Server) Close() (err error) {
	if s.isClosed == false {
		err = s.Conn.Close()
		s.isClosed = true
		for id, conn := range s.clientConns {
			delete(s.clientConns, id)
			conn.Close()
		}
		return
	}
	return errors.New("Closing already closed Connection")
}

func (s *Server) handle(clientConn net.Conn, id int) {
	log.Debug("[tcpez] New client(%s)", clientConn.RemoteAddr())
	s.clientConns[id] = clientConn
	for {
		// Timeout the connection after 5 mins
		clientConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		header, response, err := s.readHeaderAndHandleRequest(clientConn)
		if err != nil {
			if closableError(err) {
				// EOF the client has disconnected
				break
			}
			log.Error(err.Error())
			s.Stats.Increment("operation.failure")
			return
		}
		err = s.sendResponse(clientConn, header, response)
		if err != nil {
			if closableError(err) {
				// EOF the client has disconnected
				break
			}
			log.Error(err.Error())
			s.Stats.Increment("operation.failure")
			return
		}
		s.Stats.Increment("operation.success")
	}
	log.Debug("Closing connection %v", clientConn)
	clientConn.Close()
	delete(s.clientConns, id)
}

func closableError(err error) bool {
	if err, ok := err.(net.Error); ok == true {
		return err.Timeout() == true
	}
	return err == io.EOF || err == io.ErrClosedPipe || err == io.ErrUnexpectedEOF
}

func (s *Server) readHeaderAndHandleRequest(buf io.Reader) (header int32, response []byte, err error) {
	var size int32
	err = binary.Read(buf, binary.BigEndian, &size)
	if err != nil {
		return 0, nil, err
	}
	if size < 0 {
		// this is a pipelined request
		var wg sync.WaitGroup
		count := -size
		requests := make([][]byte, count)
		responses := make([][]byte, count)
		for r := 0; int32(r) < count; r++ {
			request, err := s.parseRequest(buf, 0)
			if err == nil {
				requests[r] = request
				wg.Add(1)
				go func(index int) {
					res, err := s.handleRequest(requests[index], true)
					if err == nil {
						responses[index] = res
					}
					wg.Done()
				}(r)
			}
		}
		wg.Wait()
		output := bytes.NewBuffer(nil)
		for j := 0; int32(j) < count; j++ {
			length := int32(len(responses[j]))
			err = binary.Write(output, binary.BigEndian, length)
			if err == nil {
				output.Write(responses[j])
			}
		}
		return int32(-count), output.Bytes(), err
	} else {
		request, err := s.parseRequest(buf, size)
		if err != nil {
			return 0, nil, err
		}
		response, err := s.handleRequest(request, false)
		if err != nil {
			return 0, nil, err
		}
		return int32(len(response)), response, nil
	}
	return
}

func (s *Server) sendResponse(w io.Writer, header int32, data []byte) (err error) {
	err = binary.Write(w, binary.BigEndian, header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return
}

func (s *Server) parseRequest(buf io.Reader, size int32) (request []byte, err error) {
	if size == int32(0) {
		err = binary.Read(buf, binary.BigEndian, &size)
		if err != nil {
			return nil, err
		}
	}
	request = make([]byte, size)
	_, err = io.ReadFull(buf, request)
	if err != nil {
		return nil, err
	}
	return
}

func (s *Server) handleRequest(request []byte, multi bool) (response []byte, err error) {
	span := NewSpan(s.UUIDGenerator())
	if multi == true {
		span.Attr("multi", "true")
	}
	span.Stats = s.Stats
	span.Start("duration")
	span.Add("num_connections", int64(s.NumConnections()))
	response, err = s.Handler.Respond(request, span)
	span.Finish("duration")
	log.Info("%s", span.JSON())
	span.Record()
	return
}

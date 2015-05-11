package tcpez

import (
	"github.com/golang/protobuf/proto"
	json "encoding/json"
	"errors"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/op/go-logging"
	math "math"
	"net"
	"syscall"
	"testing"
	"time"
)

type EchoHandler struct{}

func (h *EchoHandler) Respond(req []byte, span *Span) (response []byte, err error) {
	return req, nil
}

func (h *EchoHandler) Record(span *Span) {
}

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type Request struct {
	Command          *string `protobuf:"bytes,1,req,name=command" json:"command,omitempty"`
	Args             *string `protobuf:"bytes,2,req,name=args" json:"args,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}

func (m *Request) GetCommand() string {
	if m != nil && m.Command != nil {
		return *m.Command
	}
	return ""
}

func (m *Request) GetArgs() string {
	if m != nil && m.Args != nil {
		return *m.Args
	}
	return ""
}

type Response struct {
	Status           *string `protobuf:"bytes,1,req,name=status" json:"status,omitempty"`
	Message          *string `protobuf:"bytes,2,req,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}

func (m *Response) GetStatus() string {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return ""
}

func (m *Response) GetMessage() string {
	if m != nil && m.Message != nil {
		return *m.Message
	}
	return ""
}

func init() {
	logging.SetLevel(logging.ERROR, "tcpez")
}

func TestEchoServer(t *testing.T) {
	addr := "127.0.0.1:2001"
	l, err := NewServer(addr, new(EchoHandler))
	if err != nil {
		log.Error(err.Error())
	}
	assert.T(t, l != nil)
	go l.Start()
	defer l.Close()
	c, err := NewClient([]string{addr}, 3, 3*time.Second)
	assert.T(t, c != nil)
	var resp []byte
	for i := 0; i < 100; i++ {
		resp, err = c.SendRecv([]byte("PING"))
		assert.T(t, err == nil)
		assert.Equal(t, []byte("PING"), resp)
	}
}

func TestEchoServerPipelined(t *testing.T) {
	addr := "127.0.0.1:2001"
	l, _ := NewServer(addr, new(EchoHandler))
	assert.T(t, l != nil)
	go l.Start()
	defer l.Close()
	c, _ := NewClient([]string{addr}, 3, 3*time.Second)
	assert.T(t, c != nil)
	pipe := c.Pipeline()
	for i := 0; i < 10; i++ {
		pipe.Send([]byte(fmt.Sprintf("PING%d", i)))
	}
	returned, _ := pipe.Flush()
	for i := 0; i < 10; i++ {
		assert.Equal(t, string(returned[i]), (fmt.Sprintf("PING%d", i)))
	}
}

func TestProtoServer(t *testing.T) {
	addr := "127.0.0.1:2001"
	requestFunc := ProtoInitializerFunc(func() proto.Message {
		return new(Request)
	})
	responseFunc := ProtoInitializerFunc(func() proto.Message {
		return new(Response)
	})
	handlerFunc := ProtoHandlerFunc(func(req proto.Message, res proto.Message, span *Span) {
		r := req.(*Request)
		message := fmt.Sprintf("Got command: %s args: %s", r.GetCommand(), r.GetArgs())
		span.Increment("response")
		response := res.(*Response)
		response.Status = proto.String("OK")
		response.Message = proto.String(message)
	})
	l, _ := NewProtoServer(addr, requestFunc, responseFunc, handlerFunc)
	assert.T(t, l != nil)
	assert.Equal(t, addr, l.Address)
	go l.Start()
	defer l.Close()
	c, _ := NewClient([]string{addr}, 3, 3*time.Second)
	assert.T(t, c != nil)

	iter := 100

	for i := 0; i < iter; i++ {
		var req []byte
		var resp []byte
		var request *Request
		var response *Response
		var err error
		request = &Request{Command: proto.String("GET"), Args: proto.String("/")}
		req, err = proto.Marshal(request)
		assert.T(t, req != nil)
		assert.T(t, err == nil)
		resp, err = c.SendRecv(req)
		assert.T(t, err == nil)
		assert.T(t, resp != nil)
		response = new(Response)
		err = proto.Unmarshal(resp, response)
		assert.T(t, err == nil)
		assert.Equal(t, "OK", response.GetStatus())
	}
}

func TestEchoServerReconnect(t *testing.T) {
	addr := "127.0.0.1:2001"
	l, err := NewServer(addr, new(EchoHandler))
	if err != nil {
		log.Error(err.Error())
	}
	assert.T(t, l != nil)
	go l.Start()
	c, err := NewClient([]string{addr}, 3, 3*time.Second)
	assert.T(t, c != nil)
	var resp []byte
	for i := 0; i < 10; i++ {
		resp, err = c.SendRecv([]byte("PING"))
		assert.T(t, err == nil)
		assert.Equal(t, []byte("PING"), resp)
	}
	l.Close()
	time.Sleep(1 * time.Second)
	resp, err = c.SendRecv([]byte("PING"))
	assert.T(t, resp == nil)
	assert.T(t, err != nil)
	assert.Equal(t, 0, l.NumConnections())
	l, err = NewServer(addr, new(EchoHandler))
	if err != nil {
		log.Error(err.Error())
	}
	assert.T(t, l != nil)
	go l.Start()
	for i := 0; i < 10; i++ {
		resp, err = c.SendRecv([]byte("PING"))
		if err != nil {
			log.Error("Iter: %d %s", i, err.Error())
		}
		assert.T(t, err == nil)
		assert.Equal(t, []byte("PING"), resp)
	}
}

func TestBreakingRetryableError(t *testing.T) {
	assert.T(t, retryableError(errors.New("Sup")) == false)
	assert.T(t, retryableError(&net.OpError{Err: syscall.EPIPE}) == true)
}

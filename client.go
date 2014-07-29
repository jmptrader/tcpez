// Client is a basic implementation of a tcpez protocol Client with connection pooling and
// Pipelined requests. This will work out of the bat with any tcpez server, but any message
// encoding should be handled at a higher level of abstraction
package tcpez

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/fatih/pool"
	"github.com/golang/glog"
	"io"
	"math/rand"
	"net"
	"sync"
)

type Client struct {
	pool      *pool.Pool
	Addresses []string
}

// Create a new Client to connect and load balance between a pool of addresses
// given as a slice of strings in "host:port" format (the same format that net.Dial
// uses for the underlying connections. poolInit and poolMax set the initial connection
// pool and the maxPool sizes. If you're not using this client across different
// goroutines then these settings can be left at 1.
func NewClient(addresses []string, poolInit, poolMax int) (client *Client) {
	pool, err := pool.New(poolInit, poolMax, func() (net.Conn, error) {
		return net.Dial("tcp", addresses[rand.Intn(len(addresses))])
	})
	if err != nil {
		glog.Fatal(err)
	}
	return &Client{pool: pool, Addresses: addresses}
}

// Pipeline returns a new pipeline for sending requests. These requests are kept in
// an internal buffer until flushed to the connection using .Flush(). Flush() then
// returns a slice of the responses in the order they were sent.
//
//        c := tcpez.NewClient([]string{"localserver:2222"}, 3, 3)
//        p := c.Pipeline()
//        p.Send([]byte{"PING1"})
//        p.Send([]byte{"PING2"})
//        responses, _ := p.Flush()
//        responses //=> [[]byte{"PONG1"}, []byte{"PONG2"}]
//
func (c *Client) Pipeline() *Pipeline {
	return NewPipeline(c)
}

// SendRecv is the basic mechanism for making a request and getting
// a response from a tcpez Server. The connection will block until
// receiving a response.
//
//        c := tcpez.NewClient([]string{"localserver:2222"}, 3, 3)
//        resp, err := c.SendRecv([]byte{"PING"})
//        resp //=> []byte{"PONG"}
//
func (c *Client) SendRecv(req []byte) (res []byte, err error) {
	conn, _, err := c.sendRequest(req)
	defer c.pool.Put(conn)
	if err != nil {
		return nil, err
	}
	return c.readResponse(conn)
}

func (c *Client) sendRequest(data []byte) (conn net.Conn, length int, err error) {
	glog.V(3).Infof("Client Request: %s", data)
	conn, err = c.pool.Get()
	length, err = writeDataWithLength(data, conn)
	return conn, length, err
}

func (c *Client) readResponse(conn net.Conn) (response []byte, err error) {
	response, err = readDataWithLength(conn)
	if err != nil {
		return nil, err
	}
	glog.V(3).Infof("Server Response: %s", response)
	return
}

func writeDataWithLength(data []byte, buf io.Writer) (length int, err error) {
	err = binary.Write(buf, binary.BigEndian, int32(len(data)))
	if err != nil {
		return 0, err
	}
	n, err := buf.Write(data)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func readDataWithLength(conn io.Reader) (data []byte, err error) {
	var size int32
	err = binary.Read(conn, binary.BigEndian, &size)
	if err != nil {
		return nil, err
	}
	data = make([]byte, size)
	_, err = io.ReadFull(conn, data)
	if err != nil {
		return nil, err
	}
	return
}

type Pipeline struct {
	client *Client
	buf    *bytes.Buffer
	count  int32
	sync.Mutex
}

// Initializes a new Pipeline for Client. Should use client.Pipeline() if possible.
func NewPipeline(c *Client) (p *Pipeline) {
	p = &Pipeline{client: c}
	p.buf = bytes.NewBuffer(nil)
	return p
}

// Send a request over the Pipeline. This collects the request data in an internal
// buffer before flushing to the actual connection. This means the request wont actual
// be delivered until Flush() is called
func (p *Pipeline) Send(req []byte) error {
	p.Lock()
	defer p.Unlock()
	p.count++
	_, err := writeDataWithLength(req, p.buf)
	return err
}

// Flush actually delivers all the buffered request data to the connection. It then
// blocks waiting for all the responses from the server. These requests are returned
// in order and stored in an slice and returned as responses
func (p *Pipeline) Flush() (responses [][]byte, err error) {
	conn, err := p.client.pool.Get()
	defer p.client.pool.Put(conn)
	if err != nil {
		return nil, err
	}
	// Write the initial byte as -the count of the messages
	err = binary.Write(conn, binary.BigEndian, int32(-p.count))
	if err != nil {
		return nil, err
	}
	// Flush the whole buffer
	conn.Write(p.buf.Bytes())
	var responseCount int32
	err = binary.Read(conn, binary.BigEndian, &responseCount)
	if err != nil {
		return nil, err
	}
	if -responseCount != p.count {
		return nil, errors.New(fmt.Sprintf("Mismatched number of responses for pipeline request. Expected %d, got %d", p.count, -responseCount))
	}
	responses = make([][]byte, -responseCount)
	for i := int32(0); i < -responseCount; i++ {
		responses[i], err = readDataWithLength(conn)
	}
	return
}

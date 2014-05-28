// proto_server is a working (and useful) example of creating a higher level
// abstraction on top of tcpez.Server. A ProtoServer is one that accepts requests
// and returns responses encoded as Protocol Buffers. The types that define the
// request and response and the handling of the actual request are left to the
// programmer. However, with some simple configuration ProtoServer can handle the
// protobuf marshalling/unmarshalling for you, allowing you to just define the business
// logic of the application. See examples/req_rep for a simple example.
package tcpez

import (
	proto "code.google.com/p/goprotobuf/proto"
)

// ProtoInitializerFunc defines the interface for the function that has to
// return an initialized ProtocolBuffer object in the type that you've defined
// as being the "Request" schema.
type ProtoInitializerFunc func() proto.Message

// ProtoHandlerFunc is the functional equivilent of the Respond() method for
// the RequestHandler in the base tcpez.Server. Instead of taking the request and 
// sending the response as []byte, it has the request as an initialized and parsed
// protobuf and returns the response in the protocol buffer schema that represents
// the response. This is then marshalled into a []byte before being sent back 
// to the client
type ProtoHandlerFunc func(req proto.Message, span *Span) proto.Message

type ProtoServer struct {
	requestInitializer ProtoInitializerFunc
	handler            ProtoHandlerFunc
}

// Respond() does not need to be called by any outside objects, it is the method
// that fullfills the RequestHandler interface for the tcpez.Server. It uses the
// ProtoInitializerFunc and ProtoHandlerFunc to handle the actual request after
// marshalling and unmarshalling the request and response objects
func (s *ProtoServer) Respond(req []byte, span *Span) (res []byte, err error) {
	request := s.requestInitializer()
	span.Start("pb.parse")
	err = proto.Unmarshal(req, request)
	if err != nil {
		return nil, err
	}
	span.Start("pb.response")
	span.Finish("pb.parse")
	response := s.handler(request, span)
	span.Finish("pb.response")
	span.Start("pb.encode")
	res, err = proto.Marshal(response)
	span.Finish("pb.encode")
	return
}

// NewProtoServer intializes a tcpez.Server with a ProtoInitializerFunc and a ProtoHandlerFunc. A normal tcpez.Server
// is returned (meaning you still have to call .Start() on it)
// 
// 	protoFunc := tcpez.ProtoInitializerFunc(func() proto.Message {
// 	        // we call this Request here, but its whatever schema YOUR 
// 	        // request is in
// 		return new(Request)
// 	})
// 
// 	handlerFunc := tcpez.ProtoHandlerFunc(func(req proto.Message, span *tcpez.Span) (res proto.Message) {
//              // initialize a new Response object which will be returned at the end of the handler
// 		response := new(Response)
//              // We need to do a type assertion here which turns the proto.Message interface into
//              // a struct of our request type that we've defined in our proto schema
// 		request := req.(*Request)
// 		// Do whatever logic we need to with the request and response
// 		//...
// 		return response
// 	})
//      // Initialize the actual server
// 	server := tcpez.NewProtoServer(":2222", protoFunc, handlerFunc)
// 	go server.Start()
//
func NewProtoServer(address string, requestInitializer ProtoInitializerFunc, handler ProtoHandlerFunc) (s *Server) {
	return NewServer(address, &ProtoServer{requestInitializer, handler})
}

// Code generated by protoc-gen-go.
// source: reqrep.proto
// DO NOT EDIT!

package reqrep

import proto "github.com/golang/protobuf/proto"
import json "encoding/json"
import math "math"

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
}

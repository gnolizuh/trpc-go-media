//
// Copyright [2024] [https://github.com/gnolizuh]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package rtmp

import (
	"io"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/transport"
)

func init() {
	codec.Register(ProtocolName, DefaultServerCodec, DefaultClientCodec)
	transport.RegisterFramerBuilder(ProtocolName, DefaultFramerBuilder)
}

// default codec
var (
	DefaultServerCodec   = &ServerCodec{}
	DefaultClientCodec   = &ClientCodec{}
	DefaultFramerBuilder = &FramerBuilder{}
)

const (
	ProtocolName = "rtmp" // protocol name
)

// ServerCodec is an implementation of codec.Codec.
// Used for rtmp serverside codec.
type ServerCodec struct {
}

// Decode implements codec.Codec.
// It decodes the reqBuf and updates the msg that already initialized by service handler.
func (s *ServerCodec) Decode(msg codec.Msg, reqBuf []byte) ([]byte, error) {
	return nil, nil
}

// Encode implements codec.Codec.
// It encodes the rspBody to binary data and returns it to client.
func (s *ServerCodec) Encode(msg codec.Msg, rspBody []byte) ([]byte, error) {
	return nil, nil
}

// ClientCodec is an implementation of codec.Codec.
// Used for trpc clientside codec.
type ClientCodec struct {
	defaultCaller string // trpc.app.server.service
	requestID     uint32 // global unique request id
}

// Encode implements codec.Codec.
// It encodes reqBody into binary data. New msg will be cloned by client stub.
func (c *ClientCodec) Encode(msg codec.Msg, reqBody []byte) (reqBuf []byte, err error) {
	return nil, nil
}

// Decode implements codec.Codec.
// It decodes rspBuf into rspBody.
func (c *ClientCodec) Decode(msg codec.Msg, rspBuf []byte) (rspBody []byte, err error) {
	return nil, nil
}

// FramerBuilder is an implementation of codec.FramerBuilder.
// Used for trpc protocol.
type FramerBuilder struct{}

// New implements codec.FramerBuilder.
func (fb *FramerBuilder) New(reader io.Reader) codec.Framer {
	return &framer{
		reader: reader,
	}
}

// framer is an implementation of codec.Framer.
// Used for trpc protocol.
type framer struct {
	reader io.Reader
}

// ReadFrame implements codec.Framer.
func (f *framer) ReadFrame() ([]byte, error) {
	return nil, nil
}

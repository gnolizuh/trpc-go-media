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
	"math"
	"runtime"
	"time"
)

type LimitType uint8

const (
	defaultReaderBufferSize = 4 * 1024
	defaultWriterBufferSize = 4 * 1024
	defaultChunkSize        = 128
	defaultMaxChunkSize     = 0xffffff
)

// ServerTransportOptions is options of the server transport.
type ServerTransportOptions struct {
	IdleTimeout                time.Duration
	KeepAlivePeriod            time.Duration
	ReaderBufferSize           uint32
	WriterBufferSize           uint32
	MaxChunkSize               uint32
	MaxChunkStreams            uint32
	DefaultChunkSize           uint32
	MaxAckWindowSize           uint32
	DefaultAckWindowSize       uint32
	MaxBandwidthWindowSize     uint32
	DefaultBandwidthWindowSize uint32
	DefaultBandwidthLimitType  LimitType
	MaxMessageSize             uint32
	MaxMessageStreams          uint32
	ReusePort                  bool
}

// ServerTransportOption modifies the ServerTransportOptions.
type ServerTransportOption func(*ServerTransportOptions)

// WithIdleTimeout returns an ServerTransportOption which sets the idle connection time, after which it may be closed.
func WithIdleTimeout(timeout time.Duration) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.IdleTimeout = timeout
	}
}

// WithKeepAlivePeriod returns an ServerTransportOption which sets the period to keep TCP connection alive.
func WithKeepAlivePeriod(duration time.Duration) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.KeepAlivePeriod = duration
	}
}

// WithReaderBufferSize returns an ServerTransportOption with set reader buffer size.
func WithReaderBufferSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.ReaderBufferSize = size
	}
}

// WithWriterBufferSize returns an ServerTransportOption with set writer buffer size.
func WithWriterBufferSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.WriterBufferSize = size
	}
}

// WithMaxChunkSize returns an ServerTransportOption with set max chunk size.
func WithMaxChunkSize(max uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.MaxChunkSize = max
	}
}

// WithMaxChunkStreams returns an ServerTransportOption with set max chunk streams.
func WithMaxChunkStreams(max uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.MaxChunkStreams = max
	}
}

// WithDefaultChunkSize returns an ServerTransportOption with set default chunk size.
func WithDefaultChunkSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.DefaultChunkSize = size
	}
}

// WithMaxAckWindowSize returns an ServerTransportOption with set max ack window size.
func WithMaxAckWindowSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.MaxAckWindowSize = size
	}
}

// WithDefaultAckWindowSize returns an ServerTransportOption with set default ack window size.
func WithDefaultAckWindowSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.DefaultAckWindowSize = size
	}
}

// WithMaxBandwidthWindowSize returns an ServerTransportOption with set max bandwidth window size.
func WithMaxBandwidthWindowSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.MaxBandwidthWindowSize = size
	}
}

// WithDefaultBandwidthWindowSize returns an ServerTransportOption with set default bandwidth window size.
func WithDefaultBandwidthWindowSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.DefaultBandwidthWindowSize = size
	}
}

// WithDefaultBandwidthLimitType returns an ServerTransportOption with set default bandwidth limit type.
func WithDefaultBandwidthLimitType(typ LimitType) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.DefaultBandwidthLimitType = typ
	}
}

// WithMaxMessageSize returns an ServerTransportOption with set max message size.
func WithMaxMessageSize(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.MaxMessageSize = size
	}
}

// WithMaxMessageStreams returns an ServerTransportOption with set max message streams.
func WithMaxMessageStreams(size uint32) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.MaxMessageStreams = size
	}
}

// WithReusePort returns a ServerTransportOption which enable reuse port or not.
func WithReusePort(reuse bool) ServerTransportOption {
	return func(options *ServerTransportOptions) {
		options.ReusePort = reuse
		if runtime.GOOS == "windows" {
			options.ReusePort = false
		}
	}
}

func defaultServerTransportOptions() *ServerTransportOptions {
	return &ServerTransportOptions{
		ReaderBufferSize:           defaultReaderBufferSize,
		WriterBufferSize:           defaultWriterBufferSize,
		DefaultChunkSize:           defaultChunkSize,
		MaxChunkSize:               defaultMaxChunkSize,
		MaxChunkStreams:            math.MaxUint32,
		DefaultAckWindowSize:       math.MaxUint32,
		MaxAckWindowSize:           math.MaxUint32,
		DefaultBandwidthWindowSize: math.MaxUint32,
		MaxBandwidthWindowSize:     math.MaxUint32,
		MaxMessageStreams:          math.MaxUint32,
		MaxMessageSize:             defaultMaxChunkSize,
	}
}

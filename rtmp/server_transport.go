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
	"context"
	"errors"
	"fmt"
	"github.com/panjf2000/ants"
	"github.com/valyala/fasthttp/reuseport"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

const defaultTransportName = "rtmp"

func init() {
	transport.RegisterServerTransport(defaultTransportName, DefaultServerTransport)
}

var (
	errUnSupportedNetworkType = errors.New("not supported network type")
	errFileIsNotSocket        = errors.New("file is not a socket")
)

var DefaultServerTransport = NewServerTransport(WithReusePort(true))

// NewServerTransport creates a new ServerTransport.
func NewServerTransport(opt ...ServerTransportOption) transport.ServerTransport {
	st := newServerTransport(opt...)
	return &st
}

// newServerTransport creates a new serverTransport.
func newServerTransport(opt ...ServerTransportOption) serverTransport {
	// this is the default option.
	opts := defaultServerTransportOptions()
	for _, o := range opt {
		o(opts)
	}
	addrToConn := make(map[string]*tcpconn)
	return serverTransport{addrToConn: addrToConn, m: &sync.RWMutex{}, opts: opts}
}

// serverTransport is the implementation details of server transport, may be tcp or udp.
type serverTransport struct {
	addrToConn map[string]*tcpconn
	m          *sync.RWMutex
	opts       *ServerTransportOptions
}

// ListenAndServe starts Listening, returns an error on failure.
func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...transport.ListenServeOption) error {
	listenOpts := &transport.ListenServeOptions{}
	for _, opt := range opts {
		opt(listenOpts)
	}

	if listenOpts.Listener != nil {
		return s.listenAndServeStream(ctx, listenOpts)
	}

	// Support simultaneous listening TCP and UDP.
	networks := strings.Split(listenOpts.Network, ",")
	for _, network := range networks {
		listenOpts.Network = network
		switch listenOpts.Network {
		case "tcp", "tcp4", "tcp6", "unix":
			if err := s.listenAndServeStream(ctx, listenOpts); err != nil {
				return err
			}
		case "udp", "udp4", "udp6":
			if err := s.listenAndServePacket(ctx, listenOpts); err != nil {
				return err
			}
		default:
			return fmt.Errorf("server transport: not support network type %s", listenOpts.Network)
		}
	}
	return nil
}

// ---------------------------------stream server-----------------------------------------//

var (
	// listenersMap records the listeners in use in the current process.
	listenersMap = &sync.Map{}
	// inheritedListenersMap record the listeners inherited from the parent process.
	// A key(host:port) may have multiple listener fds.
	inheritedListenersMap = &sync.Map{}
	// once controls fds passed from parent process to construct listeners.
	once sync.Once
)

// listenAndServeStream starts listening, returns an error on failure.
func (s *serverTransport) listenAndServeStream(ctx context.Context, opts *transport.ListenServeOptions) error {
	if opts.FramerBuilder == nil {
		return errors.New("tcp transport FramerBuilder empty")
	}
	ln, err := s.getTCPListener(opts)
	if err != nil {
		return fmt.Errorf("get tcp listener err: %w", err)
	}
	// We MUST save the raw TCP listener (instead of (*tls.listener) if TLS is enabled)
	// to guarantee the underlying fd can be successfully retrieved for hot restart.
	listenersMap.Store(ln, struct{}{})
	go func() {
		_ = s.serveStream(ctx, ln, opts)
	}()
	return nil
}

func (s *serverTransport) serveStream(ctx context.Context, ln net.Listener, opts *transport.ListenServeOptions) error {
	return s.serveTCP(ctx, ln, opts)
}

// getTCPListener gets the TCP/Unix listener.
func (s *serverTransport) getTCPListener(opts *transport.ListenServeOptions) (listener net.Listener, err error) {
	listener = opts.Listener

	if listener != nil {
		return listener, nil
	}

	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// find the passed listener
		pln, err := getPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}

		listener, ok := pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}
		return listener, nil
	}

	// Reuse port. To speed up IO, the kernel dispatches IO ReadReady events to threads.
	if s.opts.ReusePort && opts.Network != "unix" {
		listener, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("%s reuseport error:%v", opts.Network, err)
		}
	} else {
		listener, err = net.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}
	}

	return listener, nil
}

// ---------------------------------packet server-----------------------------------------//

var errNotFound = errors.New("listener not found")

// inheritListeners stores the listener according to start listenfd and number of listenfd passed
// by environment variables.
func inheritListeners() {
	firstListenFd, err := strconv.ParseUint(os.Getenv(transport.EnvGraceFirstFd), 10, 32)
	if err != nil {
		log.Errorf("invalid %s, error: %v", transport.EnvGraceFirstFd, err)
	}

	num, err := strconv.ParseUint(os.Getenv(transport.EnvGraceRestartFdNum), 10, 32)
	if err != nil {
		log.Errorf("invalid %s, error: %v", transport.EnvGraceRestartFdNum, err)
	}

	for fd := firstListenFd; fd < firstListenFd+num; fd++ {
		file := os.NewFile(uintptr(fd), "")
		listener, addr, err := fileListener(file)
		_ = file.Close()
		if err != nil {
			log.Errorf("get file listener error: %v", err)
			continue
		}

		key := addr.Network() + ":" + addr.String()
		v, ok := inheritedListenersMap.LoadOrStore(key, []interface{}{listener})
		if ok {
			listeners := v.([]interface{})
			listeners = append(listeners, listener)
			inheritedListenersMap.Store(key, listeners)
		}
	}
}

func fileListener(file *os.File) (interface{}, net.Addr, error) {
	// Check file status.
	fin, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	// Is this a socket fd.
	if fin.Mode()&os.ModeSocket == 0 {
		return nil, nil, errFileIsNotSocket
	}

	// tcp, tcp4 or tcp6.
	if listener, err := net.FileListener(file); err == nil {
		return listener, listener.Addr(), nil
	}

	// udp, udp4 or udp6.
	if packetConn, err := net.FilePacketConn(file); err == nil {
		return packetConn, packetConn.LocalAddr(), nil
	}

	return nil, nil, errUnSupportedNetworkType
}

func getPassedListener(network, address string) (interface{}, error) {
	once.Do(inheritListeners)

	key := network + ":" + address
	v, ok := inheritedListenersMap.Load(key)
	if !ok {
		return nil, errNotFound
	}

	listeners := v.([]interface{})
	if len(listeners) == 0 {
		return nil, errNotFound
	}

	ln := listeners[0]
	listeners = listeners[1:]
	if len(listeners) == 0 {
		inheritedListenersMap.Delete(key)
	} else {
		inheritedListenersMap.Store(key, listeners)
	}

	return ln, nil
}

// ---------------------------------packet server-----------------------------------------//

// listenAndServePacket starts listening, returns an error on failure.
func (s *serverTransport) listenAndServePacket(ctx context.Context, opts *transport.ListenServeOptions) error {
	return nil
}

// getUDPListener gets UDP listener.
func (s *serverTransport) getUDPListener(opts *transport.ListenServeOptions) (udpConn net.PacketConn, err error) {
	return nil, nil
}

func (s *serverTransport) servePacket(ctx context.Context, rwc net.PacketConn, pool *ants.PoolWithFunc,
	opts *transport.ListenServeOptions) error {
	switch rwc := rwc.(type) {
	case *net.UDPConn:
		return s.serveUDP(ctx, rwc, pool, opts)
	default:
		return errors.New("transport not support PacketConn impl")
	}
}

// ------------------------ tcp/udp connection structures ----------------------------//

func (s *serverTransport) newConn(ctx context.Context, opts *transport.ListenServeOptions) *conn {
	idleTimeout := opts.IdleTimeout
	if s.opts.IdleTimeout > 0 {
		idleTimeout = s.opts.IdleTimeout
	}
	return &conn{
		ctx:         ctx,
		handler:     opts.Handler,
		idleTimeout: idleTimeout,
	}
}

// conn is the struct of connection which is established when server receive a client connecting
// request.
type conn struct {
	ctx         context.Context
	cancelCtx   context.CancelFunc
	idleTimeout time.Duration
	lastVisited time.Time
	handler     transport.Handler
}

func (c *conn) handle(ctx context.Context, req []byte) ([]byte, error) {
	return c.handler.Handle(ctx, req)
}

func (c *conn) handleClose(ctx context.Context) error {
	if closeHandler, ok := c.handler.(transport.CloseHandler); ok {
		return closeHandler.HandleClose(ctx)
	}
	return nil
}

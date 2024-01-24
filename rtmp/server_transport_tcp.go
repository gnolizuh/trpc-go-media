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
	"github.com/gnolizuh/trpc-go-multimedia/rtmp/internal/addrutil"
	"github.com/gnolizuh/trpc-go-multimedia/rtmp/internal/frame"
	"github.com/gnolizuh/trpc-go-multimedia/rtmp/internal/report"
	"github.com/gnolizuh/trpc-go-multimedia/rtmp/internal/writev"
	"github.com/panjf2000/ants"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"time"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/transport"
)

type handleParam struct {
	req   []byte
	c     *tcpconn
	start time.Time
}

func (p *handleParam) reset() {
	p.req = nil
	p.c = nil
	p.start = time.Time{}
}

var handleParamPool = &sync.Pool{
	New: func() interface{} { return new(handleParam) },
}

func createRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		param, ok := args.(*handleParam)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		report.TCPServerAsyncGoroutineScheduleDelay.Set(float64(time.Since(param.start).Microseconds()))
		if param.c == nil {
			log.Tracef("routine pool tcpconn is nil, shouldn't happen!")
			return
		}
		param.c.handleSync(param.req)
		param.reset()
		handleParamPool.Put(param)
	})
	if err != nil {
		log.Tracef("routine pool create error:%v", err)
		return nil
	}
	return pool
}

func (s *serverTransport) serveTCP(ctx context.Context, ln net.Listener, opts *transport.ListenServeOptions) error {
	var once sync.Once
	closeListener := func() {
		_ = ln.Close()
	}
	defer once.Do(closeListener)
	// Create a goroutine to watch ctx.Done() channel.
	// Once Server.Close(), TCP listener should be closed immediately and won't accept any new connection.
	go func() {
		<-ctx.Done()
		log.Tracef("recv server close event")
		once.Do(closeListener)
	}()
	// Create a goroutine pool if ServerAsync enabled.
	var pool *ants.PoolWithFunc
	if opts.ServerAsync {
		pool = createRoutinePool(opts.Routines)
	}
	for tempDelay := time.Duration(0); ; {
		rwc, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				tempDelay = doTempDelay(tempDelay)
				continue
			}
			select {
			case <-ctx.Done(): // If this error is triggered by the user, such as during a restart,
				return err // it is possible to directly return the error, causing the current listener to exit.
			default:
				// Restricted access to the internal/poll.ErrNetClosing type necessitates comparing a string literal.
				const accept, closeError = "accept", "use of closed network connection"
				const msg = "the server transport, listening on %s, encountered an error: %+v; this error was handled" +
					" gracefully by the framework to prevent abnormal termination, serving as a reference for" +
					" investigating acceptance errors that can't be filtered by the Temporary interface"
				if e, ok := err.(*net.OpError); ok && e.Op == accept && strings.Contains(e.Err.Error(), closeError) {
					log.Infof("listener with address %s is closed", ln.Addr())
					return err
				}
				log.Errorf(msg, ln.Addr(), err)
				continue
			}
		}
		tempDelay = 0
		if tcpConn, ok := rwc.(*net.TCPConn); ok {
			if err := tcpConn.SetKeepAlive(true); err != nil {
				log.Tracef("tcp conn set keepalive error:%v", err)
			}
			if s.opts.KeepAlivePeriod > 0 {
				if err := tcpConn.SetKeepAlivePeriod(s.opts.KeepAlivePeriod); err != nil {
					log.Tracef("tcp conn set keepalive period error:%v", err)
				}
			}
		}
		tc := &tcpconn{
			conn:        s.newConn(ctx, opts),
			rwc:         rwc,
			fr:          opts.FramerBuilder.New(codec.NewReader(rwc)),
			remoteAddr:  rwc.RemoteAddr(),
			localAddr:   rwc.LocalAddr(),
			serverAsync: opts.ServerAsync,
			writev:      opts.Writev,
			st:          s,
			pool:        pool,
		}
		// Start goroutine sending with writev.
		if tc.writev {
			tc.buffer = writev.NewBuffer()
			tc.closeNotify = make(chan struct{}, 1)
			tc.buffer.Start(tc.rwc, tc.closeNotify)
		}
		// To avoid over writing packages, checks whether should we copy packages by Framer and
		// some other configurations.
		tc.copyFrame = frame.ShouldCopy(opts.CopyFrame, tc.serverAsync, codec.IsSafeFramer(tc.fr))
		key := addrutil.AddrToKey(tc.localAddr, tc.remoteAddr)
		s.m.Lock()
		s.addrToConn[key] = tc
		s.m.Unlock()
		go tc.serve()
	}
}

func doTempDelay(tempDelay time.Duration) time.Duration {
	if tempDelay == 0 {
		tempDelay = 5 * time.Millisecond
	} else {
		tempDelay *= 2
	}
	if maxDelay := 1 * time.Second; tempDelay > maxDelay {
		tempDelay = maxDelay
	}
	time.Sleep(tempDelay)
	return tempDelay
}

// tcpconn is the connection which is established when server accept a client connecting request.
type tcpconn struct {
	*conn
	rwc         net.Conn
	fr          codec.Framer
	localAddr   net.Addr
	remoteAddr  net.Addr
	serverAsync bool
	writev      bool
	copyFrame   bool
	closeOnce   sync.Once
	st          *serverTransport
	pool        *ants.PoolWithFunc
	buffer      *writev.Buffer
	closeNotify chan struct{}
}

// close closes socket and cleans up.
func (c *tcpconn) close() {
	c.closeOnce.Do(func() {
		// Send error msg to handler.
		ctx, msg := codec.WithNewMessage(context.Background())
		msg.WithLocalAddr(c.localAddr)
		msg.WithRemoteAddr(c.remoteAddr)
		e := &errs.Error{
			Type: errs.ErrorTypeFramework,
			Code: errs.RetServerSystemErr,
			Desc: "trpc",
			Msg:  "Server connection closed",
		}
		msg.WithServerRspErr(e)
		// The connection closing message is handed over to handler.
		if err := c.conn.handleClose(ctx); err != nil {
			log.Trace("transport: notify connection close failed", err)
		}
		// Notify to stop writev sending goroutine.
		if c.writev {
			close(c.closeNotify)
		}

		// Remove cache in server stream transport.
		key := addrutil.AddrToKey(c.localAddr, c.remoteAddr)
		c.st.m.Lock()
		delete(c.st.addrToConn, key)
		c.st.m.Unlock()

		// Finally, close the socket connection.
		c.rwc.Close()
	})
}

func (c *tcpconn) serve() {
	defer c.close()
	for {
		// Check if upstream has closed.
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if c.idleTimeout > 0 {
			now := time.Now()
			// SetReadDeadline has poor performance, so, update timeout every 5 seconds.
			if now.Sub(c.lastVisited) > 5*time.Second {
				c.lastVisited = now
				err := c.rwc.SetReadDeadline(now.Add(c.idleTimeout))
				if err != nil {
					log.Trace("transport: tcpconn SetReadDeadline fail ", err)
					return
				}
			}
		}

		req, err := c.fr.ReadFrame()
		if err != nil {
			if err == io.EOF {
				report.TCPServerTransportReadEOF.Incr() // client has closed the connections.
				return
			}
			// Server closes the connection if client sends no package in last idle timeout.
			if e, ok := err.(net.Error); ok && e.Timeout() {
				report.TCPServerTransportIdleTimeout.Incr()
				return
			}
			report.TCPServerTransportReadFail.Incr()
			log.Trace("transport: tcpconn serve ReadFrame fail ", err)
			return
		}
		report.TCPServerTransportReceiveSize.Set(float64(len(req)))
		// if framer is not concurrent safe, copy the data to avoid over writing.
		if c.copyFrame {
			reqCopy := make([]byte, len(req))
			copy(reqCopy, req)
			req = reqCopy
		}

		c.handle(req)
	}
}

func (c *tcpconn) handle(req []byte) {
	if !c.serverAsync || c.pool == nil {
		c.handleSync(req)
		return
	}

	// Using sync.pool to dispatch package processing goroutine parameters can reduce a memory
	// allocation and slightly promote performance.
	args := handleParamPool.Get().(*handleParam)
	args.req = req
	args.c = c
	args.start = time.Now()
	if err := c.pool.Invoke(args); err != nil {
		report.TCPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport: tcpconn serve routine pool put job queue fail ", err)
		c.handleSyncWithErr(req, errs.ErrServerRoutinePoolBusy)
	}
}

func (c *tcpconn) handleSync(req []byte) {
	c.handleSyncWithErr(req, nil)
}

func (c *tcpconn) handleSyncWithErr(req []byte, e error) {

}

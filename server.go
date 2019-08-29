// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Server defines parameters for running a TFTP server. The zero value for Server is a valid configuration.
type Server struct {
	Addr string // Addr is the UDP address to listen on, ":tftp" if empty.

	Root string // Root is the path to the directory in the Operating System which holds the files to be served.

	// IdleTimeout is the maximum amount of time to wait for the
	// next packet from a client connection. The connection's
	// deadline is reset after sending a packet to the client.
	// The default value is 5 seconds.
	IdleTimeout time.Duration // Go 1.8

	// ErrorLog specifies an optional logger for errors accepting
	// connections, unexpected behavior from handlers, and
	// underlying FileSystem errors.
	// If nil, logging is done via the log package's standard logger.
	ErrorLog *log.Logger // Go 1.3

	inShutdown atomicBool // accessed atomically (non-zero means we're in Shutdown)

	mu       sync.Mutex
	doneChan chan struct{}
}

// ListenAndServe listens on the UDP network at address ":tftp" and then
// serves requests on incoming connections from the path at root.
//
// ListenAndServe always returns a non-nil error.
func ListenAndServe(root string) error {
	server := &Server{Addr: ":tftp", Root: root}
	return server.ListenAndServe()
}

// ListenAndServe listens on the UDP network address srv.Addr and then
// calls Serve to handle requests on incoming connections.
//
// If srv.Addr is blank, ":tftp" is used.
//
// ListenAndServe always returns a non-nil error. After Shutdown or Close,
// the returned error is ErrServerClosed.
func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}

	if err := os.Chdir(srv.Root); err != nil {
		return err
	}

	addr := srv.Addr
	if addr == "" {
		addr = ":tftp"
	}
	conn, err := srv.newConn(addr)
	if err != nil {
		return err
	}
	return srv.serve(conn)
}

func (srv *Server) Close() error {
	/* TODO: Implement func (srv *Server) Close() error */
	return errors.New("TODO: Implement func (srv *Server) Close() error")
}

// serve accepts incoming connections on the PacketConn conn, creating a
// new service goroutine for each. The service goroutines read requests and
// then call srv.Handler to reply to them. TODO: Design - What is my equivalent of an srv.Handler?
//
// Serve always returns a non-nil error and closes conn.
// After Shutdown or Close, the returned error is ErrServerClosed.
func (srv *Server) serve(tftpConn *conn) error {
	baseCtx := context.Background()                          // base is always background, per Issue 16220
	ctx := context.WithValue(baseCtx, ServerContextKey, srv) // TODO: Design - Do I really need this context value or Context keys?
	///////////

	tftpConn.rwc = &onceClosePacketConn{PacketConn: tftpConn.rwc}
	defer tftpConn.rwc.Close() /* TODO: Later - How do handle the error of a deferred function? */
	for {
		request, clientAddr, err := tftpConn.readWithRetry(ctx)
		if err != nil {
			return err
		}

		// :0 assigns a new ephemeral port from the OS, this becomes the server-side Transfer ID (TID) of the connection
		clientConn, err := srv.newConn("127.0.0.1:0")
		if err != nil {
			return err
		}
		go clientConn.serve(ctx, clientAddr, request)
	}
}

type atomicBool int32

func (b *atomicBool) isSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue()    { atomic.StoreInt32((*int32)(b), 1) }

func (srv *Server) shuttingDown() bool {
	return srv.inShutdown != 0
}

// ErrServerClosed is returned by the Server's serve and
// ListenAndServe methods after a call to Shutdown or Close.
var ErrServerClosed = errors.New("tftp: Server closed")

// onceClosePacketConn wraps a net.PacketConn, protecting it from
// multiple Close calls.
type onceClosePacketConn struct {
	net.PacketConn
	once     sync.Once
	closeErr error
}

func (oc *onceClosePacketConn) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceClosePacketConn) close() { oc.closeErr = oc.PacketConn.Close() }

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

var (
	// ServerContextKey is a context key. It can be used in TFTP
	// handlers with context.WithValue to access the server that
	// started the handler. The associated value will be of
	// type *Server. TODO: Design - What is my equivalent of an srv.Handler?
	ServerContextKey = &contextKey{"tftp-server"}
)

// bufferSize defines the size of buffer used to listen for TFTP read and write requests. This accomodates the
// standard Ethernet MTU blocksize (1500 bytes) minus headers of TFTP (4 bytes), UDP (8 bytes) and IP (20 bytes).
const bufferSize = 1468

func (srv *Server) getDoneChan() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.getDoneChanLocked()
}

func (srv *Server) getDoneChanLocked() chan struct{} {
	if srv.doneChan == nil {
		srv.doneChan = make(chan struct{})
	}
	return srv.doneChan
}

func (srv *Server) closeDoneChanLocked() {
	ch := srv.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by srv.mu.
		close(ch)
	}
}

func (srv *Server) logf(format string, args ...interface{}) {
	if srv.ErrorLog != nil {
		srv.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

//////////////////////////////////////////////////

// A conn represents the server side of an TFTP connection.
type conn struct {
	// server is the server on which the connection arrived.
	// Immutable; never nil.
	server *Server

	// cancelCtx cancels the connection-level context.
	cancelCtx context.CancelFunc

	// rwc is the underlying network connection.
	// This is never wrapped by other types and is the value given out
	// to CloseNotifier callers.
	rwc net.PacketConn

	buffer []byte // buffer holds the data last read from rwc

	localAddr net.Addr // address from which the handler is serving the connection
}

// Create new connection from addr.
func (srv *Server) newConn(addr string) (*conn, error) {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}
	c := &conn{
		server:    srv,
		rwc:       pc,
		buffer:    make([]byte, bufferSize),
		localAddr: pc.LocalAddr(),
	}
	return c, nil
}

func (c *conn) readWithRetry(ctx context.Context) (data []byte, addr net.Addr, err error) {
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		n, addr, err := c.rwc.ReadFrom(c.buffer)
		if err != nil {
			select {
			case <-c.server.getDoneChan():
				data := make([]byte, n)
				copy(data, c.buffer[:n])
				return data, addr, ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				c.server.logf("tftp: Request error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue // retry read
			}
			data := make([]byte, n)
			copy(data, c.buffer[:n])
			return data, addr, err // read failure
		}
		data := make([]byte, n)
		copy(data, c.buffer[:n])
		return data, addr, nil // read success
	}
}

func (c *conn) serve(ctx context.Context, clientAddr net.Addr, req []byte) {
	// http Server's CODE
	ctx, cancelCtx := context.WithCancel(ctx)
	c.cancelCtx = cancelCtx
	defer cancelCtx()
	// http Server's CODE

	// Ben's CODE
	/* FIXME: This should become something like:
	- make a new client struct
	- keep sending packets from this connection to the client's handler in a loop
	*/
	if err := clientConn.assignTID(); err != nil {
		srv.logf("tftp: Assign TID to connection at %v resulted in error: %v", clientConn.remoteAddr, err)
	}

	buffer := make([]byte, bufferSize)
	for {
		n, addr, err := c.rwc.ReadFrom(buffer)
		if err != nil {
			c.server.logf("tftp: Serve connection error on read from packet connection:\n\tclientTID: %v\n\tserverTID: %v\n\terror: %v",
				c.remoteTID, c.localTID, err)

		}

		// go s.Handler.ServeTFTP(r ResponseWriter, *Request) // TODO 3
		go s.Handler.ServeTFTP(s.pc, &Request{s.clients, addr, buffer[:n]})
	}
	// Ben's CODE

	// http Server's CODE
	for {
		w, err := c.readRequest(ctx)
		if c.r.remain != c.server.initialReadLimitSize() {
			// If we read any bytes off the wire, we're active.
			c.setState(c.rwc, StateActive)
		}
		if err != nil {
			const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"

			if err == errTooLarge {
				// Their HTTP client may or may not be
				// able to read this if we're
				// responding to them and hanging up
				// while they're still writing their
				// request. Undefined behavior.
				const publicErr = "431 Request Header Fields Too Large"
				fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
				c.closeWriteAndWait()
				return
			}
			if isCommonNetReadError(err) {
				return // don't reply
			}

			publicErr := "400 Bad Request"
			if v, ok := err.(badRequestError); ok {
				publicErr = publicErr + ": " + string(v)
			}

			fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
			return
		}

		// Expect 100 Continue support
		req := w.req
		if req.expectsContinue() {
			if req.ProtoAtLeast(1, 1) && req.ContentLength != 0 {
				// Wrap the Body reader with one that replies on the connection
				req.Body = &expectContinueReader{readCloser: req.Body, resp: w}
			}
		} else if req.Header.get("Expect") != "" {
			w.sendExpectationFailed()
			return
		}

		c.curReq.Store(w)

		if requestBodyRemains(req.Body) {
			registerOnHitEOF(req.Body, w.conn.r.startBackgroundRead)
		} else {
			w.conn.r.startBackgroundRead()
		}

		// HTTP cannot have multiple simultaneous active requests.[*]
		// Until the server replies to this request, it can't read another,
		// so we might as well run the handler in this goroutine.
		// [*] Not strictly true: HTTP pipelining. We could let them all process
		// in parallel even if their responses need to be serialized.
		// But we're not going to implement HTTP pipelining because it
		// was never deployed in the wild and the answer is HTTP/2.
		serverHandler{c.server}.ServeHTTP(w, w.req)
		w.cancelCtx()
		if c.hijacked() {
			return
		}
		w.finishRequest()
		if !w.shouldReuseConnection() {
			if w.requestBodyLimitHit || w.closedRequestBodyEarly() {
				c.closeWriteAndWait()
			}
			return
		}
		c.setState(c.rwc, StateIdle)
		c.curReq.Store((*response)(nil))

		if !w.conn.server.doKeepAlives() {
			// We're in shutdown mode. We might've replied
			// to the user without "Connection: close" and
			// they might think they can send another
			// request, but such is life with HTTP/1.1.
			return
		}

		if d := c.server.idleTimeout(); d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
			if _, err := c.bufr.Peek(4); err != nil {
				return
			}
		}
		c.rwc.SetReadDeadline(time.Time{})
	}
	// http Server's CODE
}

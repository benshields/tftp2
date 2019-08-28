// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import (
	"context"
	"errors"
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
	conn, err := net.ListenPacket("udp", addr)
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
func (srv *Server) serve(conn net.PacketConn) error {
	/* TODO: Implement func (srv *Server) serve(conn net.PacketConn) error */
	conn = &onceClosePacketConn{PacketConn: conn}
	defer conn.Close()

	var tempDelay time.Duration     // how long to sleep on accept failure
	baseCtx := context.Background() // base is always background, per Issue 16220
	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	buffer := make([]byte, bufferSize)
	for {
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
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
				srv.logf("tftp: Request error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0
		// FIXME This is where I'm leaving off for the night. This is where I'll setup a client connection and serve it.
		c := srv.newConn(rw)
		c.setState(c.rwc, StateNew) // before Serve can return
		go c.serve(ctx)
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

func (s *Server) logf(format string, args ...interface{}) {
	if s.ErrorLog != nil {
		s.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

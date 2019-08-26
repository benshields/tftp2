// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import (
	"log"
	"time"
)

// Server defines parameters for running a TFTP server. The zero value for Server is a valid configuration.
type Server struct {
	Addr string // Addr is the UDP address to listen on, ":tftp" if empty

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
}

func (srv *Server) Close() error {
	/* TODO: Implement func (srv *Server) Close() */
	return nil
}

func (srv *Server) ListenAndServe() error {
	/* TODO: Implement func (srv *Server) ListenAndServe() */
	return nil
}

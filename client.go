// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import "net"

// client defines parameters for maintaining a TFTP client connection.
type client struct {
	addr net.UDPAddr // addr is the UDP address at which this client can be reached.

	conn net.UDPConn // conn is the server-assigned connection to serve the client from.

	/* TODO: change this into a tftp.packet */
	lastPacket []byte // lastPacket holds the last packet received from the client in case of re-transmission.

	handler readWriteCloser // handler interfaces with the file that the client is reading from or writing to.
}

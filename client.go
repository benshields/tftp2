// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import (
	"net"
	"strconv"
)

// client defines parameters for maintaining a TFTP client connection.
type client struct {
	conn // the server-assigned connection to serve the client from.

	/* TODO: change this into a tftp.packet */
	lastPacket []byte // lastPacket holds the last packet received from the client in case of re-transmission.

	handler readWriteCloser // handler interfaces with the file that the client is reading from or writing to.

	localTID uint16 // this is the port of localAddr which is the server-side Transfer ID in TFTP

	remoteAddr net.Addr // address at which this client can be reached

	remoteTID uint16 // this is the port of remoteAddr which is the client-side Transfer ID in TFTP
}

// FIXME: This function got ripped out from type *conn, so adapt it to take addresses
// Create a new client for this connection and assign the endpoint ports as the Transfer IDs.
func (c *client) assignTID() error {
	// the local port serves as the connections local TID
	c.localAddr = conn.LocalAddr()
	_, localTIDstr, err := net.SplitHostPort(c.localAddr.String())
	if err != nil {
		return err
	}
	localTID, err := strconv.Atoi(localTIDstr)
	if err != nil {
		return err
	}
	c.localTID = uint16(localTID)

	// the remote port serves as the connections remote TID
	_, remoteTIDstr, err := net.SplitHostPort(c.remoteAddr.String())
	if err != nil {
		return err
	}
	remoteTID, err := strconv.Atoi(remoteTIDstr)
	if err != nil {
		return err
	}
	c.remoteTID = uint16(remoteTID)

	return nil
}

// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import (
	"bufio"
	"io"
	"os"
)

// readWriteCloser is the type that must be implemented by the handler of the client type.
type readWriteCloser io.ReadWriteCloser

// openFlag controls behavior of opening a file with a blockStreamer.
type openFlag int

const (
	// read is a flag to open a file in read-only mode.
	read openFlag = openFlag(os.O_RDONLY)

	// write is a flag to open a file in write-only append mode, creating it if it does not exist.
	write openFlag = openFlag(os.O_CREATE | os.O_APPEND | os.O_WRONLY)
)

// encodingFlag controls encoding / decoding behavior of writing and reading to a file with a blockStreamer.
type encodingFlag int

const (
	// netascii is a flag to handle data according to netascii as described by RFC 764.
	netascii encodingFlag = iota

	// octet is a flag to handle data as-is in 8-bit binary form.
	octet
)

// blockStreamer provides an efficient interface for streaming small,
// block-sized read-only or write-only file operations together.
type blockStreamer struct {
	filename string       // filename is used to open the underlying operating system file.
	openMode openFlag     // openMode controls which type of I/O operation will be streamed; read-only or write-only.
	encoding encodingFlag // encoding controls whether data will be streamed as netascii or not.

	fileReference *os.File          // fileReference will be opened and closed by the Open() & Close() calls to a blockStreamer.
	buffer        *bufio.ReadWriter // buffer serves as the intermediary reader or writer to the fileReference.
}

func newBlockStreamer(filename string, openFlag openFlag, encFlag encodingFlag) *blockStreamer {
	fh := blockStreamer{
		filename,
		openFlag,
		encFlag,
		nil,
		nil}
	return &fh
}

func (fh blockStreamer) Open() error {
	/* TODO: Implement func (fh blockStreamer) open(filename, mode string) error */
	return nil
}

func (fh blockStreamer) Close() error {
	/* TODO: Implement func (fh blockStreamer) close() error */
	return nil
}

func (fh blockStreamer) Read(b []byte) (n int, err error) {
	/* TODO: Implement func (fh blockStreamer) read() ([]byte, error) */
	return 0, nil
}

func (fh blockStreamer) Write(b []byte) (n int, err error) {
	/* TODO: Implement func (fh blockStreamer) write(data []byte) error */
	return 0, nil
}

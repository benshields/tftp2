// Copyright (c) 2019, Benjamin Shields. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tftp2 implements the Trivial File Transfer Protocol as defined in RFC 1350.
package tftp2

import "os"

type fileHandler struct {
	filename      string // TODO LATER maybe make this an *os.File?
	mode          string
	blockSize     int64
	fileReference os.File
}

func NewFileHandler(filename, mode string, blockSize int64) *fileHandler {
	return &fileHandler{filename, mode, blockSize, nil}
}

func (fh fileHandler) open() error {
	/* TODO: Implement func (fh fileHandler) open(filename, mode string) error */
	return nil
}

func (fh fileHandler) close() error {
	/* TODO: Implement func (fh fileHandler) close() error */
	return nil
}

func (fh fileHandler) read() ([]byte, error) {
	/* TODO: Implement func (fh fileHandler) read() ([]byte, error) */
	return nil, nil
}

func (fh fileHandler) write(data []byte) error {
	/* TODO: Implement func (fh fileHandler) write(data []byte) error */
	return nil
}

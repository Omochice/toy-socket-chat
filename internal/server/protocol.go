package server

import (
	"bufio"
	"bytes"
	"net"
)

type protocolType int

const (
	protocolTCP protocolType = iota
	protocolHTTP
)

// detectProtocol peeks at the first bytes to determine protocol type
func detectProtocol(conn net.Conn) (protocolType, *bufio.Reader, error) {
	reader := bufio.NewReader(conn)

	// Peek first 4 bytes to detect HTTP
	// HTTP requests start with: "GET ", "POST", "PUT ", "HEAD", etc.
	// TCP protobuf messages start with binary data
	peek, err := reader.Peek(4)
	if err != nil {
		return protocolTCP, reader, err
	}

	// Check for HTTP methods
	if bytes.HasPrefix(peek, []byte("GET ")) ||
		bytes.HasPrefix(peek, []byte("POST")) ||
		bytes.HasPrefix(peek, []byte("PUT ")) ||
		bytes.HasPrefix(peek, []byte("HEAD")) {
		return protocolHTTP, reader, nil
	}

	// Default to TCP for binary data
	return protocolTCP, reader, nil
}

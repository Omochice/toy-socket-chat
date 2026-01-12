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

	peek, err := reader.Peek(4)
	if err != nil {
		return protocolTCP, reader, err
	}

	if bytes.HasPrefix(peek, []byte("GET ")) ||
		bytes.HasPrefix(peek, []byte("POST")) ||
		bytes.HasPrefix(peek, []byte("PUT ")) ||
		bytes.HasPrefix(peek, []byte("HEAD")) {
		return protocolHTTP, reader, nil
	}

	return protocolTCP, reader, nil
}

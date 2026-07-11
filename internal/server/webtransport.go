package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// WebTransportConnection wraps a WebTransport session and its single
// bidirectional stream so the chat session can be treated as a byte stream,
// exactly like TCP. Framing follows the same assumption as TCP: one Read is
// expected to yield one whole protobuf message.
type WebTransportConnection struct {
	session *webtransport.Session
	stream  *webtransport.Stream
}

// NewWebTransportConnection creates a new WebTransportConnection from an
// upgraded session and the bidirectional stream that carries the chat session.
func NewWebTransportConnection(
	session *webtransport.Session,
	stream *webtransport.Stream,
) *WebTransportConnection {
	return &WebTransportConnection{
		session: session,
		stream:  stream,
	}
}

// RemoteAddr returns the remote address of the underlying QUIC connection.
func (c *WebTransportConnection) RemoteAddr() net.Addr {
	return c.session.RemoteAddr()
}

// Write sends binary data to the client over the bidirectional stream.
func (c *WebTransportConnection) Write(data []byte) (int, error) {
	return c.stream.Write(data)
}

// Read receives binary data from the client over the bidirectional stream.
func (c *WebTransportConnection) Read(buf []byte) (int, error) {
	return c.stream.Read(buf)
}

// Close closes the stream and the session. The session is closed as well as the
// stream so the underlying QUIC connection is torn down and no goroutine (the
// HTTP handler blocked on the session context, or the session's capsule reader)
// lingers after a client disconnects.
func (c *WebTransportConnection) Close() error {
	streamErr := c.stream.Close()
	sessErr := c.session.CloseWithError(0, "")
	return errors.Join(streamErr, sessErr)
}

// SetReadDeadline sets the read deadline on the bidirectional stream.
func (c *WebTransportConnection) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

// startWebTransport binds a WebTransport (HTTP/3 over QUIC) endpoint to the same
// port as the already-listening TCP listener and starts serving in a background
// goroutine tracked by s.wg.
//
// The UDP port is derived from s.listener.Addr() rather than from s.address
// because the TCP listener may have been created with a wildcard port (":0"),
// in which case the real port is only known after the listener is bound.
func (s *Server) startWebTransport() error {
	tcpAddr, ok := s.listener.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("listener address is not TCP: %T", s.listener.Addr())
	}

	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: tcpAddr.IP, Port: tcpAddr.Port})
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", tcpAddr.Port, err)
	}

	h3 := &http3.Server{
		// NextProtos must advertise the HTTP/3 ALPN; webtransport-go passes this
		// TLSConfig straight to the QUIC listener without adding it.
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{*s.tlsCert},
			NextProtos:   []string{http3.NextProtoH3},
		},
		Handler: http.HandlerFunc(s.handleWebTransport),
	}
	// ConfigureHTTP3Server installs the SETTINGS and ConnContext that
	// Server.Upgrade relies on; without it Upgrade cannot find the QUIC
	// connection and the client's WebTransport negotiation fails.
	webtransport.ConfigureHTTP3Server(h3)
	s.wtServer = &webtransport.Server{H3: h3}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Serve always returns a non-nil error; a shutdown-triggered error is
		// expected once Stop closes the server, so only report unexpected ones.
		if err := s.wtServer.Serve(udpConn); err != nil {
			select {
			case <-s.quit:
			default:
				log.Printf("WebTransport server error: %v", err)
			}
		}
	}()

	log.Printf("WebTransport enabled on udp %s", udpConn.LocalAddr().String())
	return nil
}

// handleWebTransport upgrades an incoming HTTP/3 request to a WebTransport
// session, takes over the single bidirectional stream the client opens, and
// registers it as a chat client. It blocks until the session ends because the
// HTTP/3 request stream is the session's control stream: returning early would
// tear the session down while the chat is still in progress.
func (s *Server) handleWebTransport(w http.ResponseWriter, r *http.Request) {
	session, err := s.wtServer.Upgrade(w, r)
	if err != nil {
		log.Printf("WebTransport upgrade failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	stream, err := session.AcceptStream(r.Context())
	if err != nil {
		log.Printf("Failed to accept WebTransport stream: %v", err)
		if closeErr := session.CloseWithError(0, ""); closeErr != nil {
			log.Printf("Error closing WebTransport session: %v", closeErr)
		}
		return
	}

	conn := NewWebTransportConnection(session, stream)
	log.Printf("WebTransport connection from %s", conn.RemoteAddr())
	s.register(conn)

	// register handles the client in a background goroutine that owns conn.
	// Keep the request stream alive until the session is closed (by the client
	// leaving or by Stop) so the session is not torn down prematurely.
	<-session.Context().Done()
}

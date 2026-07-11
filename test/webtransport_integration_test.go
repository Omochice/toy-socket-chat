package test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/omochice/toy-socket-chat/internal/client"
	"github.com/omochice/toy-socket-chat/internal/server"
	"github.com/omochice/toy-socket-chat/pkg/protocol"
)

// generateTestCertificate creates a self-signed ECDSA certificate valid for
// "localhost", 127.0.0.1 and ::1, entirely in memory. It returns the
// tls.Certificate the server presents and a CertPool a WebTransport client can
// use to verify that certificate, so tests need neither temporary files nor
// external tooling.
func generateTestCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// The certificate is added directly to the client's root pool, so it must
		// be usable as a CA to verify itself.
		IsCA:        true,
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(leaf)

	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
		Leaf:        leaf,
	}
	return cert, pool
}

// loopbackAddr rewrites the server's listening address to a dialable loopback
// address. srv.Addr() may report a wildcard host such as "[::]:52801", which is
// not a valid target for a WebTransport client dialing "https://<addr>/". Only
// the port is significant, so the host is replaced with 127.0.0.1.
func loopbackAddr(t *testing.T, addr string) string {
	t.Helper()

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("Failed to split server address %q: %v", addr, err)
	}
	return net.JoinHostPort("127.0.0.1", port)
}

// drainJoinMessages consumes queued non-text messages (join notifications) so a
// subsequent read observes the text message under test rather than an earlier
// join broadcast.
func drainJoinMessages(t *testing.T, c *client.Client) {
	t.Helper()

	for {
		select {
		case <-c.Messages():
		case <-time.After(300 * time.Millisecond):
			return
		}
	}
}

// awaitTextMessage waits for the next text message on the client, skipping any
// non-text (join/leave) messages that may still be queued.
func awaitTextMessage(t *testing.T, c *client.Client) protocol.Message {
	t.Helper()

	for {
		select {
		case msg := <-c.Messages():
			if msg.Type == protocol.MessageTypeText {
				return msg
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for text message")
		}
	}
}

// TestIntegration_WebTransportClientsExchangeMessages verifies that two
// WebTransport clients connected to the same server can exchange a broadcast
// text message.
func TestIntegration_WebTransportClientsExchangeMessages(t *testing.T) {
	cert, pool := generateTestCertificate(t)

	srv := server.New(":0", server.WithTLS(cert))
	go func() {
		_ = srv.Start()
	}()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	serverAddr := loopbackAddr(t, srv.Addr())

	client1 := client.New(serverAddr, "wt-user1", "wt", client.WithRootCAs(pool))
	if err := client1.Connect(); err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer client1.Disconnect()
	if err := client1.Join(); err != nil {
		t.Fatalf("Client 1 failed to join: %v", err)
	}

	client2 := client.New(serverAddr, "wt-user2", "wt", client.WithRootCAs(pool))
	if err := client2.Connect(); err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer client2.Disconnect()
	if err := client2.Join(); err != nil {
		t.Fatalf("Client 2 failed to join: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if count := srv.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	drainJoinMessages(t, client1)
	drainJoinMessages(t, client2)

	testMsg := "Hello from wt-user1"
	if err := client1.SendMessage(testMsg); err != nil {
		t.Fatalf("Client 1 failed to send message: %v", err)
	}

	msg := awaitTextMessage(t, client2)
	if msg.Content != testMsg {
		t.Errorf("Expected message %q, got %q", testMsg, msg.Content)
	}
	if msg.Sender != "wt-user1" {
		t.Errorf("Expected sender %q, got %q", "wt-user1", msg.Sender)
	}
}

// TestIntegration_TCPAndWebTransportClientsBroadcastBothDirections verifies that
// a TCP client and a WebTransport client attached to the same server exchange
// broadcasts in both directions. This exercises the highest-risk path, where two
// distinct listeners (the TCP listener and the QUIC listener) feed a single
// shared client set.
func TestIntegration_TCPAndWebTransportClientsBroadcastBothDirections(t *testing.T) {
	cert, pool := generateTestCertificate(t)

	srv := server.New(":0", server.WithTLS(cert))
	go func() {
		_ = srv.Start()
	}()
	defer srv.Stop()

	time.Sleep(100 * time.Millisecond)

	serverAddr := loopbackAddr(t, srv.Addr())

	tcpClient := client.New(serverAddr, "tcp-user", "tcp")
	if err := tcpClient.Connect(); err != nil {
		t.Fatalf("TCP client failed to connect: %v", err)
	}
	defer tcpClient.Disconnect()
	if err := tcpClient.Join(); err != nil {
		t.Fatalf("TCP client failed to join: %v", err)
	}

	wtClient := client.New(serverAddr, "wt-user", "wt", client.WithRootCAs(pool))
	if err := wtClient.Connect(); err != nil {
		t.Fatalf("WebTransport client failed to connect: %v", err)
	}
	defer wtClient.Disconnect()
	if err := wtClient.Join(); err != nil {
		t.Fatalf("WebTransport client failed to join: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if count := srv.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}

	drainJoinMessages(t, tcpClient)
	drainJoinMessages(t, wtClient)

	tcpToWT := "Hello over TCP"
	if err := tcpClient.SendMessage(tcpToWT); err != nil {
		t.Fatalf("TCP client failed to send message: %v", err)
	}
	if msg := awaitTextMessage(t, wtClient); msg.Content != tcpToWT || msg.Sender != "tcp-user" {
		t.Errorf(
			"WebTransport client received %q from %q, want %q from %q",
			msg.Content,
			msg.Sender,
			tcpToWT,
			"tcp-user",
		)
	}

	wtToTCP := "Hello over WebTransport"
	if err := wtClient.SendMessage(wtToTCP); err != nil {
		t.Fatalf("WebTransport client failed to send message: %v", err)
	}
	if msg := awaitTextMessage(t, tcpClient); msg.Content != wtToTCP || msg.Sender != "wt-user" {
		t.Errorf(
			"TCP client received %q from %q, want %q from %q",
			msg.Content,
			msg.Sender,
			wtToTCP,
			"wt-user",
		)
	}
}

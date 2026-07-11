package main

import (
	"bufio"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/omochice/toy-socket-chat/internal/client"
)

func main() {
	// Parse command-line flags
	serverAddr := flag.String("server", "localhost:8080", "Server address (e.g., localhost:8080)")
	username := flag.String("username", "", "Username for chat")
	protocol := flag.String("protocol", "tcp", "Protocol to use (tcp, ws, or wt)")
	caPath := flag.String(
		"ca",
		"",
		"Path to a PEM CA certificate to trust (only used with -protocol wt)",
	)
	flag.Parse()

	if *username == "" {
		log.Fatal("Username is required. Use -username flag")
	}

	// Validate protocol
	if *protocol != "tcp" && *protocol != "ws" && *protocol != "wt" {
		log.Fatalf("Invalid protocol: %s. Use 'tcp', 'ws', or 'wt'", *protocol)
	}

	// Create client
	c := client.New(*serverAddr, *username, *protocol, buildOptions(*protocol, *caPath)...)

	// Connect to server
	if err := c.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer c.Disconnect()

	log.Printf("Connected to %s as %s", *serverAddr, *username)

	// Send join message
	if err := c.Join(); err != nil {
		log.Fatalf("Failed to join chat: %v", err)
	}

	// Start goroutine to receive and display messages
	go func() {
		for msg := range c.Messages() {
			switch msg.Type {
			case 0: // MessageTypeText
				fmt.Printf("[%s]: %s\n", msg.Sender, msg.Content)
			case 1: // MessageTypeJoin
				fmt.Printf("*** %s joined the chat ***\n", msg.Sender)
			case 2: // MessageTypeLeave
				fmt.Printf("*** %s left the chat ***\n", msg.Sender)
			}
		}
	}()

	// Read from stdin and send messages
	fmt.Println("Type your messages (or 'quit' to exit):")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		if text == "quit" || text == "exit" {
			break
		}

		if err := c.SendMessage(text); err != nil {
			log.Printf("Failed to send message: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}

	// Send leave message before disconnecting
	if err := c.Leave(); err != nil {
		log.Printf("Failed to send leave message: %v", err)
	}

	log.Println("Disconnected from server")
}

// buildOptions translates the CA flag into client options. The CA only affects
// TLS verification for WebTransport, so it is ignored (with a notice) for the
// plaintext tcp and ws protocols rather than failing the whole invocation.
func buildOptions(protocol, caPath string) []client.Option {
	if caPath == "" {
		return nil
	}
	if protocol != "wt" {
		log.Printf(
			"Notice: -ca is only used with -protocol wt; ignoring it for protocol %q",
			protocol,
		)
		return nil
	}

	pemData, err := os.ReadFile(caPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate %q: %v", caPath, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		log.Fatalf("Failed to parse CA certificate from %q", caPath)
	}
	return []client.Option{client.WithRootCAs(pool)}
}

package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/omochice/toy-socket-chat/internal/server"
)

func main() {
	// Parse command-line flags
	port := flag.String("port", ":8080", "Port to listen on (e.g., :8080)")
	certFile := flag.String("cert", "", "Path to TLS certificate PEM file (enables WebTransport)")
	keyFile := flag.String("key", "", "Path to TLS private key PEM file (enables WebTransport)")
	flag.Parse()

	// Both cert and key are required to enable WebTransport; a single one is a
	// misconfiguration rather than a valid TCP/WebSocket-only setup.
	var opts []server.Option
	switch {
	case *certFile != "" && *keyFile != "":
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to load TLS key pair: %v", err)
		}
		opts = append(opts, server.WithTLS(cert))
	case *certFile != "" || *keyFile != "":
		log.Fatal("Both -cert and -key must be provided to enable WebTransport")
	}

	// Create and start server
	srv := server.New(*port, opts...)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s...", *port)
		errChan <- srv.Start()
	}()

	// Wait for either error or shutdown signal
	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
		srv.Stop()
	}

	log.Println("Server stopped")
}

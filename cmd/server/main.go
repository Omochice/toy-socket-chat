package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/omochice/toy-socket-chat/internal/server"
)

func main() {
	// Parse command-line flags
	port := flag.String("port", ":8080", "Port to listen on for both TCP and WebSocket (e.g., :8080)")
	flag.Parse()

	// Create and start unified server on single port
	srv := server.NewUnifiedServer(*port, "")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting unified server on %s...", *port)
		log.Printf("  Accepting both TCP socket and WebSocket connections")
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

	log.Println("Unified server stopped")
}

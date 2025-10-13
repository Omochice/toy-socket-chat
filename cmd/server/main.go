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
	tcpPort := flag.String("tcp", ":8080", "TCP port to listen on (e.g., :8080)")
	wsPort := flag.String("ws", ":8081", "WebSocket port to listen on (e.g., :8081)")
	flag.Parse()

	// Create and start unified server
	srv := server.NewUnifiedServer(*tcpPort, *wsPort)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting unified server...")
		log.Printf("  TCP on %s", *tcpPort)
		log.Printf("  WebSocket on %s", *wsPort)
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

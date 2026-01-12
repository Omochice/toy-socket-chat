package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/omochice/toy-socket-chat/internal/chat"
	"github.com/omochice/toy-socket-chat/internal/transport/tcp"
	"github.com/omochice/toy-socket-chat/internal/transport/ws"
)

func main() {
	tcpPort := flag.String("tcp", ":8080", "TCP port to listen on")
	wsPort := flag.String("ws", ":8081", "WebSocket port to listen on")
	flag.Parse()

	hub := chat.NewHub()

	tcpSrv := tcp.New(*tcpPort, hub)
	wsSrv := ws.New(*wsPort, hub)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	errChan := make(chan error, 2)

	go func() {
		log.Printf("Starting TCP server on %s...", *tcpPort)
		errChan <- tcpSrv.Start()
	}()

	go func() {
		log.Printf("Starting WebSocket server on %s...", *wsPort)
		errChan <- wsSrv.Start()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
		tcpSrv.Stop()
		wsSrv.Stop()
		hub.Stop()
	}

	log.Println("Server stopped")
}

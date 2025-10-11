package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/omochice/tcp-socket/internal/client"
)

func main() {
	// Parse command-line flags
	serverAddr := flag.String("server", "localhost:8080", "Server address (e.g., localhost:8080)")
	username := flag.String("username", "", "Username for chat")
	flag.Parse()

	if *username == "" {
		log.Fatal("Username is required. Use -username flag")
	}

	// Create client
	c := client.New(*serverAddr, *username)

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

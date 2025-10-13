package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/omochice/toy-socket-chat/internal/client"
)

func main() {
	serverAddr := flag.String("server", "ws://localhost:8080", "WebSocket server address (e.g., ws://localhost:8080)")
	username := flag.String("username", "", "Username for chat")
	flag.Parse()

	if *username == "" {
		log.Fatal("Username is required. Use -username flag")
	}

	c := client.NewWebSocketClient(*serverAddr, *username)

	if err := c.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer c.Disconnect()

	log.Printf("Connected to %s as %s", *serverAddr, *username)

	if err := c.Join(); err != nil {
		log.Fatalf("Failed to join chat: %v", err)
	}

	go func() {
		for msg := range c.Messages() {
			switch msg.Type {
			case 0:
				fmt.Printf("[%s]: %s\n", msg.Sender, msg.Content)
			case 1:
				fmt.Printf("*** %s joined the chat ***\n", msg.Sender)
			case 2:
				fmt.Printf("*** %s left the chat ***\n", msg.Sender)
			}
		}
	}()

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

	if err := c.Leave(); err != nil {
		log.Printf("Failed to send leave message: %v", err)
	}

	log.Println("Disconnected from server")
}

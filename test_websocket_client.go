package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run test_websocket_client.go <filter_key>")
	}

	filterKey := os.Args[1]

	// Connect to WebSocket
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/" + filterKey}
	fmt.Printf("Connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	// Setup interrupt handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Read messages
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			// Parse and log the message with details
			var wsMsg map[string]interface{}
			if err := json.Unmarshal(message, &wsMsg); err == nil {
				msgType, _ := wsMsg["type"].(string)
				timestamp, _ := wsMsg["timestamp"].(string)

				switch msgType {
				case "connected":
					fmt.Printf("✅ Connected to filter subscription at %s\n", timestamp)
				case "event":
					if data, ok := wsMsg["data"].(map[string]interface{}); ok {
						did, _ := data["did"].(string)
						if ops, ok := data["ops"].([]interface{}); ok && len(ops) > 0 {
							if op, ok := ops[0].(map[string]interface{}); ok {
								action, _ := op["action"].(string)
								path, _ := op["path"].(string)
								fmt.Printf("📥 Received event: action=%s, path=%s (repo: %s...) at %s\n",
									action, path, did[8:20], timestamp)
							}
						} else {
							fmt.Printf("📥 Received event (repo: %s...) at %s\n", did[8:20], timestamp)
						}
					} else {
						fmt.Printf("📥 Received event at %s\n", timestamp)
					}
				case "error":
					if data, ok := wsMsg["data"].(map[string]interface{}); ok {
						errorMsg, _ := data["error"].(string)
						fmt.Printf("❌ Error: %s at %s\n", errorMsg, timestamp)
					}
				default:
					fmt.Printf("📨 Received message: %s\n", string(message))
				}
			} else {
				// Fallback to raw message display
				fmt.Printf("📨 Received: %s\n", string(message))
			}
		}
	}()

	fmt.Println("WebSocket client connected! Waiting for filtered events...")
	fmt.Println("Press Ctrl+C to exit")

	// Wait for interrupt
	select {
	case <-done:
		return
	case <-interrupt:
		fmt.Println("\nClosing connection...")

		// Send close message
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("write close:", err)
			return
		}

		// Wait for close or timeout
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		return
	}
}

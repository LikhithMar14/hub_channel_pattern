package main

import (
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
)

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	hub := NewRedisHub(redisURL)
	defer hub.Close()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading to WebSocket: %v", err)
			return
		}

		clientID := uuid.New().String()
		client := &Client{
			id:   clientID,
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}

		if err := hub.RegisterClient(clientID); err != nil {
			log.Printf("Error registering client: %v", err)
			conn.Close()
			return
		}

		hub.SetClientConnection(clientID, map[string]interface{}{
			"connected_at": "now",
			"ip":           r.RemoteAddr,
		})

		go client.readPump()
		go client.writePump()
	})

	log.Println("Server is running on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

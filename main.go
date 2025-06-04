package main

import (
	"log"
	"net/http"

	"github.com/google/uuid"
)

func main() {
	hub := &Hub{
		clients: make(map[*Client]bool),
		broadcast: make(chan *Message),
		register: make(chan *Client),
		unregister: make(chan *Client),
	}
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading to WebSocket: %v", err)
			return
		}
		clientID := uuid.New().String()
		client := &Client{
			id: clientID,
			hub: hub,
			conn: conn,
			send: make(chan []byte, 256),
		}
		hub.register <- client
		go client.readPump()
		go client.writePump()
	})

	log.Println("Server is running on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
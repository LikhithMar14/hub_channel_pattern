package main

import (
	"log"
)

type Message struct {
	ClientID string `json:"client_id"`
	Message  string `json:"message"`
}

type Hub struct {
	clients map[*Client]bool

	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

			log.Printf("Client %s registered", client.id)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- []byte(msg.Message):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

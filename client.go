package main

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	id   string
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 // Made consistent with ReadLimit
	writeWait      = 10 * time.Second
)

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, text, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		log.Printf("Received raw message: %s", text)

		// Check if the message is valid JSON by looking at the first character
		trimmed := bytes.TrimSpace(text)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			// Attempt to decode as JSON
			msg := &Message{}
			reader := bytes.NewReader(text)
			decoder := json.NewDecoder(reader)

			if err := decoder.Decode(msg); err != nil {
				log.Printf("Invalid JSON format: %v", err)
				// Treat as plain text message
				c.hub.broadcast <- &Message{
					ClientID: c.id,
					Message:  string(text),
				}
				continue
			}

			if msg.ClientID == "" {
				msg.ClientID = c.id
			}

			log.Printf("Decoded JSON message: %+v", msg)
			c.hub.broadcast <- msg
		} else {
			// Handle as plain text message
			log.Printf("Received plain text message from client %s", c.id)
			c.hub.broadcast <- &Message{
				ClientID: c.id,
				Message:  string(text),
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}
		}
	}
}

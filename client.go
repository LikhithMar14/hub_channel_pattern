package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	id   string
	hub  *RedisHub
	conn *websocket.Conn
	send chan []byte
}

type MessageBroadcaster struct {
	hub    *RedisHub
	client *Client
	ctx    context.Context
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
	writeWait      = 10 * time.Second
)

func NewMessageBroadcaster(hub *RedisHub, client *Client) *MessageBroadcaster {
	return &MessageBroadcaster{
		hub:    hub,
		client: client,
		ctx:    context.Background(),
	}
}

func (mb *MessageBroadcaster) listenForMessages() {
	pubsub := mb.hub.client.Subscribe(mb.ctx, "broadcast")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var broadcastMsg Message
		if err := json.Unmarshal([]byte(msg.Payload), &broadcastMsg); err != nil {
			log.Printf("Error unmarshaling broadcast message: %v", err)
			continue
		}

		if broadcastMsg.ClientID != mb.client.id {
			select {
			case mb.client.send <- []byte(broadcastMsg.Message):
			default:
				close(mb.client.send)
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.UnregisterClient(c.id)
		c.hub.RemoveClientConnection(c.id)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.hub.SetClientHeartbeat(c.id)
		return nil
	})

	mb := NewMessageBroadcaster(c.hub, c)
	go mb.listenForMessages()

	for {
		messageType, text, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		log.Printf("Received raw message: %d", messageType)
		log.Printf("Received raw message: %s", text)

		trimmed := bytes.TrimSpace(text)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			msg := &Message{}
			reader := bytes.NewReader(text)
			decoder := json.NewDecoder(reader)

			if err := decoder.Decode(msg); err != nil {
				log.Printf("Invalid JSON format: %v", err)
				c.hub.BroadcastMessage(&Message{
					ClientID: c.id,
					Message:  string(text),
				})
				continue
			}

			if msg.ClientID == "" {
				msg.ClientID = c.id
			}

			log.Printf("Decoded JSON message: %+v", msg)
			c.hub.BroadcastMessage(msg)
		} else {
			log.Printf("Received plain text message from client %s", c.id)
			c.hub.BroadcastMessage(&Message{
				ClientID: c.id,
				Message:  string(text),
			})
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
			c.hub.SetClientHeartbeat(c.id)
		}
	}
}

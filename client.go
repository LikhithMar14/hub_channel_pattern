package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	id    string
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	close chan struct{}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		close(c.close)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.hub.ctx.Done():
			return
		default:
			_, messageBytes, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error for client %s: %v", c.id, err)
				}
				return
			}

			var msg Message
			if err := json.Unmarshal(messageBytes, &msg); err != nil {
				log.Printf("Invalid message from client %s: %v", c.id, err)
				continue
			}

			if msg.SenderID == "" {
				msg.SenderID = c.id
			}

			switch msg.Type {
			case TypeBid:
				if msg.Action == ActionPlaceBid && msg.BiddingPrice > 0 {
					c.hub.bid <- &Bid{
						SenderID: msg.SenderID,
						Price:    msg.BiddingPrice,
					}
				} else {
					log.Printf("Invalid bid from client %s: price %f", c.id, msg.BiddingPrice)
				}
			case TypePing:
				pong, _ := json.Marshal(Message{
					Type:      TypePong,
					AuctionID: msg.AuctionID,
					SenderID:  c.id,
					Timestamp: time.Now(),
				})
				select {
				case c.send <- pong:
				default:
					return
				}
			default:
				log.Printf("Unhandled message type %s from client %s", msg.Type, c.id)
			}
		}
	}
}

func (c *Client) writePump() {
	defer close(c.close)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.hub.ctx.Done():
			return

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error for client %s: %v", c.id, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

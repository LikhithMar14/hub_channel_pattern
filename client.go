package main

import (
	"encoding/json"
	"log"
	"github.com/gorilla/websocket"
)



type MessageType string

const (
	TypeAuction MessageType = "auction"
	TypeBid     MessageType = "bid"
)

type AuctionAction string

const (
	ActionJoin  AuctionAction = "join"
	ActionLeave AuctionAction = "leave"
)

const ActionPlaceBid = "place_bid"

type Message struct {
	Type         MessageType `json:"type"`
	Action       string      `json:"action,omitempty"`
	AuctionID    string      `json:"auctionId"`
	SenderID     string      `json:"senderId"`
	BiddingPrice float64     `json:"biddingPrice,omitempty"`
	Content      string      `json:"content,omitempty"`
}



type Client struct {
	id   string
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Println("invalid message:", err)
			continue
		}

		c.hub.broadcast <- messageBytes
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("write error:", err)
			break
		}
	}

}

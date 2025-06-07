package main

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type Bid struct {
	SenderID  string    `json:"sender_id"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	bid        chan *Bid
	unregister chan *Client
	ctx        context.Context
	cancel     context.CancelFunc
	highestBid *Bid
	auctionID  string
	minBidIncrement float64
}

func NewHub(auctionID string) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:         make(map[*Client]bool),
		broadcast:       make(chan []byte, 256), 
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		bid:             make(chan *Bid),
		ctx:             ctx,
		cancel:          cancel,
		auctionID:       auctionID,
		minBidIncrement: 1.0, 
	}
}

func (h *Hub) Run() {
	defer func() {
		for client := range h.clients {
			close(client.send)
		}
	}()

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client %s registered to auction %s", client.id, h.auctionID)
			
			if h.highestBid != nil {
				if response, err := json.Marshal(Message{
					Type:         TypeBid,
					Action:       ActionCurrentBid,
					AuctionID:    h.auctionID,
					SenderID:     h.highestBid.SenderID,
					BiddingPrice: h.highestBid.Price,
					Content:      "Current highest bid",
					Timestamp:    h.highestBid.Timestamp,
				}); err == nil {
					select {
					case client.send <- response:
					case <-time.After(time.Second):
						log.Printf("Failed to send current bid to client %s", client.id)
					}
				}
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client %s unregistered from auction %s", client.id, h.auctionID)
			}

		case message := <-h.broadcast:
			h.broadcastToClients(message)

		case bid := <-h.bid:
			h.processBid(bid)

		case <-h.ctx.Done():
			log.Printf("Hub for auction %s shutting down", h.auctionID)
			return
		}
	}
}

func (h *Hub) processBid(bid *Bid) {
	if bid.Price <= 0 {
		log.Printf("Invalid bid amount: %f from %s", bid.Price, bid.SenderID)
		return
	}

	if h.highestBid != nil {
		if bid.Price <= h.highestBid.Price {
			log.Printf("Rejected bid from %s: %f not higher than current %f", 
				bid.SenderID, bid.Price, h.highestBid.Price)
			h.sendBidRejection(bid.SenderID, "Bid must be higher than current highest bid")
			return
		}
		
		if bid.Price < h.highestBid.Price + h.minBidIncrement {
			log.Printf("Rejected bid from %s: increment too small", bid.SenderID)
			h.sendBidRejection(bid.SenderID, "Bid increment too small")
			return
		}
	}
	
	bid.Timestamp = time.Now()
	h.highestBid = bid
	
	response, err := json.Marshal(Message{
		Type:         TypeBid,
		Action:       ActionPlaceBid,
		AuctionID:    h.auctionID,
		SenderID:     bid.SenderID,
		BiddingPrice: bid.Price,
		Content:      "New highest bid",
		Timestamp:    bid.Timestamp,
	})
	
	if err != nil {
		log.Printf("Error marshaling bid response: %v", err)
		return
	}

	log.Printf("New highest bid in auction %s: %f from %s", h.auctionID, bid.Price, bid.SenderID)
	h.broadcastToClients(response)
}

func (h *Hub) sendBidRejection(senderID, reason string) {
	response, err := json.Marshal(Message{
		Type:      TypeError,
		Action:    ActionBidRejected,
		AuctionID: h.auctionID,
		SenderID:  senderID,
		Content:   reason,
		Timestamp: time.Now(),
	})
	
	if err != nil {
		return
	}

	for client := range h.clients {
		if client.id == senderID {
			select {
			case client.send <- response:
			case <-time.After(time.Second):
				log.Printf("Failed to send rejection to client %s", client.id)
			}
			break
		}
	}
}

func (h *Hub) broadcastToClients(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
				case <-time.After(time.Second):
			log.Printf("Client %s send channel blocked, removing", client.id)
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) GetClientCount() int {
	return len(h.clients)
}
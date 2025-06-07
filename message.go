package main

import "time"

type MessageType string

const (
	TypeAuction MessageType = "auction"
	TypeBid     MessageType = "bid"
	TypeError   MessageType = "error"
	TypePing    MessageType = "ping"
	TypePong    MessageType = "pong"
)

type AuctionAction string

const (
	ActionJoin        AuctionAction = "join"
	ActionLeave       AuctionAction = "leave"
	ActionPlaceBid    string        = "place_bid"
	ActionCurrentBid  string        = "current_bid"
	ActionBidRejected string        = "bid_rejected"
)

type Message struct {
	Type         MessageType `json:"type"`
	Action       string      `json:"action,omitempty"`
	AuctionID    string      `json:"auctionId"`
	SenderID     string      `json:"senderId"`
	BiddingPrice float64     `json:"biddingPrice,omitempty"`
	Content      string      `json:"content,omitempty"`
	Timestamp    time.Time   `json:"timestamp,omitempty"`
}

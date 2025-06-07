package main

import (
	"encoding/json"
	"sync"
	"time"
)

type HubManager struct {
	hubs map[string]*Hub
	mu   sync.RWMutex
}

type HubStats struct {
	AuctionID   string `json:"auction_id"`
	ClientCount int    `json:"client_count"`
	HighestBid  *Bid   `json:"highest_bid,omitempty"`
}

type Stats struct {
	TotalHubs int        `json:"total_hubs"`
	Hubs      []HubStats `json:"hubs"`
	Timestamp time.Time  `json:"timestamp"`
}

func NewHubManager() *HubManager {
	return &HubManager{
		hubs: make(map[string]*Hub),
	}
}

func (m *HubManager) GetHub(auctionId string) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[auctionId]
}

func (m *HubManager) GetOrCreateHub(auctionId string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if hub, exists := m.hubs[auctionId]; exists {
		return hub
	}
	
	hub := NewHub(auctionId)
	m.hubs[auctionId] = hub
	go hub.Run()
	return hub
}

func (m *HubManager) DeleteHub(auctionId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if hub, exists := m.hubs[auctionId]; exists {
		hub.cancel()
		delete(m.hubs, auctionId)
	}
}

func (m *HubManager) GetStats() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var hubStats []HubStats
	for auctionID, hub := range m.hubs {
		stats := HubStats{
			AuctionID:   auctionID,
			ClientCount: hub.GetClientCount(),
			HighestBid:  hub.highestBid,
		}
		hubStats = append(hubStats, stats)
	}
	
	stats := Stats{
		TotalHubs: len(m.hubs),
		Hubs:      hubStats,
		Timestamp: time.Now(),
	}
	
	data, _ := json.Marshal(stats)
	return string(data)
}


func (m *HubManager) CleanupInactiveHubs() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for auctionID, hub := range m.hubs {
		if hub.GetClientCount() == 0 {
			hub.cancel()
			delete(m.hubs, auctionID)
		}
	}
}
package main

import "sync"

type HubManager struct {
	hubs map[string]*Hub
	mu sync.RWMutex
}

func NewHubManager() *HubManager {
	return &HubManager {
		hubs: make(map[string]*Hub),
	}
}

func (m *HubManager)GetHub(auctionId string) *Hub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[auctionId]
}

func (m *HubManager)AddHub(auctionId string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.hubs[auctionId]; exists {
		return nil
	}
	hub := NewHub()
	m.hubs[auctionId] = hub
	go hub.Run()
	return hub
}
func (m *HubManager)DeleteHub(auctionId string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.hubs[auctionId]; !exists {
		return
	}
	hub := m.hubs[auctionId]
	hub.cancel()
	delete(m.hubs, auctionId)
}
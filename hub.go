package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Message struct {
	ClientID string `json:"client_id"`
	Message  string `json:"message"`
}

type RedisHub struct {
	client    *redis.Client
	pubsub    *redis.PubSub
	broadcast chan *Message
	ctx       context.Context
}

func NewRedisHub(redisURL string) *RedisHub {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Failed to parse Redis URL:", err)
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	pubsub := rdb.Subscribe(ctx, "broadcast")

	return &RedisHub{
		client:    rdb,
		pubsub:    pubsub,
		broadcast: make(chan *Message, 100),
		ctx:       ctx,
	}
}

func (h *RedisHub) RegisterClient(clientID string) error {
	return h.client.SAdd(h.ctx, "clients", clientID).Err()
}

func (h *RedisHub) UnregisterClient(clientID string) error {
	return h.client.SRem(h.ctx, "clients", clientID).Err()
}

func (h *RedisHub) BroadcastMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.client.Publish(h.ctx, "broadcast", data).Err()
}

func (h *RedisHub) SetClientConnection(clientID string, data map[string]interface{}) error {
	return h.client.HSet(h.ctx, "client:"+clientID, data).Err()
}

func (h *RedisHub) RemoveClientConnection(clientID string) error {
	return h.client.Del(h.ctx, "client:"+clientID).Err()
}

func (h *RedisHub) SetClientHeartbeat(clientID string) error {
	return h.client.Set(h.ctx, "heartbeat:"+clientID, time.Now().Unix(), 70*time.Second).Err()
}

func (h *RedisHub) Close() error {
	h.pubsub.Close()
	return h.client.Close()
}

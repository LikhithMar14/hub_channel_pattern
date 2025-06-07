package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	hubManager = NewHubManager()
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	auctionId := r.URL.Query().Get("auctionId")
	senderId := r.URL.Query().Get("senderId")

	if auctionId == "" || senderId == "" {
		http.Error(w, "auctionId and senderId are required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	// we repeated this logic twice becuase if there are differnt types of clients let's say premium user and a normal user one can send huge data like pictures and stuff and
	// other can only send messages so we need to handle both cases

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	hub := hubManager.GetOrCreateHub(auctionId)

	client := &Client{
		id:   senderId,
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := hubManager.GetStats()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(stats))
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/stats", statsHandler)

	addr := ":8080"
	log.Printf("Server started at http://localhost%s", addr)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
	
		for range ticker.C {
			hubManager.CleanupInactiveHubs()
		}
	}()
	

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

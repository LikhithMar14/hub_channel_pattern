package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
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

	hub := hubManager.GetHub(auctionId)
	if hub == nil {
		hub = hubManager.AddHub(auctionId)
	}

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

func main() {
	http.HandleFunc("/ws", wsHandler)

	addr := ":8080"
	log.Printf("Server started at http://localhost%s", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

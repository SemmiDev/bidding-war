package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	item    = NewItem("item1", 1, time.Now().UTC(), time.Now().UTC().Add(1*time.Hour))
	auction = NewAuction(item)
)

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server starting at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}

type Type string

const (
	tryBiddingType Type = "Try Bidding"
	getWinnerType  Type = "Get Winner"
)

var (
	upgrader      = websocket.Upgrader{}
	connectionsMu sync.Mutex
	connections   = make(map[string]*WebSocketConnection)
)

type WebSocketConnection struct {
	*websocket.Conn
	Email string
}

type SocketPayload struct {
	ItemID string  `json:"itemID"`
	Amount float64 `json:"amount"`
	Type   Type    `json:"type"`
}

type SocketResponse struct {
	Item    *Item  `json:"item"`
	Winner  *Bid   `json:"winner"`
	BidHist []*Bid `json:"bidHistory"`
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Could not open requested file", http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, "%s", content)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	currentGorillaConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		log.Println("Error: Empty email")
		http.Error(w, "Empty email is not allowed", http.StatusBadRequest)
		return
	}

	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	currentConn := WebSocketConnection{Conn: currentGorillaConn, Email: email}
	connections[email] = &currentConn

	go handleIO(&currentConn)
}

func handleIO(currentConn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				ejectConnection(currentConn)
				return
			}

			log.Println("ERROR: Reading JSON from WebSocket", err.Error())
			continue
		}

		if payload.Type == tryBiddingType {
			bid := NewBid(currentConn.Email, payload.Amount)
			err := auction.TryBid(bid)
			if err != nil {
				// todo: send error message to client
			}
		}

		broadcastMessage(currentConn, getWinnerType, "")
	}
}

func broadcastMessage(currentConn *WebSocketConnection, kind Type, message string) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	if kind == getWinnerType {
		for _, eachConn := range connections {
			_ = eachConn.WriteJSON(SocketResponse{
				Item:    auction.Item,
				Winner:  auction.GetWinner(),
				BidHist: auction.GetBidHistory(),
			})
		}
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	delete(connections, currentConn.Email)
}

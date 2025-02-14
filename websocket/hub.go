package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from the clients
	broadcast chan []byte

	// Subscriptions for specific trips
	tripSubscriptions map[int64]map[*Client]bool
	subMutex         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:         make(chan []byte),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		clients:          make(map[*Client]bool),
		tripSubscriptions: make(map[int64]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.unsubscribeFromAllTrips(client)
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// SubscribeToTrip subscribes a client to updates for a specific trip
func (h *Hub) SubscribeToTrip(client *Client, tripID int64) {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()

	if h.tripSubscriptions[tripID] == nil {
		h.tripSubscriptions[tripID] = make(map[*Client]bool)
	}
	h.tripSubscriptions[tripID][client] = true
}

// UnsubscribeFromTrip unsubscribes a client from a specific trip
func (h *Hub) UnsubscribeFromTrip(client *Client, tripID int64) {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()

	if subs, exists := h.tripSubscriptions[tripID]; exists {
		delete(subs, client)
		if len(subs) == 0 {
			delete(h.tripSubscriptions, tripID)
		}
	}
}

// unsubscribeFromAllTrips removes client from all trip subscriptions
func (h *Hub) unsubscribeFromAllTrips(client *Client) {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()

	for tripID, subs := range h.tripSubscriptions {
		delete(subs, client)
		if len(subs) == 0 {
			delete(h.tripSubscriptions, tripID)
		}
	}
}

// BroadcastToTrip sends a message to all clients subscribed to a specific trip
func (h *Hub) BroadcastToTrip(tripID int64, message interface{}) {
	h.subMutex.RLock()
	subscribers := h.tripSubscriptions[tripID]
	h.subMutex.RUnlock()

	if len(subscribers) == 0 {
		return
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	for client := range subscribers {
		select {
		case client.send <- jsonMessage:
		default:
			h.unregister <- client
		}
	}
}

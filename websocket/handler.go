package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, this should be more restrictive
		return true
	},
	// Disable compression for compatibility with some browsers
	EnableCompression: false,
}

func HandleWebSocket(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log all WebSocket connection attempts with details
		log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)
		log.Printf("Request URL: %s", r.URL.String())
		log.Printf("Request Headers: %v", r.Header)
		
		// Add CORS headers for pre-flight requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// For development purposes, use default values if auth info is not present
		var userID int64 = 1 // Default user ID for testing
		userType := "driver"  // Default user type for testing
		
		// Try to get values from context, but don't fail if they're not there
		if id, ok := r.Context().Value("userID").(int); ok {
			userID = int64(id)
		}
		
		if uType, ok := r.Context().Value("userType").(string); ok {
			userType = uType
		}
		
		log.Printf("Attempting to upgrade connection to WebSocket for user %d (%s)", userID, userType)
		
		// Upgrade the HTTP connection to a WebSocket with explicit headers
		responseHeader := http.Header{}
		responseHeader.Add("Sec-WebSocket-Protocol", "chat")
		
		// Try to upgrade connection
		conn, err := upgrader.Upgrade(w, r, responseHeader)
		if err != nil {
			log.Printf("ERROR: Failed to upgrade connection: %v", err)
			http.Error(w, fmt.Sprintf("Failed to upgrade connection: %v", err), http.StatusInternalServerError)
			return
		}
		
		log.Printf("WebSocket connection successfully established with %s", r.RemoteAddr)
		
		// Create a new client and register it with the hub
		client := &Client{
			hub:      hub,
			conn:     conn,
			send:     make(chan []byte, 256),
			userID:   userID,
			userType: userType,
		}
		
		// Register this client with the hub
		hub.register <- client
		
		log.Printf("Client registered with hub: UserID=%d, UserType=%s", userID, userType)
		
		// Start the read and write pumps in separate goroutines
		go client.writePump()
		go client.readPump()
		
		// Send an initial acknowledgment message
		welcomeMsg := map[string]interface{}{
			"type":    "welcome",
			"message": "Connected to HakikaRide WebSocket Server",
		}
		
		// Convert the welcome message to JSON
		welcomeJSON, err := json.Marshal(welcomeMsg)
		if err == nil {
			client.send <- welcomeJSON
			log.Printf("Sent welcome message to client")
		}
	}
}

// NotifyTripUpdate broadcasts a trip update to all subscribed clients
func (h *Hub) NotifyTripUpdate(tripID int64, updateType string, data interface{}) {
	message := Message{
		Type: updateType,
		Payload: map[string]interface{}{
			"tripId": tripID,
			"data":   data,
		},
	}
	h.BroadcastToTrip(tripID, message)
}

// NotifyLocationUpdate broadcasts a location update to all subscribed clients
func (h *Hub) NotifyLocationUpdate(tripID int64, location interface{}) {
	h.NotifyTripUpdate(tripID, "location_update", location)
}

// NotifyAttendanceUpdate broadcasts an attendance update to all subscribed clients
func (h *Hub) NotifyAttendanceUpdate(tripID int64, attendance interface{}) {
	h.NotifyTripUpdate(tripID, "attendance_update", attendance)
}

// NotifyTripStatusChange broadcasts a trip status change to all subscribed clients
func (h *Hub) NotifyTripStatusChange(tripID int64, status string) {
	h.NotifyTripUpdate(tripID, "status_change", map[string]string{
		"status": status,
	})
}

// NotifyEmergency broadcasts an emergency alert to all subscribed clients
func (h *Hub) NotifyEmergency(tripID int64, message string, location interface{}) {
	h.NotifyTripUpdate(tripID, "emergency", map[string]interface{}{
		"message":  message,
		"location": location,
	})
}

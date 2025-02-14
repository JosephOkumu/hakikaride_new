package websocket

import (
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
}

func HandleWebSocket(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user info from context (set by auth middleware)
		userID := r.Context().Value("userID").(int)
		userType := r.Context().Value("userType").(string)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
			return
		}

		ServeWs(hub, conn, int64(userID), userType)
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

package api

import (
	"database/sql"
	"encoding/json"
	"hakikaride/websocket"
	"net/http"
	"time"
)

type EmergencyReport struct {
	TripID    int64      `json:"tripId"`
	Timestamp time.Time  `json:"timestamp"`
	Location  *Location  `json:"location"`
}

// HandleEmergencyReport handles emergency reports from drivers
func HandleEmergencyReport(db *sql.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var report EmergencyReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Get user info from context (set by auth middleware)
		userID := r.Context().Value("userID").(int)
		
		// Save emergency report to database
		_, err := db.Exec(`
			INSERT INTO EmergencyReports (UserID, TripID, Latitude, Longitude, Timestamp)
			VALUES (?, ?, ?, ?, ?)`,
			userID, report.TripID, 
			report.Location.Latitude, report.Location.Longitude, 
			time.Now())
		if err != nil {
			http.Error(w, "Error saving emergency report", http.StatusInternalServerError)
			return
		}

		// Notify all users connected to this trip via WebSocket
		if hub != nil {
			message := "Emergency reported by driver. Support team notified."
			hub.NotifyEmergency(report.TripID, message, report.Location)
		}

		// Return success response
		response := map[string]interface{}{
			"success": true,
			"message": "Emergency reported successfully",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

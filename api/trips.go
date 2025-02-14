package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Trip struct {
	TripID    int64     `json:"tripId"`
	DriverID  int64     `json:"driverId"`
	BusID     int64     `json:"busId"`
	RouteID   int64     `json:"routeId"`
	StartTime time.Time `json:"startTime"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Status    string    `json:"status"`
}

type TripRequest struct {
	DriverID int64 `json:"driverId"`
	BusID    int64 `json:"busId"`
	RouteID  int64 `json:"routeId"`
}

func HandleStartTrip(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TripRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Check if driver has any active trips
		var activeTrips int
		err = tx.QueryRow(`
			SELECT COUNT(*) FROM Trips 
			WHERE DriverID = ? AND Status IN ('scheduled', 'in_progress')`,
			req.DriverID).Scan(&activeTrips)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if activeTrips > 0 {
			http.Error(w, "Driver already has an active trip", http.StatusConflict)
			return
		}

		// Check if bus is available
		var busInUse int
		err = tx.QueryRow(`
			SELECT COUNT(*) FROM Trips 
			WHERE BusID = ? AND Status IN ('scheduled', 'in_progress')`,
			req.BusID).Scan(&busInUse)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if busInUse > 0 {
			http.Error(w, "Bus is already in use", http.StatusConflict)
			return
		}

		// Create new trip
		result, err := tx.Exec(`
			INSERT INTO Trips (DriverID, BusID, RouteID, StartTime, Status)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP, 'in_progress')`,
			req.DriverID, req.BusID, req.RouteID)
		if err != nil {
			http.Error(w, "Error creating trip", http.StatusInternalServerError)
			return
		}

		tripID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting trip ID", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			http.Error(w, "Error completing trip creation", http.StatusInternalServerError)
			return
		}

		// Return trip details
		var trip Trip
		err = db.QueryRow(`
			SELECT TripID, DriverID, BusID, RouteID, StartTime, EndTime, Status
			FROM Trips WHERE TripID = ?`, tripID).Scan(
			&trip.TripID, &trip.DriverID, &trip.BusID, &trip.RouteID,
			&trip.StartTime, &trip.EndTime, &trip.Status)
		if err != nil {
			http.Error(w, "Error retrieving trip details", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(trip)
	}
}

func HandleEndTrip(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := r.URL.Query().Get("tripId")
		if tripID == "" {
			http.Error(w, "Trip ID is required", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// Update trip status
		result, err := tx.Exec(`
			UPDATE Trips 
			SET Status = 'completed', EndTime = CURRENT_TIMESTAMP
			WHERE TripID = ? AND Status = 'in_progress'`,
			tripID)
		if err != nil {
			http.Error(w, "Error updating trip", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Error checking update result", http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, "Trip not found or already completed", http.StatusNotFound)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			http.Error(w, "Error completing trip update", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "Trip completed successfully",
		})
	}
}

func HandleGetTrips(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		driverID := r.URL.Query().Get("driverId")

		query := `
			SELECT TripID, DriverID, BusID, RouteID, StartTime, EndTime, Status
			FROM Trips
			WHERE 1=1`
		args := []interface{}{}

		if status != "" {
			query += " AND Status = ?"
			args = append(args, status)
		}
		if driverID != "" {
			query += " AND DriverID = ?"
			args = append(args, driverID)
		}

		query += " ORDER BY StartTime DESC LIMIT 100"

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		trips := []Trip{}
		for rows.Next() {
			var trip Trip
			err := rows.Scan(
				&trip.TripID, &trip.DriverID, &trip.BusID, &trip.RouteID,
				&trip.StartTime, &trip.EndTime, &trip.Status)
			if err != nil {
				http.Error(w, "Error scanning trip data", http.StatusInternalServerError)
				return
			}
			trips = append(trips, trip)
		}

		json.NewEncoder(w).Encode(trips)
	}
}

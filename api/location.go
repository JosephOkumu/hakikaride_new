package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Location struct {
	LocationID int64     `json:"locationId"`
	TripID     int64     `json:"tripId"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Speed      float64   `json:"speed"`
	Timestamp  time.Time `json:"timestamp"`
}

type LocationUpdate struct {
	TripID    int64   `json:"tripId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
}

func HandleLocationUpdate(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var update LocationUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate trip exists and is in progress
		var tripStatus string
		err := db.QueryRow("SELECT Status FROM Trips WHERE TripID = ?", update.TripID).Scan(&tripStatus)
		if err == sql.ErrNoRows {
			http.Error(w, "Trip not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if tripStatus != "in_progress" {
			http.Error(w, "Trip is not in progress", http.StatusBadRequest)
			return
		}

		// Insert location update
		result, err := db.Exec(`
			INSERT INTO LocationUpdates (TripID, Latitude, Longitude, Speed, Timestamp)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			update.TripID, update.Latitude, update.Longitude, update.Speed)
		if err != nil {
			http.Error(w, "Error saving location update", http.StatusInternalServerError)
			return
		}

		locationID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting location ID", http.StatusInternalServerError)
			return
		}

		// Return the created location update
		var location Location
		err = db.QueryRow(`
			SELECT LocationID, TripID, Latitude, Longitude, Speed, Timestamp
			FROM LocationUpdates WHERE LocationID = ?`,
			locationID).Scan(
			&location.LocationID, &location.TripID, &location.Latitude,
			&location.Longitude, &location.Speed, &location.Timestamp)
		if err != nil {
			http.Error(w, "Error retrieving location update", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(location)
	}
}

func HandleGetTripLocations(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := r.URL.Query().Get("tripId")
		if tripID == "" {
			http.Error(w, "Trip ID is required", http.StatusBadRequest)
			return
		}

		// Get last 100 location updates for the trip
		rows, err := db.Query(`
			SELECT LocationID, TripID, Latitude, Longitude, Speed, Timestamp
			FROM LocationUpdates
			WHERE TripID = ?
			ORDER BY Timestamp DESC
			LIMIT 100`,
			tripID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		locations := []Location{}
		for rows.Next() {
			var location Location
			err := rows.Scan(
				&location.LocationID, &location.TripID, &location.Latitude,
				&location.Longitude, &location.Speed, &location.Timestamp)
			if err != nil {
				http.Error(w, "Error scanning location data", http.StatusInternalServerError)
				return
			}
			locations = append(locations, location)
		}

		json.NewEncoder(w).Encode(locations)
	}
}

func HandleGetLastLocation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := r.URL.Query().Get("tripId")
		if tripID == "" {
			http.Error(w, "Trip ID is required", http.StatusBadRequest)
			return
		}

		var location Location
		err := db.QueryRow(`
			SELECT LocationID, TripID, Latitude, Longitude, Speed, Timestamp
			FROM LocationUpdates
			WHERE TripID = ?
			ORDER BY Timestamp DESC
			LIMIT 1`,
			tripID).Scan(
			&location.LocationID, &location.TripID, &location.Latitude,
			&location.Longitude, &location.Speed, &location.Timestamp)

		if err == sql.ErrNoRows {
			http.Error(w, "No location updates found for this trip", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(location)
	}
}

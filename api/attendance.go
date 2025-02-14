package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Attendance struct {
	AttendanceID int64     `json:"attendanceId"`
	TripID       int64     `json:"tripId"`
	StudentID    int64     `json:"studentId"`
	Status       string    `json:"status"` // picked_up, dropped_off
	Timestamp    time.Time `json:"timestamp"`
	Location     *Location `json:"location,omitempty"`
}

func HandleStudentPickup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TripID    int64   `json:"tripId"`
			StudentID int64   `json:"studentId"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}

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

		// Verify trip is in progress
		var tripStatus string
		err = tx.QueryRow("SELECT Status FROM Trips WHERE TripID = ?", req.TripID).Scan(&tripStatus)
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

		// Record location
		locationResult, err := tx.Exec(`
			INSERT INTO LocationUpdates (TripID, Latitude, Longitude, Timestamp)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			req.TripID, req.Latitude, req.Longitude)
		if err != nil {
			http.Error(w, "Error recording location", http.StatusInternalServerError)
			return
		}

		locationID, err := locationResult.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting location ID", http.StatusInternalServerError)
			return
		}

		// Record attendance
		result, err := tx.Exec(`
			INSERT INTO Attendance (TripID, StudentID, Status, LocationID, Timestamp)
			VALUES (?, ?, 'picked_up', ?, CURRENT_TIMESTAMP)`,
			req.TripID, req.StudentID, locationID)
		if err != nil {
			http.Error(w, "Error recording attendance", http.StatusInternalServerError)
			return
		}

		attendanceID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting attendance ID", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			http.Error(w, "Error completing attendance record", http.StatusInternalServerError)
			return
		}

		// Return attendance record
		var attendance Attendance
		err = db.QueryRow(`
			SELECT a.AttendanceID, a.TripID, a.StudentID, a.Status, a.Timestamp,
				   l.LocationID, l.Latitude, l.Longitude, l.Speed, l.Timestamp
			FROM Attendance a
			JOIN LocationUpdates l ON a.LocationID = l.LocationID
			WHERE a.AttendanceID = ?`,
			attendanceID).Scan(
			&attendance.AttendanceID, &attendance.TripID, &attendance.StudentID,
			&attendance.Status, &attendance.Timestamp,
			&attendance.Location.LocationID, &attendance.Location.Latitude,
			&attendance.Location.Longitude, &attendance.Location.Speed,
			&attendance.Location.Timestamp)
		if err != nil {
			http.Error(w, "Error retrieving attendance record", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(attendance)
	}
}

func HandleStudentDropoff(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TripID    int64   `json:"tripId"`
			StudentID int64   `json:"studentId"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}

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

		// Verify trip is in progress
		var tripStatus string
		err = tx.QueryRow("SELECT Status FROM Trips WHERE TripID = ?", req.TripID).Scan(&tripStatus)
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

		// Record location
		locationResult, err := tx.Exec(`
			INSERT INTO LocationUpdates (TripID, Latitude, Longitude, Timestamp)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
			req.TripID, req.Latitude, req.Longitude)
		if err != nil {
			http.Error(w, "Error recording location", http.StatusInternalServerError)
			return
		}

		locationID, err := locationResult.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting location ID", http.StatusInternalServerError)
			return
		}

		// Record attendance
		result, err := tx.Exec(`
			INSERT INTO Attendance (TripID, StudentID, Status, LocationID, Timestamp)
			VALUES (?, ?, 'dropped_off', ?, CURRENT_TIMESTAMP)`,
			req.TripID, req.StudentID, locationID)
		if err != nil {
			http.Error(w, "Error recording attendance", http.StatusInternalServerError)
			return
		}

		attendanceID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Error getting attendance ID", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			http.Error(w, "Error completing attendance record", http.StatusInternalServerError)
			return
		}

		// Return attendance record
		var attendance Attendance
		err = db.QueryRow(`
			SELECT a.AttendanceID, a.TripID, a.StudentID, a.Status, a.Timestamp,
				   l.LocationID, l.Latitude, l.Longitude, l.Speed, l.Timestamp
			FROM Attendance a
			JOIN LocationUpdates l ON a.LocationID = l.LocationID
			WHERE a.AttendanceID = ?`,
			attendanceID).Scan(
			&attendance.AttendanceID, &attendance.TripID, &attendance.StudentID,
			&attendance.Status, &attendance.Timestamp,
			&attendance.Location.LocationID, &attendance.Location.Latitude,
			&attendance.Location.Longitude, &attendance.Location.Speed,
			&attendance.Location.Timestamp)
		if err != nil {
			http.Error(w, "Error retrieving attendance record", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(attendance)
	}
}

func HandleGetTripAttendance(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := r.URL.Query().Get("tripId")
		if tripID == "" {
			http.Error(w, "Trip ID is required", http.StatusBadRequest)
			return
		}

		rows, err := db.Query(`
			SELECT a.AttendanceID, a.TripID, a.StudentID, a.Status, a.Timestamp,
				   l.LocationID, l.Latitude, l.Longitude, l.Speed, l.Timestamp
			FROM Attendance a
			JOIN LocationUpdates l ON a.LocationID = l.LocationID
			WHERE a.TripID = ?
			ORDER BY a.Timestamp DESC`,
			tripID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		attendance := []Attendance{}
		for rows.Next() {
			var record Attendance
			record.Location = &Location{}
			err := rows.Scan(
				&record.AttendanceID, &record.TripID, &record.StudentID,
				&record.Status, &record.Timestamp,
				&record.Location.LocationID, &record.Location.Latitude,
				&record.Location.Longitude, &record.Location.Speed,
				&record.Location.Timestamp)
			if err != nil {
				http.Error(w, "Error scanning attendance record", http.StatusInternalServerError)
				return
			}
			attendance = append(attendance, record)
		}

		json.NewEncoder(w).Encode(attendance)
	}
}

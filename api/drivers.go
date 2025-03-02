package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type Driver struct {
	DriverID       int    `json:"driverId"`
	UserID         int    `json:"userId"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	PhoneNumber    string `json:"phoneNumber"`
	BusID          *int   `json:"busId,omitempty"`
	BusNumberPlate string `json:"busNumberPlate,omitempty"`
	IsActive       bool   `json:"isActive"`
}

// HandleAddDriver handles adding a new driver
func HandleAddDriver(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var driver Driver
		if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Insert into Users table first
		var userID int64
		err = tx.QueryRow(`
			INSERT INTO Users (Username, Email, PasswordHash, UserType)
			VALUES (?, ?, ?, 'driver')
			RETURNING UserID`,
			driver.PhoneNumber, // Using phone number as username
			driver.PhoneNumber+"@hakikaride.com", // Temporary email
			"defaultpass", // This should be changed on first login
		).Scan(&userID)

		if err != nil {
			tx.Rollback()
			log.Printf("Error creating user: %v", err)
			http.Error(w, "Error creating driver", http.StatusInternalServerError)
			return
		}

		// Insert into Drivers table
		_, err = tx.Exec(`
			INSERT INTO Drivers (UserID, FirstName, LastName, PhoneNumber, IsActive)
			VALUES (?, ?, ?, ?, true)`,
			userID, driver.FirstName, driver.LastName, driver.PhoneNumber)

		if err != nil {
			tx.Rollback()
			log.Printf("Error creating driver: %v", err)
			http.Error(w, "Error creating driver", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Error creating driver", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Driver added successfully",
		})
	}
}

// HandleUpdateDriver handles updating driver information
func HandleUpdateDriver(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var driver Driver
		if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Update driver information
		_, err = tx.Exec(`
			UPDATE Drivers 
			SET FirstName = ?, LastName = ?, PhoneNumber = ?
			WHERE DriverID = ?`,
			driver.FirstName, driver.LastName, driver.PhoneNumber, driver.DriverID)

		if err != nil {
			tx.Rollback()
			log.Printf("Error updating driver: %v", err)
			http.Error(w, "Error updating driver", http.StatusInternalServerError)
			return
		}

		// Update user information
		_, err = tx.Exec(`
			UPDATE Users 
			SET Username = ?, Email = ?
			WHERE UserID = ?`,
			driver.PhoneNumber, driver.PhoneNumber+"@hakikaride.com", driver.UserID)

		if err != nil {
			tx.Rollback()
			log.Printf("Error updating user: %v", err)
			http.Error(w, "Error updating driver", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Error updating driver", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Driver updated successfully",
		})
	}
}

// HandleListDriversDetailed handles retrieving all active drivers with detailed information
func HandleListDriversDetailed(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT d.DriverID, d.UserID, d.FirstName, d.LastName, d.PhoneNumber, d.IsActive, d.BusID, b.NumberPlate
			FROM Drivers d
			LEFT JOIN Buses b ON d.BusID = b.BusID
			WHERE d.IsActive = true
			ORDER BY d.FirstName, d.LastName`)
		if err != nil {
			log.Printf("Error querying drivers: %v", err)
			http.Error(w, "Error retrieving drivers", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var drivers []Driver
		for rows.Next() {
			var d Driver
			var busID sql.NullInt64
			var numberPlate sql.NullString
			if err := rows.Scan(&d.DriverID, &d.UserID, &d.FirstName, &d.LastName, &d.PhoneNumber, &d.IsActive, &busID, &numberPlate); err != nil {
				log.Printf("Error scanning driver row: %v", err)
				continue
			}
			drivers = append(drivers, d)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"drivers": drivers,
		})
	}
}

// HandleDeleteDriver handles deleting a driver
func HandleDeleteDriver(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		driverID := r.URL.Query().Get("id")
		if driverID == "" {
			http.Error(w, "Driver ID is required", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Get UserID for the driver
		var userID int
		err = tx.QueryRow("SELECT UserID FROM Drivers WHERE DriverID = ?", driverID).Scan(&userID)
		if err != nil {
			tx.Rollback()
			log.Printf("Error getting UserID: %v", err)
			http.Error(w, "Error deleting driver", http.StatusInternalServerError)
			return
		}

		// Delete from Drivers table
		_, err = tx.Exec("DELETE FROM Drivers WHERE DriverID = ?", driverID)
		if err != nil {
			tx.Rollback()
			log.Printf("Error deleting driver: %v", err)
			http.Error(w, "Error deleting driver", http.StatusInternalServerError)
			return
		}

		// Delete from Users table
		_, err = tx.Exec("DELETE FROM Users WHERE UserID = ?", userID)
		if err != nil {
			tx.Rollback()
			log.Printf("Error deleting user: %v", err)
			http.Error(w, "Error deleting driver", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Error deleting driver", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Driver deleted successfully",
		})
	}
}

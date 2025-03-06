package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	
	"hakikaride/auth"
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
		w.Header().Set("Content-Type", "application/json")
		var driver Driver
		if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid request body",
			})
			return
		}

		// Generate a secure random password for the new driver
		password, err := auth.GenerateSecurePassword(10) // 10-character password
		if err != nil {
			log.Printf("Error generating password: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error creating driver",
			})
			return
		}

		// Hash the password for storage
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error creating driver",
			})
			return
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Database error",
			})
			return
		}

		// If busNumberPlate is provided, look up the BusID
		if driver.BusNumberPlate != "" {
			var busID int
			err := tx.QueryRow("SELECT BusID FROM Buses WHERE NumberPlate = ?", driver.BusNumberPlate).Scan(&busID)
			if err != nil {
				if err == sql.ErrNoRows {
					// Bus not found
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": "Bus with number plate " + driver.BusNumberPlate + " not found",
					})
				} else {
					// Database error
					log.Printf("Error looking up bus: %v", err)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": "Database error when looking up bus",
					})
				}
				tx.Rollback()
				return
			}
			id := busID
			driver.BusID = &id
		}

		// Insert into Users table first
		var userID int64
		err = tx.QueryRow(`
			INSERT INTO Users (Username, Email, PasswordHash, UserType, PasswordResetRequired)
			VALUES (?, ?, ?, 'driver', true)
			RETURNING UserID`,
			driver.PhoneNumber, // Using phone number as username
			driver.PhoneNumber+"@hakikaride.com", // Temporary email
			hashedPassword, // Securely generated and hashed password
		).Scan(&userID)

		if err != nil {
			tx.Rollback()
			log.Printf("Error creating user: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error creating driver",
			})
			return
		}

		// Insert into Drivers table
		_, err = tx.Exec(`
			INSERT INTO Drivers (UserID, FirstName, LastName, PhoneNumber, BusID, IsActive)
			VALUES (?, ?, ?, ?, ?, true)`,
			userID, driver.FirstName, driver.LastName, driver.PhoneNumber, driver.BusID)

		if err != nil {
			tx.Rollback()
			log.Printf("Error creating driver: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error creating driver",
			})
			return
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			log.Printf("Error committing transaction: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error creating driver",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Driver added successfully",
			"driverDetails": map[string]interface{}{
				"phoneNumber": driver.PhoneNumber,
				"initialPassword": password,
			},
		})
	}
}

// HandleUpdateDriver handles updating driver information
func HandleUpdateDriver(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Read the body for logging
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error reading request body",
			})
			return
		}
		
		// Log the received body
		log.Printf("Received update driver request body: %s", string(bodyBytes))
		
		// Recreate the body for further processing
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		
		var driver Driver
		if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
			log.Printf("Error decoding driver JSON: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid request body",
			})
			return
		}
		
		// Log the decoded driver
		log.Printf("Decoded driver: %+v", driver)

		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Database error",
			})
			return
		}

		// If busNumberPlate is provided, look up the BusID
		if driver.BusNumberPlate != "" {
			var busID int
			err := tx.QueryRow("SELECT BusID FROM Buses WHERE NumberPlate = ?", driver.BusNumberPlate).Scan(&busID)
			if err != nil {
				if err == sql.ErrNoRows {
					// Bus not found
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": "Bus with number plate " + driver.BusNumberPlate + " not found",
					})
				} else {
					// Database error
					log.Printf("Error looking up bus: %v", err)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": "Database error when looking up bus",
					})
				}
				tx.Rollback()
				return
			}
			id := busID
			driver.BusID = &id
		}

		// Update driver information
		_, err = tx.Exec(`
			UPDATE Drivers 
			SET FirstName = ?, LastName = ?, PhoneNumber = ?, BusID = ?
			WHERE DriverID = ?`,
			driver.FirstName, driver.LastName, driver.PhoneNumber, driver.BusID, driver.DriverID)

		if err != nil {
			tx.Rollback()
			log.Printf("Error updating driver: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error updating driver",
			})
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
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error updating driver",
			})
			return
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			log.Printf("Error committing transaction: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error updating driver",
			})
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
		w.Header().Set("Content-Type", "application/json")
		rows, err := db.Query(`
			SELECT d.DriverID, d.UserID, d.FirstName, d.LastName, d.PhoneNumber, d.IsActive, d.BusID, b.NumberPlate
			FROM Drivers d
			LEFT JOIN Buses b ON d.BusID = b.BusID
			WHERE d.IsActive = true
			ORDER BY d.FirstName, d.LastName`)
		if err != nil {
			log.Printf("Error querying drivers: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error retrieving drivers",
			})
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
			
			if busID.Valid {
				id := int(busID.Int64)
				d.BusID = &id
			}
			
			if numberPlate.Valid {
				d.BusNumberPlate = numberPlate.String
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
		w.Header().Set("Content-Type", "application/json")
		// Extract driver ID from URL path
		parts := strings.Split(r.URL.Path, "/")
		driverID := parts[len(parts)-1]
		
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

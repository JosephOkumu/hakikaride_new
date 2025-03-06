package api

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

// HandleAddBus adds a new bus to the system
func HandleAddBus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			NumberPlate string `json:"numberPlate"`
			RouteId     string `json:"routeId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Printf("Error decoding request body: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid request data",
			})
			return
		}

		// Validate inputs
		if request.NumberPlate == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Number plate is required",
			})
			return
		}

		// Now we're using routeId as a string directly, no need to convert to int
		routeName := request.RouteId
		if routeName == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Route is required",
			})
			return
		}

		// Check if number plate already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM Buses WHERE NumberPlate = ?", request.NumberPlate).Scan(&count)
		if err != nil {
			log.Printf("Error checking for existing number plate: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error checking for existing number plate",
			})
			return
		}

		if count > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "A bus with this number plate already exists",
			})
			return
		}

		// Insert new bus with route name instead of ID
		result, err := db.Exec(
			"INSERT INTO Buses (NumberPlate, Route, IsActive) VALUES (?, ?, true)",
			request.NumberPlate, routeName,
		)
		if err != nil {
			log.Printf("Error inserting bus: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error creating bus",
			})
			return
		}

		busID, _ := result.LastInsertId()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Bus added successfully",
			"busId":   busID,
		})
	}
}

// HandleUpdateBus updates an existing bus
func HandleUpdateBus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			BusId       string `json:"busId"`
			NumberPlate string `json:"numberPlate"`
			RouteId     string `json:"routeId"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Printf("Error decoding request body: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid request data",
			})
			return
		}

		// Validate inputs
		if request.BusId == "" || request.NumberPlate == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Bus ID and number plate are required",
			})
			return
		}

		busID, err := strconv.Atoi(request.BusId)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid bus ID",
			})
			return
		}

		// Now we're using routeId as a string directly, no need to convert to int
		routeName := request.RouteId
		if routeName == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Route is required",
			})
			return
		}

		// Check if number plate already exists for a different bus
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM Buses WHERE NumberPlate = ? AND BusID != ?", 
			request.NumberPlate, busID).Scan(&count)
		if err != nil {
			log.Printf("Error checking for existing number plate: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error checking for existing number plate",
			})
			return
		}

		if count > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Another bus with this number plate already exists",
			})
			return
		}

		// Update bus with route name instead of ID
		_, err = db.Exec(
			"UPDATE Buses SET NumberPlate = ?, Route = ? WHERE BusID = ?",
			request.NumberPlate, routeName, busID,
		)
		if err != nil {
			log.Printf("Error updating bus: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error updating bus",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Bus updated successfully",
		})
	}
}

// HandleDeleteBus deletes a bus
func HandleDeleteBus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["id"]
		if !ok {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "No bus ID provided",
			})
			return
		}

		busID, err := strconv.Atoi(id)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid bus ID",
			})
			return
		}

		// Check if bus is assigned to any active drivers
		var driversCount int
		err = db.QueryRow("SELECT COUNT(*) FROM Drivers WHERE BusID = ? AND IsActive = true", busID).Scan(&driversCount)
		if err != nil {
			log.Printf("Error checking for assigned drivers: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error checking for assigned drivers",
			})
			return
		}

		if driversCount > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Cannot delete bus: It is assigned to one or more drivers",
			})
			return
		}

		// Check if bus is used in any active trips
		var tripsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM Trips WHERE BusID = ? AND Status IN ('scheduled', 'in_progress')", busID).Scan(&tripsCount)
		if err != nil {
			log.Printf("Error checking for active trips: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error checking for active trips",
			})
			return
		}

		if tripsCount > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Cannot delete bus: It is assigned to one or more active trips",
			})
			return
		}

		// Soft delete the bus
		_, err = db.Exec("UPDATE Buses SET IsActive = false WHERE BusID = ?", busID)
		if err != nil {
			log.Printf("Error deleting bus: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error deleting bus",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Bus deleted successfully",
		})
	}
}

// HandleListBusesDetailed returns a detailed list of all buses
func HandleListBusesDetailed(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT b.BusID, b.NumberPlate, b.Route,
			       CASE WHEN t.TripID IS NOT NULL THEN 'In Trip' ELSE 'Available' END as Status
			FROM Buses b
			LEFT JOIN Trips t ON b.BusID = t.BusID AND t.Status = 'in_progress'
			WHERE b.IsActive = true
			ORDER BY b.NumberPlate`)
		if err != nil {
			log.Printf("Error getting buses: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting buses list",
			})
			return
		}
		defer rows.Close()

		var buses []map[string]interface{}
		for rows.Next() {
			var bus struct {
				BusID       int
				NumberPlate string
				Route       string
				Status      string
			}
			if err := rows.Scan(&bus.BusID, &bus.NumberPlate, &bus.Route, &bus.Status); err != nil {
				continue
			}
			buses = append(buses, map[string]interface{}{
				"id":     bus.BusID,
				"plate":  bus.NumberPlate,
				"routeId": bus.Route,
				"route":  bus.Route,
				"status": bus.Status,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"buses":   buses,
		})
	}
}

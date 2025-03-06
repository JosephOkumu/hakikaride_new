package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// AdminDashboardStats represents the statistics shown on the admin dashboard
type AdminDashboardStats struct {
	Success      bool `json:"success"`
	ActiveBuses  int  `json:"activeBuses"`
	TotalStudents int `json:"totalStudents"`
	ActiveRoutes  int `json:"activeRoutes"`
	IssuesCount  int  `json:"issuesCount"`
}

// HandleAdminDashboardStats returns statistics for the admin dashboard
func HandleAdminDashboardStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var stats AdminDashboardStats

		// Get active buses count
		err := db.QueryRow("SELECT COUNT(*) FROM Buses WHERE IsActive = true").Scan(&stats.ActiveBuses)
		if err != nil {
			log.Printf("Error getting active buses count: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting dashboard stats",
			})
			return
		}

		// Get total students
		err = db.QueryRow("SELECT COUNT(*) FROM Students WHERE IsActive = true").Scan(&stats.TotalStudents)
		if err != nil {
			log.Printf("Error getting students count: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting dashboard stats",
			})
			return
		}

		// Get active routes
		err = db.QueryRow("SELECT COUNT(*) FROM Routes WHERE IsActive = true").Scan(&stats.ActiveRoutes)
		if err != nil {
			log.Printf("Error getting active routes count: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting dashboard stats",
			})
			return
		}

		// Get issues count (trips with issues in the last 24 hours)
		err = db.QueryRow(`
			SELECT COUNT(*) FROM Trips 
			WHERE Status = 'cancelled' 
			AND StartTime >= datetime('now', '-24 hours')`).Scan(&stats.IssuesCount)
		if err != nil {
			log.Printf("Error getting issues count: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting dashboard stats",
			})
			return
		}

		stats.Success = true
		json.NewEncoder(w).Encode(stats)
	}
}

// HandleListDrivers returns a list of all drivers
func HandleListDrivers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT d.DriverID, d.FirstName, d.LastName, d.PhoneNumber, 
			       COALESCE(t.ActiveTrips, 0) as ActiveTrips
			FROM Drivers d
			LEFT JOIN (
				SELECT DriverID, COUNT(*) as ActiveTrips 
				FROM Trips 
				WHERE Status = 'in_progress'
				GROUP BY DriverID
			) t ON d.DriverID = t.DriverID
			WHERE d.IsActive = true`)
		if err != nil {
			log.Printf("Error getting drivers: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting drivers list",
			})
			return
		}
		defer rows.Close()

		var drivers []map[string]interface{}
		for rows.Next() {
			var driver struct {
				DriverID    int
				FirstName   string
				LastName    string
				PhoneNumber string
				ActiveTrips int
			}
			if err := rows.Scan(&driver.DriverID, &driver.FirstName, &driver.LastName, 
				&driver.PhoneNumber, &driver.ActiveTrips); err != nil {
				continue
			}
			drivers = append(drivers, map[string]interface{}{
				"id": driver.DriverID,
				"name": driver.FirstName + " " + driver.LastName,
				"phone": driver.PhoneNumber,
				"activeTrips": driver.ActiveTrips,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"drivers": drivers,
		})
	}
}

// HandleListRoutes returns a list of all routes
func HandleListRoutes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT r.RouteID, r.RouteName, r.Description,
			       COUNT(DISTINCT b.BusID) as AssignedBuses,
			       COUNT(DISTINCT t.TripID) as ActiveTrips
			FROM Routes r
			LEFT JOIN Buses b ON b.Route = r.RouteName AND b.IsActive = true
			LEFT JOIN Trips t ON t.Route = r.RouteName AND t.Status = 'in_progress'
			WHERE r.IsActive = true
			GROUP BY r.RouteID`)
		if err != nil {
			log.Printf("Error getting routes: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error getting routes list",
			})
			return
		}
		defer rows.Close()

		var routes []map[string]interface{}
		for rows.Next() {
			var route struct {
				RouteID       int
				RouteName     string
				Description   string
				AssignedBuses int
				ActiveTrips   int
			}
			if err := rows.Scan(&route.RouteID, &route.RouteName, &route.Description,
				&route.AssignedBuses, &route.ActiveTrips); err != nil {
				continue
			}
			routes = append(routes, map[string]interface{}{
				"id": route.RouteID,
				"name": route.RouteName,
				"description": route.Description,
				"buses": route.AssignedBuses,
				"activeTrips": route.ActiveTrips,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"routes": routes,
		})
	}
}

// HandleListBuses returns a list of all buses
func HandleListBuses(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT b.BusID, b.NumberPlate, b.Route,
			       CASE WHEN t.TripID IS NOT NULL THEN 'In Trip' ELSE 'Available' END as Status
			FROM Buses b
			LEFT JOIN Trips t ON b.BusID = t.BusID AND t.Status = 'in_progress'
			WHERE b.IsActive = true`)
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
				"id": bus.BusID,
				"plate": bus.NumberPlate,
				"route": bus.Route,
				"status": bus.Status,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"buses": buses,
		})
	}
}

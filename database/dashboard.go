package database

import (
	"database/sql"
)

type DashboardStats struct {
	ActiveBuses    int `json:"activeBuses"`
	ActiveRoutes   int `json:"activeRoutes"`
	TotalStudents  int `json:"totalStudents"`
}

func GetDashboardStats(db *sql.DB) (DashboardStats, error) {
	stats := DashboardStats{}

	// Get active buses count
	err := db.QueryRow("SELECT COUNT(*) FROM Buses WHERE IsActive = true").Scan(&stats.ActiveBuses)
	if err != nil {
		return stats, err
	}

	// Get active routes count
	err = db.QueryRow("SELECT COUNT(*) FROM Routes WHERE IsActive = true").Scan(&stats.ActiveRoutes)
	if err != nil {
		return stats, err
	}

	// Get total students count
	err = db.QueryRow("SELECT COUNT(*) FROM Students WHERE IsActive = true").Scan(&stats.TotalStudents)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

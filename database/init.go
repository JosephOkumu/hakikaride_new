package database

import (
	"database/sql"
	"log"
	"os"
	"hakikaride/auth"
)

func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", os.Getenv("DATABASE_PATH"))
	if err != nil {
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

func recreateStudentsTable(db *sql.DB) error {
	// Drop the Students table if it exists
	_, err := db.Exec(`DROP TABLE IF EXISTS Students`)
	if err != nil {
		return err
	}

	// Create the Students table with new schema
	_, err = db.Exec(`
		CREATE TABLE Students (
			StudentID INTEGER PRIMARY KEY AUTOINCREMENT,
			ParentID INTEGER NOT NULL,
			FirstName VARCHAR(100) NOT NULL,
			LastName VARCHAR(100) NOT NULL,
			Grade VARCHAR(50) NOT NULL,
			AdmNumber VARCHAR(50) NOT NULL UNIQUE,
			Address TEXT NOT NULL,
			EmergencyContact VARCHAR(15),
			IsActive BOOLEAN DEFAULT TRUE,
			FOREIGN KEY (ParentID) REFERENCES Parents(ParentID)
		)
	`)
	return err
}

func createTables(db *sql.DB) error {
	// Create tables using the schema
	schema := `
	CREATE TABLE IF NOT EXISTS Users (
		UserID INTEGER PRIMARY KEY AUTOINCREMENT,
		Username VARCHAR(255) NOT NULL UNIQUE,
		Email VARCHAR(255) NOT NULL UNIQUE,
		PasswordHash VARCHAR(255) NOT NULL,
		UserType VARCHAR(10) NOT NULL CHECK (UserType IN ('admin', 'parent', 'driver')),
		CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
		LastLogin DATETIME,
		IsActive BOOLEAN DEFAULT 1,
		PasswordResetRequired BOOLEAN DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS Parents (
		ParentID INTEGER PRIMARY KEY AUTOINCREMENT,
		UserID INTEGER NOT NULL,
		FirstName VARCHAR(100) NOT NULL,
		LastName VARCHAR(100) NOT NULL,
		PhoneNumber VARCHAR(15) NOT NULL,
		Address TEXT,
		IsActive BOOLEAN DEFAULT TRUE,
		FOREIGN KEY (UserID) REFERENCES Users(UserID)
	);

	CREATE TABLE IF NOT EXISTS Students (
		StudentID INTEGER PRIMARY KEY AUTOINCREMENT,
		ParentID INTEGER NOT NULL,
		FirstName VARCHAR(100) NOT NULL,
		LastName VARCHAR(100) NOT NULL,
		Grade VARCHAR(50) NOT NULL,
		AdmNumber VARCHAR(50) NOT NULL UNIQUE,
		Address TEXT NOT NULL,
		EmergencyContact VARCHAR(15),
		IsActive BOOLEAN DEFAULT TRUE,
		FOREIGN KEY (ParentID) REFERENCES Parents(ParentID)
	);

	CREATE TABLE IF NOT EXISTS Routes (
		RouteID INTEGER PRIMARY KEY AUTOINCREMENT,
		RouteName VARCHAR(255) NOT NULL,
		Description TEXT,
		IsActive BOOLEAN DEFAULT TRUE
	);

	CREATE TABLE IF NOT EXISTS Buses (
		BusID INTEGER PRIMARY KEY AUTOINCREMENT,
		NumberPlate VARCHAR(50) NOT NULL UNIQUE,
		RouteID INTEGER NOT NULL,
		IsActive BOOLEAN DEFAULT TRUE,
		FOREIGN KEY (RouteID) REFERENCES Routes(RouteID)
	);

	CREATE TABLE IF NOT EXISTS Drivers (
		DriverID INTEGER PRIMARY KEY AUTOINCREMENT,
		UserID INTEGER NOT NULL,
		FirstName VARCHAR(100) NOT NULL,
		LastName VARCHAR(100) NOT NULL,
		PhoneNumber VARCHAR(15) NOT NULL,
		BusID INTEGER,
		IsActive BOOLEAN DEFAULT TRUE,
		FOREIGN KEY (UserID) REFERENCES Users(UserID),
		FOREIGN KEY (BusID) REFERENCES Buses(BusID)
	);

	CREATE TABLE IF NOT EXISTS Trips (
		TripID INTEGER PRIMARY KEY AUTOINCREMENT,
		DriverID INTEGER NOT NULL,
		BusID INTEGER NOT NULL,
		RouteID INTEGER NOT NULL,
		StartTime DATETIME DEFAULT CURRENT_TIMESTAMP,
		EndTime DATETIME,
		Status VARCHAR(20) CHECK (Status IN ('scheduled', 'in_progress', 'completed', 'cancelled')) DEFAULT 'scheduled',
		FOREIGN KEY (DriverID) REFERENCES Drivers(DriverID),
		FOREIGN KEY (BusID) REFERENCES Buses(BusID),
		FOREIGN KEY (RouteID) REFERENCES Routes(RouteID)
	);

	CREATE TABLE IF NOT EXISTS LocationUpdates (
		LocationID INTEGER PRIMARY KEY AUTOINCREMENT,
		TripID INTEGER NOT NULL,
		Latitude DECIMAL(10, 8) NOT NULL,
		Longitude DECIMAL(11, 8) NOT NULL,
		Speed DECIMAL(5, 2),
		Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (TripID) REFERENCES Trips(TripID)
	);

	CREATE TABLE IF NOT EXISTS Attendance (
		AttendanceID INTEGER PRIMARY KEY AUTOINCREMENT,
		TripID INTEGER NOT NULL,
		StudentID INTEGER NOT NULL,
		Status VARCHAR(20) CHECK (Status IN ('picked_up', 'dropped_off')) NOT NULL,
		LocationID INTEGER NOT NULL,
		Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (TripID) REFERENCES Trips(TripID),
		FOREIGN KEY (StudentID) REFERENCES Students(StudentID),
		FOREIGN KEY (LocationID) REFERENCES LocationUpdates(LocationID)
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON Users(Email);
	CREATE INDEX IF NOT EXISTS idx_users_username ON Users(Username);
	CREATE INDEX IF NOT EXISTS idx_students_admission ON Students(AdmNumber);
	CREATE INDEX IF NOT EXISTS idx_location_updates_trip ON LocationUpdates(TripID);
	CREATE INDEX IF NOT EXISTS idx_trips_status ON Trips(Status);
	CREATE INDEX IF NOT EXISTS idx_attendance_trip ON Attendance(TripID);
	CREATE INDEX IF NOT EXISTS idx_attendance_student ON Attendance(StudentID);
	CREATE INDEX IF NOT EXISTS idx_attendance_timestamp ON Attendance(Timestamp);
	`

	_, err := db.Exec(schema)
	if err != nil {
		log.Printf("Error creating tables: %v", err)
		return err
	}

	return nil
}

// CreateDefaultAdmin creates default admin users if no admin exists
func CreateDefaultAdmin(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM Users WHERE UserType = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Admins already exist
	}

	// Default admin credentials
	defaultAdmins := []struct {
		username string
		email    string
		password string
	}{
		{
			username: "admin",
			email:    "admin@hakikaride.com",
			password: "admin123",
		},
		{
			username: "supervisor",
			email:    "supervisor@hakikaride.com",
			password: "supervisor456",
		},
		{
			username: "manager",
			email:    "manager@hakikaride.com",
			password: "manager789",
		},
	}

	// Create each admin account
	for _, admin := range defaultAdmins {
		hashedPassword, err := auth.HashPassword(admin.password)
		if err != nil {
			return err
		}

		_, err = db.Exec(`
			INSERT INTO Users (Username, Email, PasswordHash, UserType, CreatedAt, IsActive, PasswordResetRequired)
			VALUES (?, ?, ?, 'admin', CURRENT_TIMESTAMP, true, true)`,
			admin.username, admin.email, hashedPassword)

		if err != nil {
			return err
		}
	}

	return nil
}

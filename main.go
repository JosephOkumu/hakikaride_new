package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"

	"hakikaride/api"
	"hakikaride/auth"
	"hakikaride/database"
	"hakikaride/websocket"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"encoding/json"
)

var (
	db  *sql.DB
	hub *websocket.Hub
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}


	// Initialize database
	var err error
	db, err = database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create default admin if none exists
	if err := database.CreateDefaultAdmin(db); err != nil {
		log.Printf("Warning: Failed to create default admin: %v", err)
	}

	// Initialize WebSocket hub
	hub = websocket.NewHub()
	go hub.Run()

	// Create router
	r := mux.NewRouter()

	// Static file server
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Serve favicon.ico
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})

	// Auth routes
	r.HandleFunc("/api/auth/login", auth.HandleLogin(db)).Methods("POST")
	r.HandleFunc("/api/auth/register", auth.HandleRegister(db)).Methods("POST")

	// Config routes - accessible without authentication
	r.HandleFunc("/api/config/here-api-key", api.HandleHereApiKey()).Methods("GET")

	// Protected API routes
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(auth.AuthMiddleware)
	
	// Password change endpoint (requires authentication)
	apiRouter.HandleFunc("/auth/change-password", auth.HandlePasswordChange(db)).Methods("POST")

	// Mock driver info API for development
	apiRouter.HandleFunc("/driver/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"success": true,
			"firstName": "John",
			"lastName": "Driver",
			"driverId": 1,
		}
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")
	
	// Mock student list API for development
	apiRouter.HandleFunc("/driver/students", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		students := []map[string]interface{}{
			{"id": 1, "name": "Alice Smith", "isPickedUp": false},
			{"id": 2, "name": "Bob Johnson", "isPickedUp": false},
			{"id": 3, "name": "Carol Williams", "isPickedUp": false},
			{"id": 4, "name": "David Brown", "isPickedUp": false},
		}
		response := map[string]interface{}{
			"success": true,
			"students": students,
		}
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// Admin routes
	adminRouter := apiRouter.PathPrefix("/admin").Subrouter()
	adminRouter.Use(adminMiddleware)
	adminRouter.HandleFunc("/dashboard-stats", api.HandleAdminDashboardStats(db)).Methods("GET")

	// Driver management routes
	adminRouter.HandleFunc("/drivers/add", api.HandleAddDriver(db)).Methods("POST")
	adminRouter.HandleFunc("/drivers/update", api.HandleUpdateDriver(db)).Methods("PUT")
	adminRouter.HandleFunc("/drivers/delete/{id:[0-9]+}", api.HandleDeleteDriver(db)).Methods("DELETE")
	adminRouter.HandleFunc("/drivers/list", api.HandleListDriversDetailed(db)).Methods("GET")

	// Other admin routes
	adminRouter.HandleFunc("/routes", api.HandleListRoutes(db)).Methods("GET")
	adminRouter.HandleFunc("/buses", api.HandleListBuses(db)).Methods("GET")

	// Bus management routes
	adminRouter.HandleFunc("/buses/add", api.HandleAddBus(db)).Methods("POST")
	adminRouter.HandleFunc("/buses/update", api.HandleUpdateBus(db)).Methods("PUT")
	adminRouter.HandleFunc("/buses/delete/{id:[0-9]+}", api.HandleDeleteBus(db)).Methods("DELETE")
	adminRouter.HandleFunc("/buses/list", api.HandleListBusesDetailed(db)).Methods("GET")

	// Student management routes
	adminRouter.HandleFunc("/students/add", api.HandleAddStudent(db)).Methods("POST")
	adminRouter.HandleFunc("/students/bulk-upload", api.HandleBulkUploadStudents(db)).Methods("POST")
	adminRouter.HandleFunc("/students/update", api.HandleUpdateStudent(db)).Methods("PUT")
	adminRouter.HandleFunc("/students/delete", api.HandleDeleteStudent(db)).Methods("DELETE")
	adminRouter.HandleFunc("/students/list", api.HandleListStudents(db)).Methods("GET")

	// Trip routes
	apiRouter.HandleFunc("/trips/start", api.HandleStartTrip(db)).Methods("POST")
	apiRouter.HandleFunc("/trips/end", api.HandleEndTrip(db)).Methods("GET")
	apiRouter.HandleFunc("/trips", api.HandleGetTrips(db)).Methods("GET")

	// Location routes
	apiRouter.HandleFunc("/location/update", api.HandleLocationUpdate(db)).Methods("POST")
	apiRouter.HandleFunc("/location/trip", api.HandleGetTripLocations(db)).Methods("GET")
	apiRouter.HandleFunc("/location/last", api.HandleGetLastLocation(db)).Methods("GET")

	// Attendance routes
	apiRouter.HandleFunc("/attendance/pickup", api.HandleStudentPickup(db)).Methods("POST")
	apiRouter.HandleFunc("/attendance/dropoff", api.HandleStudentDropoff(db)).Methods("POST")
	apiRouter.HandleFunc("/attendance/trip", api.HandleGetTripAttendance(db)).Methods("GET")

	// WebSocket endpoint for real-time communication
	r.HandleFunc("/ws", websocket.HandleWebSocket(hub))
	
	// For Firefox compatibility - add endpoint at old location
	r.HandleFunc("/ws/driver", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Redirecting old WebSocket endpoint from /ws/driver to /ws")
		http.Redirect(w, r, "/ws", http.StatusTemporaryRedirect)
	})

	// Template routes
	r.HandleFunc("/", serveLoginPage)
	r.HandleFunc("/register", serveRegisterPage)
	r.Handle("/change-password", auth.AuthMiddleware(http.HandlerFunc(serveChangePasswordPage)))
	r.HandleFunc("/driver/dashboard", serveDriverDashboard)
	r.HandleFunc("/parent/dashboard", serveParentDashboard)
	r.HandleFunc("/admin/dashboard", serveAdminDashboard)
	r.HandleFunc("/admin/students", serveStudentManagement)
	r.HandleFunc("/admin/drivers", serveDriverManagement)
	r.HandleFunc("/admin/fleet-management", serveFleetManagement)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

// Template functions
func serveLoginPage(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func serveRegisterPage(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func serveDriverDashboard(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/driver-dashboard.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func serveParentDashboard(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/parent-dashboard.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user type from context (set by auth middleware)
		userType, ok := r.Context().Value("userType").(string)
		if !ok || userType != "admin" {
			http.Error(w, "Unauthorized: Admin access required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func serveAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// Get dashboard stats
	stats, err := database.GetDashboardStats(db)
	if err != nil {
		log.Printf("Error getting dashboard stats: %v", err)
		stats = database.DashboardStats{} // Use empty stats in case of error
	}

	// Prepare template data
	data := struct {
		Stats database.DashboardStats
	}{
		Stats: stats,
	}

	templ, err := template.ParseFiles("templates/admin-dashboard.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	templ.Execute(w, data)
}

func serveStudentManagement(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/student-management.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func serveDriverManagement(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/driver-management.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func serveFleetManagement(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/fleet-management.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

func serveChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	templ, err := template.ParseFiles("templates/change-password.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

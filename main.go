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

	// Auth routes
	r.HandleFunc("/api/auth/login", auth.HandleLogin(db)).Methods("POST")
	r.HandleFunc("/api/auth/register", auth.HandleRegister(db)).Methods("POST")

	// Protected API routes
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(auth.AuthMiddleware)

	// Admin routes
	adminRouter := apiRouter.PathPrefix("/admin").Subrouter()
	adminRouter.Use(adminMiddleware)
	adminRouter.HandleFunc("/dashboard-stats", api.HandleAdminDashboardStats(db)).Methods("GET")
	adminRouter.HandleFunc("/drivers", api.HandleListDrivers(db)).Methods("GET")
	adminRouter.HandleFunc("/routes", api.HandleListRoutes(db)).Methods("GET")
	adminRouter.HandleFunc("/buses", api.HandleListBuses(db)).Methods("GET")

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

	// WebSocket route
	apiRouter.HandleFunc("/ws", websocket.HandleWebSocket(hub))

	// Template routes
	r.HandleFunc("/", serveLoginPage)
	r.HandleFunc("/register", serveRegisterPage)
	r.HandleFunc("/driver/dashboard", serveDriverDashboard)
	r.HandleFunc("/parent/dashboard", serveParentDashboard)
	r.HandleFunc("/admin/dashboard", serveAdminDashboard)

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
	templ, err := template.ParseFiles("templates/admin-dashboard.html")
	if err != nil {
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}
	templ.Execute(w, nil)
}

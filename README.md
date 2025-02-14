# HakikaRide - School Bus Tracking System

A real-time school bus tracking system that helps parents monitor their children's school bus location and ensures safe transportation.

## Features

- Real-time bus tracking with GPS integration
- Separate dashboards for parents, drivers, and school administrators
- Student pickup and drop-off management
- Route visualization and monitoring
- Secure authentication system
- Responsive design for all devices

## Tech Stack

- Frontend: HTML5, CSS3, JavaScript
- Backend: Go (Golang)
- Database: SQLite3
- Maps: Leaflet.js
- Real-time updates: WebSocket

## Setup

1. Install Go 1.21 or later
2. Install SQLite3
3. Clone this repository
4. Run `go mod tidy` to install dependencies
5. Run `go run main.go` to start the server
6. Access the application at `http://localhost:8080`

## Project Structure

```
hakikaride/
├── static/          # Static files (CSS, JS, images)
├── templates/       # HTML templates
├── models/          # Database models
├── handlers/        # HTTP handlers
├── database/        # Database initialization and queries
└── main.go         # Entry point
```

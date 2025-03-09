package api

import (
	"encoding/json"
	"net/http"
	"os"
)

// HandleHereApiKey returns the HERE Maps API key from environment variables
func HandleHereApiKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("HERE_API_KEY")
		
		if apiKey == "" {
			http.Error(w, "HERE Maps API key not configured", http.StatusInternalServerError)
			return
		}
		
		response := map[string]interface{}{
			"success": true,
			"apiKey":  apiKey,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

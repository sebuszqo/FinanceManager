package interfaces

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func respondError(w http.ResponseWriter, status int, message string, errors ...[]string) {
	payload := map[string]interface{}{
		"status":  "error",
		"message": message,
		"code":    status,
	}

	if len(errors) > 0 && len(errors[0]) > 0 {
		payload["errors"] = errors[0]
	}

	respondJSON(w, status, payload)
}

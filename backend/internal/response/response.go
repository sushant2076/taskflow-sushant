package response

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

func ValidationError(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusBadRequest, map[string]any{
		"error":  "validation failed",
		"fields": fields,
	})
}

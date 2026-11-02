package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// RespondJSON writes a JSON response with status code
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// RespondError writes a standard JSON error response
func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"error": message})
}

// ParseLimit parses limit query parameter with fallback
func ParseLimit(limitStr string, defaultLimit int) int {
	if strings.EqualFold(limitStr, "all") || limitStr == "-1" {
		return 0 // 0 means return all
	}
	if limitStr == "" {
		return defaultLimit
	}
	val, err := strconv.Atoi(limitStr)
	if err != nil || val <= 0 {
		return defaultLimit
	}
	return val
}

// GetQuerySlice extracts comma-separated or multi-value query parameter slices
func GetQuerySlice(r *http.Request, key string) []string {
	values := r.URL.Query()[key]
	var result []string
	for _, v := range values {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" && !strings.EqualFold(trimmed, "all") {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

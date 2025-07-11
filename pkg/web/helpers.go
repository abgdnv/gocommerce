package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

func RespondJSON(w http.ResponseWriter, logger *slog.Logger, status int, payload any) {
	// Handle nil payload
	if payload == nil {
		w.WriteHeader(status)
		return
	}

	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Error encoding response to JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(response)
}

func RespondError(w http.ResponseWriter, logger *slog.Logger, status int, message string) {
	RespondJSON(w, logger, status, map[string]string{"error": message})
}

// ParseID extracts and validates the ID from the request path. Returns the ID and a boolean indicating success.
func ParseID(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (uuid.UUID, bool) {
	pathValueID := r.PathValue("id")
	id, err := uuid.Parse(pathValueID)
	if err != nil {
		RespondError(w, logger, http.StatusBadRequest, fmt.Sprintf("Invalid ID: %s", pathValueID))
		return uuid.UUID{}, false
	}
	return id, true
}

// GetUserID retrieves the user ID from the request context. Returns the user ID and a boolean indicating success.
func GetUserID(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		RespondError(w, logger, http.StatusUnauthorized, "Unauthorized: Missing or invalid user ID")
		return uuid.Nil, false
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		RespondError(w, logger, http.StatusBadRequest, fmt.Sprintf("Invalid user ID: %s", userID))
		return uuid.Nil, false
	}
	return parsedUserID, true
}

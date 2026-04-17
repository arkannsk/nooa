package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/arkannsk/nooa/examples/models"
)

// CreateUser creates a new user account.
// This godoc comment will be extracted by the AST generator
// as summary/description for OpenAPI spec.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	// Простая валидация (в продакшене используйте github.com/go-playground/validator)
	if req.Name == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FIELDS", "name and email are required")
		return
	}

	// Mock: создаём пользователя
	user := models.User{
		ID:        "usr_" + time.Now().Format("20060102150405"),
		Name:      req.Name,
		Email:     req.Email,
		Age:       req.Age,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(user)
}

// GetUser returns a user by ID.
func GetUser(w http.ResponseWriter, r *http.Request) {
	// В реальном приложении: id := chi.URLParam(r, "id")
	user := models.User{
		ID:        "usr_123",
		Name:      "Alice",
		Email:     "alice@example.com",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}

// ExportUsers exports users as CSV.
func ExportUsers(w http.ResponseWriter, r *http.Request) {
	// Mock CSV output
	csv := "id,name,email\nusr_123,Alice,alice@example.com\n"

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=users.csv")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(csv))
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.Error{
		Code:    code,
		Message: msg,
	})
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/arkannsk/nooa"
	ginAdapter "github.com/arkannsk/nooa/adapter/gin"
	"github.com/gin-gonic/gin"
)

// --- Модели ---

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// --- Middleware ---

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("[LOG] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// --- Хендлеры ---

func listUsers(w http.ResponseWriter, r *http.Request) {
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com"},
		{ID: "2", Name: "Bob", Email: "bob@example.com"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	user := User{ID: "new-1", Name: req.Name, Email: req.Email}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Spec с глобальным middleware (логирование)
	spec := nooa.NewSpec(nooa.Info{
		Title:   "Gin Example API",
		Version: "1.0.0",
	})
	spec.Use(loggingMiddleware)

	// Регистрируем роуты
	listRoute := nooa.NewRoute[struct{}, []User]("GET", "/users", listUsers).
		Summary("List all users").
		Tags("Users").
		Spec()

	createRoute := nooa.NewRoute[CreateUserRequest, User]("POST", "/users", createUser).
		Summary("Create a new user").
		Tags("Users").
		Use(authMiddleware). // route-level middleware
		PossibleErr(http.StatusBadRequest).
		Spec()

	deleteRoute := nooa.NewRoute[struct{}, struct{}]("DELETE", "/users/:id", deleteUser).
		Summary("Delete a user").
		Tags("Users").
		Use(authMiddleware).
		OnNoContent(http.StatusNoContent, "User deleted").
		Spec()

	// API роуты в Gin
	ginAdapter.RegisterSpecAndMux(spec, r, listRoute, createRoute, deleteRoute)

	// OpenAPI JSON (Spec реализует http.Handler)
	r.GET("/openapi.json", ginAdapter.WrapHandler(spec.ServeHTTP))

	// Redoc UI
	r.Any("/redoc/*filepath", ginAdapter.WrapHandler(nooa.RedocUIHandler("/redoc", "/openapi.json").ServeHTTP))

	log.Println("Server starting on http://localhost:8080")
	log.Println("OpenAPI JSON: http://localhost:8080/openapi.json")
	log.Println("Redoc UI:     http://localhost:8080/redoc/")
	log.Fatal(r.Run(":8080"))
}

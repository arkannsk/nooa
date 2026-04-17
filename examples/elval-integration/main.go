package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
)

func createUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user := models.User{
		ID:    "usr_123",
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
		Role:  req.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func init() {
	// Регистрация моделей для интеграции схем
	nooa.RegisterModel[models.User]("User")
	nooa.RegisterModel[models.CreateUserRequest]("CreateUserRequest")
}

func main() {
	mux := http.NewServeMux()

	nooa.NewRoute[models.CreateUserRequest, models.User]("POST", "/users", createUser).
		Summary("Register new user").
		Description("Creates a new user account with validation based on elval annotations.").
		Tags("Users").
		OnSuccess(201, "User created successfully").
		OnClientErr(400, "Validation failed").
		Register(mux).
		RegisterGlobal()

	// Используем стандартный middleware
	handler := nooa.SpecMiddleware(
		mux,
		nooa.Info{
			Title:       "Elval Integration Demo",
			Version:     "1.0.0",
			Description: "Demonstrates nooa + elval integration with rich schemas.",
		},
	)

	log.Println(" Server starting on http://localhost:8080")
	log.Println("📄 OpenAPI Spec: http://localhost:8080/openapi.json")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

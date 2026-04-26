package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
)

// --- Хендлеры для v1 ---
func createUserV1(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	user := models.User{
		ID: "usr_v1_123", Name: req.Name, Email: req.Email, Age: req.Age, Role: req.Role,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// --- Хендлеры для v2 ---
func createUserV2(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	user := models.User{
		ID: "usr_v2_456", Name: req.Name, Email: req.Email, Age: req.Age, Role: req.Role,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func main() {
	mux := http.NewServeMux()

	// 1. Создаем спецификации
	v1Spec := nooa.NewSpec(nooa.Info{
		Title:       "Elval Integration Demo - v1",
		Version:     "1.0.0",
		Description: "API Version 1 Documentation",
	})

	v2Spec := nooa.NewSpec(nooa.Info{
		Title:       "Elval Integration Demo - v2",
		Version:     "2.0.0",
		Description: "API Version 2 Documentation",
	})

	// 2. Регистрируем роуты для v1
	nooa.NewRoute[models.CreateUserRequest, models.User]("POST", "/v1/users", createUserV1).
		Summary("Register new user (v1)").
		Tags("Users").
		OnSuccess(201, "User created successfully").
		OnClientErr(400, "Validation failed").
		Register(mux).       // HTTP Handler
		RegisterSpec(v1Spec) // Привязка к Spec

	// 3. Регистрируем роуты для v2
	nooa.NewRoute[models.CreateUserRequest, models.User]("POST", "/v2/users", createUserV2).
		Summary("Register new user (v2)").
		Tags("Users").
		OnSuccess(200, "User updated/created").
		OnClientErr(400, "Validation failed").
		Register(mux).
		RegisterSpec(v2Spec)

	// 4. Автоматическая регистрация документации
	// Теперь пути будут:
	// v1: /v1/openapi.json и /docs/v1/
	// v2: /v2/openapi.json и /docs/v2/
	nooa.RegisterVersionedAPI("v1", v1Spec, mux)
	nooa.RegisterVersionedAPI("v2", v2Spec, mux)

	log.Println("Server starting on http://localhost:8080")
	log.Println("Swagger UI v1: http://localhost:8080/docs/v1/")
	log.Println("Raw JSON v1:   http://localhost:8080/v1/openapi.json")
	log.Println("Swagger UI v2: http://localhost:8080/docs/v2/")
	log.Println("Raw JSON v2:   http://localhost:8080/v2/openapi.json")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

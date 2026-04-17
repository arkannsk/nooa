package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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
		ID: "usr_123", Name: req.Name, Email: req.Email, Age: req.Age, Role: req.Role,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func init() {
	fmt.Println("DEBUG: Running init()...")
	nooa.RegisterModel[models.User]("User")
	nooa.RegisterModel[models.CreateUserRequest]("CreateUserRequest")
}

func main() {
	mux := http.NewServeMux()

	// 1. Регистрируем роуты
	nooa.NewRoute[models.CreateUserRequest, models.User]("POST", "/users", createUser).
		Summary("Register new user").
		Tags("Users").
		RequestBodySchema("CreateUserRequest").
		ResponseSchema(201, "User").
		OnSuccess(201, "User created successfully").
		OnClientErr(400, "Validation failed").
		Register(mux)

	swaggerHandler := nooa.SwaggerUIHandler("/openapi.json")
	apiHandler := nooa.SpecMiddleware(
		mux,
		nooa.Info{
			Title:       "Elval Integration Demo",
			Version:     "1.0.0",
			Description: "Interactive docs with embedded Swagger UI.",
		},
	)

	// Объединяем всё в один главный хендлер
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если путь начинается с /docs, отдаем Swagger UI
		if r.URL.Path == "/docs" || strings.HasPrefix(r.URL.Path, "/docs/") {
			swaggerHandler.ServeHTTP(w, r)
			return
		}
		// Иначе передаем в API хендлер (который отдаст openapi.json или проксирует в mux)
		apiHandler.ServeHTTP(w, r)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("Swagger UI: http://localhost:8080/docs")
	log.Println("Raw JSON:   http://localhost:8080/openapi.json")

	log.Fatal(http.ListenAndServe(":8080", finalHandler))
}

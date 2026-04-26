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
		ID: "usr_123", Name: req.Name, Email: req.Email, Age: req.Age, Role: req.Role,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "ElVal Integration Demo",
		Description: "Interactive docs with embedded Swagger UI.",
		Version:     "1.0.0",
	})

	nooa.NewRoute[models.CreateUserRequest, models.User]("POST", "/users", createUser).
		Summary("Register new user").
		Tags("Users").
		OnSuccess(200, "User created").
		OnSuccess(201, "User created successfully").
		OnClientErr(400, "Validation failed").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)

	log.Println("Server starting on http://localhost:8080")
	log.Println("Swagger UI: http://localhost:8080/docs/")
	log.Println("Raw JSON:   http://localhost:8080/openapi.json")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

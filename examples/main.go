package main

import (
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/handlers"
	"github.com/arkannsk/nooa/examples/models"
)

func main() {
	mux := http.NewServeMux()

	nooa.NewRoute[models.CreateUserRequest, models.User]("POST", "/users", handlers.CreateUser).
		Summary("Create new user").
		Description("Creates a user account with name, email and optional age").
		Tags("users", "accounts").
		OnSuccess(201, "User created successfully").
		OnClientErr(400, "Invalid request body", nooa.CTJSON, nooa.CTProblemJSON).
		OnClientErr(409, "Email already exists").
		OnServerErr(500, "Internal server error").
		Register(mux).
		RegisterGlobal()

	nooa.NewRoute[struct{}, models.User]("GET", "/users/{id}", handlers.GetUser).
		Summary("Get user by ID").
		Description("Returns a single user by their unique identifier").
		Tags("users").
		OnSuccess(200, "User found").
		OnClientErr(404, "User not found").
		OnServerErr(500, "Internal server error").
		Register(mux).
		RegisterGlobal()

	nooa.NewRoute[struct{}, []byte]("GET", "/users/export", handlers.ExportUsers).
		Summary("Export users as CSV").
		Description("Downloads a CSV file with all users").
		Tags("users", "export").
		RequestContentType(nooa.CTPlainText).
		OnSuccess(200, "CSV file", nooa.CTCSV).
		OnServerErr(500, "Export failed").
		Deprecated().
		Register(mux).
		RegisterGlobal()

	// Middleware для отдачи OpenAPI spec
	handler := nooa.SpecMiddleware(mux, nooa.Info{
		Title:   "User Service API",
		Version: "1.0.0",
		Description: "Example service demonstrating nooa router.\n\n" +
			"## Authentication\nThis API uses OAuth2. Use `/oauth/token` to obtain access.",
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("OpenAPI spec: http://localhost:8080/openapi.json")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	nested_structs "github.com/arkannsk/nooa/examples/models/03_nested"
)

func handleNested(w http.ResponseWriter, r *http.Request) {
	resp := nested_structs.UserWithAddress{
		ID:   "user_123",
		Name: "John Doe",
		Billing: nested_structs.Address{
			Street:  "123 Main St",
			City:    "New York",
			ZipCode: "12345",
			Country: "US",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "03 Nested Demo",
		Version:     "1.0.0",
		Description: "Integration example: nested structures",
	})

	nooa.NewRoute[nested_structs.UserWithAddress, nested_structs.UserWithAddress](
		"GET", "/user", handleNested).
		Summary("Get nested user").
		Description("Returns a user with a nested address").
		Tags("Nested").
		OnSuccess(200, "User retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)

	log.Println("Server starting on http://localhost:9090")
	log.Fatal(http.ListenAndServe(":9090", mux))
}

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	gen "github.com/arkannsk/nooa/examples/models/05_generics"
)

func handleGenericStruct(w http.ResponseWriter, r *http.Request) {
	resp := gen.GenericStruct{
		Name: gen.Option[string]{},
		Age:  gen.Option[int]{},
		Data: gen.Result[string, string]{},
		Items: []gen.Option[gen.Item]{
			gen.Option[gen.Item]{},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCustomGeneric(w http.ResponseWriter, r *http.Request) {
	resp := gen.WithCustomGeneric{
		Value: gen.CustomGeneric[string]{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "05 Generics Demo",
		Version:     "1.0.0",
		Description: "Integration example: generic types (Option, Result, custom generics)",
	})

	nooa.NewRoute[gen.GenericStruct, gen.GenericStruct](
		"GET", "/generic", handleGenericStruct).
		Summary("Get generic struct").
		Description("Returns a struct with generic Option and Result fields").
		Tags("Generics").
		OnSuccess(200, "Generic struct retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[gen.WithCustomGeneric, gen.WithCustomGeneric](
		"GET", "/custom-generic", handleCustomGeneric).
		Summary("Get custom generic struct").
		Description("Returns a struct with a custom generic type rewritten as string").
		Tags("Generics").
		OnSuccess(200, "Custom generic struct retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)

	log.Println("Server starting on http://localhost:9090")
	log.Println("Swagger UI: http://localhost:9090/docs/")
	log.Println("Raw JSON:   http://localhost:9090/openapi.json")
	log.Fatal(http.ListenAndServe(":9090", mux))
}

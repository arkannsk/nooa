package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	custom "github.com/arkannsk/nooa/examples/models/12_custom_types"
)

func handleWithAliases(w http.ResponseWriter, r *http.Request) {
	var req custom.WithAliases
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(req)
}

func handleWithCustomReader(w http.ResponseWriter, r *http.Request) {
	var req custom.WithCustomReader
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(req)
}

func handleWithEmbed(w http.ResponseWriter, r *http.Request) {
	resp := custom.WithEmbed{
		EmbedStruct: custom.EmbedStruct{
			ID:   "embed-1",
			Name: "Embedded",
		},
		Extra: "additional",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "12 Custom Types Demo",
		Version:     "1.0.0",
		Description: "Integration example: type aliases with @oa:rewrite.type, custom reader with @oa:file, and embedded structs",
	})

	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	nooa.NewRoute[custom.WithAliases, custom.WithAliases](
		"POST", "/aliases", handleWithAliases).
		Summary("Type aliases with validation").
		Description("Demonstrates type aliases rewritten as primitive types via @oa:rewrite.type. The 'Email' field uses a CustomString alias with min length and email pattern validation.").
		Tags("Custom Types").
		OnSuccess(200, "Aliases validated successfully").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[custom.WithCustomReader, custom.WithCustomReader](
		"POST", "/custom-reader", handleWithCustomReader).
		Summary("Custom reader as file").
		Description("Demonstrates a custom Reader type annotated with @oa:file, rendered as a binary file field in the OpenAPI schema.").
		Tags("Custom Types").
		OnSuccess(200, "Custom reader accepted").
		PossibleErr(http.StatusBadRequest, http.StatusInternalServerError).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[custom.WithEmbed, custom.WithEmbed](
		"GET", "/embed", handleWithEmbed).
		Summary("Embedded struct").
		Description("Demonstrates Go struct embedding. The embedded struct's fields are represented as a nested reference in the generated OpenAPI schema.").
		Tags("Custom Types").
		OnSuccess(200, "Embedded struct returned").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)
	nooa.RegisterScalar("", spec, mux)

	log.Println("Server starting on http://localhost:9090")
	log.Println("Swagger UI: http://localhost:9090/docs/")
	log.Println("Raw JSON:   http://localhost:9090/openapi.json")
	log.Println("Scalar UI:  http://localhost:9090/scalar/")
	log.Fatal(http.ListenAndServe(":9090", mux))
}

const CTJSON = "application/json"

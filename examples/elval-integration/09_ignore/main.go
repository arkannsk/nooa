package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	ign "github.com/arkannsk/nooa/examples/models/09_ignore"
)

func handleWithIgnoredField(w http.ResponseWriter, r *http.Request) {
	resp := ign.WithIgnoredField{
		Public:   "visible data",
		Internal: "this will not appear in OpenAPI schema",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleWithOverride(w http.ResponseWriter, r *http.Request) {
	resp := ign.WithOverride{
		File: &ign.IgnoredType{
			Data: []byte("binary data"),
		},
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleOnlyIgnoredFields(w http.ResponseWriter, r *http.Request) {
	resp := ign.OnlyIgnoredFields{
		Field1: "ignored",
		Field2: 42,
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "09 Ignore Demo",
		Version:     "1.0.0",
		Description: "Integration example: @oa:ignore annotation on types and fields",
	})

	spec.AddTag("Ignore", "Исключение полей и типов из OpenAPI через @oa:ignore")

	nooa.NewRoute[ign.WithIgnoredField, ign.WithIgnoredField](
		"GET", "/ignored-field", handleWithIgnoredField).
		Summary("Struct with an ignored field").
		Description("Demonstrates @oa:ignore on a struct field. The `Public` field appears in the OpenAPI schema, but `Internal` (annotated with @oa:ignore) is excluded.").
		Tags("Ignore").
		OnSuccess(200, "Struct with ignored field retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[ign.WithOverride, ign.WithOverride](
		"GET", "/override", handleWithOverride).
		Summary("Field overrides ignored type").
		Description("Demonstrates @oa:file overriding an @oa:ignore type. The `IgnoredType` struct is ignored").
		Tags("Ignore").
		OnSuccess(200, "Override struct retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[ign.OnlyIgnoredFields, ign.OnlyIgnoredFields](
		"GET", "/only-ignored", handleOnlyIgnoredFields).
		Summary("Struct with all fields ignored").
		Description("Demonstrates a struct where all fields are annotated with @oa:ignore. The resulting OpenAPI schema has an empty properties map.").
		Tags("Ignore").
		OnSuccess(200, "All-ignored struct retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)
	nooa.RegisterScalar("", spec, mux)

	log.Println("Server starting on http://localhost:9090")
	log.Println("Swagger UI: http://localhost:9090/docs/")
	log.Println("Raw JSON:   http://localhost:9090/openapi.json")
	log.Fatal(http.ListenAndServe(":9090", mux))
}

const CTJSON = "application/json"

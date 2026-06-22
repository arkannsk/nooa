package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	edge "github.com/arkannsk/nooa/examples/models/11_edge_cases"
)

func handleEmptyStruct(w http.ResponseWriter, r *http.Request) {
	resp := edge.EmptyStruct{}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handlePointerChain(w http.ResponseWriter, r *http.Request) {
	s := "hello"
	resp := edge.PointerChain{
		Level1: &s,
		Level3: &edge.Nested{Value: "nested"},
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleCircularRef(w http.ResponseWriter, r *http.Request) {
	resp := edge.CircularRefA{}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleInterfaceOneOf(w http.ResponseWriter, r *http.Request) {
	resp := edge.WithInterface{
		Data: edge.StringEdgeValue{Value: "hello"},
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleUntypedNil(w http.ResponseWriter, r *http.Request) {
	resp := edge.WithUntypedNil{}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "11 Edge Cases Demo",
		Version:     "1.0.0",
		Description: "Integration example: edge cases such as empty structs, pointer chains, circular references, oneOf interfaces, and untyped nil fields",
	})

	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	spec.AddTag("Edge Cases", "Пограничные случаи: пустые структуры, цепочки указателей, циклические ссылки, oneOf, nil")

	nooa.NewRoute[edge.EmptyStruct, edge.EmptyStruct](
		"GET", "/empty", handleEmptyStruct).
		Summary("Empty struct").
		Description("Demonstrates an empty struct with no fields. The generated schema has no properties and no required fields.").
		Tags("Edge Cases").
		OnSuccess(200, "Empty struct returned").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[edge.PointerChain, edge.PointerChain](
		"GET", "/pointer-chain", handlePointerChain).
		Summary("Pointer chain").
		Description("Demonstrates multiple levels of pointers: *string, **string, and *struct. Tests that pointer types are handled correctly in the schema.").
		Tags("Edge Cases").
		OnSuccess(200, "Pointer chain returned").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[edge.CircularRefA, edge.CircularRefA](
		"GET", "/circular", handleCircularRef).
		Summary("Circular reference").
		Description("Demonstrates circular references between two structs (A -> B -> A). Tests that schema generation handles cycles without infinite recursion.").
		Tags("Edge Cases").
		OnSuccess(200, "Circular reference returned").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[edge.WithInterface, edge.WithInterface](
		"GET", "/oneof", handleInterfaceOneOf).
		Summary("Interface with oneOf").
		Description("Demonstrates an any/interface field annotated with @oa:oneOf. The schema generates a oneOf constraint referencing the allowed concrete types.").
		Tags("Edge Cases").
		OnSuccess(200, "oneOf interface returned").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[edge.WithUntypedNil, edge.WithUntypedNil](
		"GET", "/untyped-nil", handleUntypedNil).
		Summary("Untyped nil fields").
		Description("Demonstrates nullable pointer fields: *string, *struct, and *[]slice. Tests that untyped nil pointers are represented correctly in the schema.").
		Tags("Edge Cases").
		OnSuccess(200, "Untyped nil struct returned").
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

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/arkannsk/nooa"
	httpparams "github.com/arkannsk/nooa/examples/models/08_http_params"
)

func handleQueryParams(w http.ResponseWriter, r *http.Request) {
	params := httpparams.QueryParams{}
	if err := params.ParseRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Если ParseRequest не заполнил — подставим дефолтные значения
	if params.Query == "" {
		params.Query = "golang"
	}
	if params.Page == 0 {
		params.Page = 1
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(params)
}

func handlePathParams(w http.ResponseWriter, r *http.Request) {
	params := httpparams.PathParams{}
	if err := params.ParseRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if params.UserID == "" {
		params.UserID = "usr_123"
	}
	if params.ResourceID == 0 {
		params.ResourceID = 42
	}
	params.Payload = "updated"

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(params)
}

func handleHeaderParams(w http.ResponseWriter, r *http.Request) {
	params := httpparams.HeaderParams{}
	if err := params.ParseRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if params.APIKey == "" {
		params.APIKey = "demo-key"
	}
	if params.RequestID == "" {
		params.RequestID = "req-001"
	}
	params.Content = "hello from headers"

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(params)
}

func handleMixedParams(w http.ResponseWriter, r *http.Request) {
	params := httpparams.MixedParams{}
	if err := params.ParseRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if params.ID == "" {
		params.ID = "item-1"
	}
	if params.Filter == "" {
		params.Filter = "active"
	}
	if params.AuthToken == "" {
		params.AuthToken = "token-abc"
	}
	params.Data = "mixed payload"

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(params)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "08 HTTP Parameters Demo",
		Version:     "1.0.0",
		Description: "Integration example: query, path, and header parameters with @oa:in annotations",
	})

	spec.AddTag("Parameters", "HTTP-параметры: query, path, header через @oa:in")

	nooa.NewRoute[httpparams.QueryParams, httpparams.QueryParams](
		"GET", "/search", handleQueryParams).
		Summary("Search with query parameters").
		Description("Demonstrates query parameters: q, page, limit, status. Fields annotated with @oa:in query are documented as operation parameters.").
		Tags("Parameters").
		OnSuccess(200, "Search results").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[httpparams.PathParams, httpparams.PathParams](
		"PUT", "/users/{userId}/resources/{resource_id}", handlePathParams).
		Summary("Update resource by path parameters").
		Description("Demonstrates path parameters: userId, resource_id. Fields annotated with @oa:in path are documented as required path parameters.").
		Tags("Parameters").
		OnSuccess(200, "Resource updated").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[httpparams.HeaderParams, httpparams.HeaderParams](
		"POST", "/header-demo", handleHeaderParams).
		Summary("Request with header parameters").
		Description("Demonstrates header parameters: X-API-Key, request-id. Fields annotated with @oa:in header are documented as header parameters.").
		Tags("Parameters").
		OnSuccess(200, "Header params received").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[httpparams.MixedParams, httpparams.MixedParams](
		"GET", "/items/{id}", handleMixedParams).
		Summary("Mixed parameter locations").
		Description("Demonstrates a combination of path, query, and header parameters in a single struct. Each field's location is determined by its @oa:in annotation.").
		Tags("Parameters").
		OnSuccess(200, "Item retrieved").
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

// unused import to keep url in go.mod if needed
var _ = url.PathEscape

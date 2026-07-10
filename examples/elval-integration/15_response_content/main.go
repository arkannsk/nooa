package main

//go:generate go run github.com/arkannsk/nooa/cmd/clientgen -pkg . -out ../../clients/15_response_content/client.go

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	response_content "github.com/arkannsk/nooa/examples/models/15_response_content"
)

func handleUserResponse(w http.ResponseWriter, r *http.Request) {
	resp := response_content.UserResponse{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req response_content.CreateUserResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := response_content.CreateUserResponse{
		ID:        42,
		Name:      req.Name,
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	w.Header().Set("Content-Type", CTJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func handleErrorResponse(w http.ResponseWriter, r *http.Request) {
	resp := response_content.ErrorResponse{
		Code:    400,
		Message: "Bad request",
	}

	w.Header().Set("Content-Type", CTJSON)
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}

func handleMultipleResponses(w http.ResponseWriter, r *http.Request) {
	resp := response_content.MultipleResponses{
		Data: "success",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleNoContentType(w http.ResponseWriter, r *http.Request) {
	resp := response_content.NoContentType{
		Value: "hello",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleNoMediaTypes(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "15 Response Content Demo",
		Version:     "1.0.0",
		Description: "Demonstrates @oa:response annotations from elval. Models define HTTP status codes and media types via annotations, and nooa automatically includes these responses in the OpenAPI spec.",
	})

	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	spec.AddTag("Responses", "Ответы с @oa:response аннотациями")
	spec.AddTag("MediaTypes", "Разные media types: JSON, XML")
	spec.AddTag("MultiStatus", "Множественные status code")

	// GET /user — ответ с несколькими media types (JSON + XML)
	// Модель UserResponse имеет @oa:response "200" "application/json,application/xml"
	nooa.NewRoute[struct{}, response_content.UserResponse](
		"GET", "/user", handleUserResponse).
		Summary("Get user response").
		Description("UserResponse model defines @oa:response annotation with status 200 and media types application/json and application/xml. nooa automatically includes these responses in the OpenAPI spec.").
		Tags("Responses", "MediaTypes").
		OnSuccess(200, "User retrieved").
		RegisterSpecAndMux(mux, spec)

	// POST /create — ответ с несколькими status code (200 и 201)
	// Модель CreateUserResponse имеет @oa:response "200" и @oa:response "201"
	nooa.NewRoute[response_content.CreateUserResponse, response_content.CreateUserResponse](
		"POST", "/create", handleCreateUser).
		Summary("Create user").
		Description("CreateUserResponse model defines two @oa:response annotations: status 200 with JSON+XML and status 201 with JSON only. nooa merges these responses into the OpenAPI spec.").
		Tags("Responses", "MultiStatus").
		OnSuccess(201, "User created").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// POST /error — ответ с ошибкой
	// Модель ErrorResponse имеет @oa:response "400" "application/json"
	nooa.NewRoute[struct{}, response_content.ErrorResponse](
		"POST", "/error", handleErrorResponse).
		Summary("Error response").
		Description("ErrorResponse model defines @oa:response annotation with status 400 and media type application/json.").
		Tags("Responses").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// GET /multi — ответ с тремя status code
	// Модель MultipleResponses имеет @oa:response "200", "401", "500"
	nooa.NewRoute[struct{}, response_content.MultipleResponses](
		"GET", "/multi", handleMultipleResponses).
		Summary("Multiple response codes").
		Description("MultipleResponses model defines three @oa:response annotations: status 200, 401, and 500, all with application/json.").
		Tags("MultiStatus").
		OnSuccess(200, "Success").
		RegisterSpecAndMux(mux, spec)

	// GET /no-content — структура без response аннотации
	nooa.NewRoute[struct{}, response_content.NoContentType](
		"GET", "/no-content", handleNoContentType).
		Summary("No response annotation").
		Description("NoContentType has no @oa:response annotation, so no model responses are generated. Only explicit OnSuccess responses apply.").
		Tags("Responses").
		OnSuccess(200, "OK").
		RegisterSpecAndMux(mux, spec)

	// GET /no-media — ответ без media types (204 No Content)
	nooa.NewRoute[struct{}, response_content.NoMediaTypes](
		"GET", "/no-media", handleNoMediaTypes).
		Summary("No media types").
		Description("NoMediaTypes has @oa:response annotation with empty media types, resulting in a response without content body.").
		Tags("Responses").
		OnNoContent(204, "No content").
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

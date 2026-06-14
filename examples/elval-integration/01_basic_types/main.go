package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	basic_types "github.com/arkannsk/nooa/examples/models/01_basic_types"
)

func createPrimitives(w http.ResponseWriter, r *http.Request) {
	var req basic_types.SimplePrimitives
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func getPointers(w http.ResponseWriter, r *http.Request) {
	res := basic_types.WithPointers{
		Name: strPtr("optional_name"),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func createDefaults(w http.ResponseWriter, r *http.Request) {
	var req basic_types.WithDefaults
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func strPtr(s string) *string {
	return &s
}

func main() {
	mux := http.NewServeMux()

	// Создаем спецификацию
	spec := nooa.NewSpec(nooa.Info{
		Title:       "01 Basic Types Demo",
		Version:     "1.0.0",
		Description: "Integration example: primitive Go types mapped to OpenAPI schemas",
	})

	// Регистрируем ошибки (модели регистрируются автоматически)
	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusNotFound, new(models.NotFoundError), "Resource not found")
	spec.AddError(http.StatusUnauthorized, new(models.UnauthorizedError), "Unauthorized")
	spec.AddError(http.StatusForbidden, new(models.ForbiddenError), "Forbidden")
	spec.AddError(http.StatusConflict, new(models.ConflictError), "Conflict")
	spec.AddError(http.StatusTooManyRequests, new(models.RateLimitError), "Rate limit exceeded")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	// POST /primitives — примитивные типы
	nooa.NewRoute[basic_types.SimplePrimitives, basic_types.SimplePrimitives](
		"POST", "/primitives", createPrimitives).
		Summary("Create with primitive types").
		Description("Demonstrates all Go primitive types mapped to OpenAPI schemas").
		Tags("BasicTypes").
		OnSuccess(201, "Primitive object created").
		PossibleErr(http.StatusBadRequest, http.StatusUnauthorized).
		RegisterSpecAndMux(mux, spec)

	// GET /pointers — получение объекта с указателями
	nooa.NewRoute[basic_types.WithPointers, basic_types.WithPointers](
		"GET", "/pointers", getPointers).
		Summary("Get nullable pointer fields").
		Description("Demonstrates pointer fields (nullable) in OpenAPI").
		Tags("BasicTypes").
		OnSuccess(200, "Pointer fields retrieved").
		Register(mux).
		RegisterSpec(spec)

	// POST /defaults — объект со значениями по умолчанию
	nooa.NewRoute[basic_types.WithDefaults, basic_types.WithDefaults](
		"POST", "/defaults", createDefaults).
		Summary("Create with default values").
		Description("Demonstrates fields with default values in OpenAPI").
		Tags("BasicTypes").
		OnSuccess(201, "Defaults object created").
		PossibleErr(http.StatusBadRequest).
		Register(mux).
		RegisterSpec(spec)

	// Монтируем документацию
	nooa.RegisterVersionedAPI("", spec, mux)

	log.Println("Server starting on http://localhost:8080")
	log.Println("Swagger UI: http://localhost:8080/docs/")
	log.Println("Raw JSON:   http://localhost:8080/openapi.json")

	log.Fatal(http.ListenAndServe(":9090", mux))
}

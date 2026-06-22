package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	mixed "github.com/arkannsk/nooa/examples/models/13_mixed"
)

func handleCreateMegaStruct(w http.ResponseWriter, r *http.Request) {
	var req mixed.MegaStruct
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

func handleGetMegaStruct(w http.ResponseWriter, r *http.Request) {
	var params mixed.MegaStruct
	if err := params.ParseRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fill defaults for fields not parsed from request
	if params.ID == "" {
		params.ID = "mega-001"
	}
	if params.Name == "" {
		params.Name = "Demo User"
	}
	if params.Avatar == nil {
		// Avatar is required by validation, but for a GET demo we skip it
	}
	if params.Status == "" {
		params.Status = "active"
	}
	if params.Tags == nil {
		params.Tags = []string{"demo", "mixed"}
	}
	params.Address = mixed.Address{
		Street:  "Main Street 1",
		City:    "Springfield",
		Country: "US",
	}
	params.Variant = mixed.UserVariant{
		Kind:     "user",
		Username: "demo_user",
	}
	if params.APIVersion == "" {
		params.APIVersion = "v1"
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(params)
}

func handleUserVariant(w http.ResponseWriter, r *http.Request) {
	resp := mixed.UserVariant{
		Kind:     "user",
		Username: "alice",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleAdminVariant(w http.ResponseWriter, r *http.Request) {
	resp := mixed.AdminVariant{
		Kind:       "admin",
		AdminLevel: 3,
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleAddress(w http.ResponseWriter, r *http.Request) {
	resp := mixed.Address{
		Street:  "Oak Avenue 42",
		City:    "Portland",
		Country: "US",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "13 Mixed Features Demo",
		Version:     "1.0.0",
		Description: "Comprehensive integration example combining all features: primitives, files/streams, collections, nested structs, generics, polymorphism, HTTP parameters, ignored fields, and custom types",
	})

	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	spec.AddTag("Mixed", "Комплексный пример, объединяющий все фичи nooa")
	spec.AddTag("Variants", "Полиморфные варианты: UserVariant и AdminVariant")
	spec.AddTag("Nested", "Вложенные структуры и их маршруты")

	spec.AddSecurityScheme("bearerAuth", nooa.SecuritySchemeBearer("JWT authorization"))
	spec.AddSecurityScheme("basicAuth", nooa.SecuritySchemeBasic("HTTP Basic auth"))
	spec.AddSecurityScheme("oauth2", nooa.SecuritySchemeOAuth2("oauth2", "OAuth2 authorization code flow", nooa.OAuth2Config{
		Flow:     nooa.OAuth2FlowAuthorizationCode,
		AuthURL:  "https://example.com/oauth/authorize",
		TokenURL: "https://example.com/oauth/token",
		Scopes: []nooa.OAuth2Scope{
			{Name: "read", Description: "Read access"},
			{Name: "write", Description: "Write access"},
		},
	}))
	spec.DefaultSecurity(nooa.SecurityRequirement{Scheme: "bearerAuth", Scopes: []string{"read", "write"}})

	// POST /mega — полный пример со всеми фичами (body + валидация)
	nooa.NewRoute[mixed.MegaStruct, mixed.MegaStruct](
		"POST", "/mega", handleCreateMegaStruct).
		Summary("Create mega struct").
		Description("Comprehensive example combining all features: primitives with validators, file/stream fields, enum and slice collections, nested structs, generics, oneOf polymorphism with discriminator, HTTP parameters (path/query/header), ignored fields, and custom types. All constraints defined via @evl:validate and @oa:* annotations.").
		Tags("Mixed").
		OnSuccess(200, "Mega struct created successfully").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// GET /mega — демонстрация парсинга HTTP-параметров
	nooa.NewRoute[mixed.MegaStruct, mixed.MegaStruct](
		"GET", "/mega", handleGetMegaStruct).
		Summary("Get mega struct from HTTP parameters").
		Description("Demonstrates parsing path, query, and header parameters from a request into the MegaStruct. Fields annotated with @oa:in path/query/header are extracted automatically via ParseRequest().").
		Tags("Mixed").
		OnSuccess(200, "Mega struct retrieved from parameters").
		RegisterSpecAndMux(mux, spec)

	// GET /variant/user — пользовательский вариант
	nooa.NewRoute[mixed.UserVariant, mixed.UserVariant](
		"GET", "/variant/user", handleUserVariant).
		Summary("Get user variant").
		Description("Returns a UserVariant, one of the polymorphic options in the MegaStruct.Variant oneOf field.").
		Tags("Variants").
		OnSuccess(200, "User variant retrieved").
		RegisterSpecAndMux(mux, spec)

	// GET /variant/admin — админский вариант
	nooa.NewRoute[mixed.AdminVariant, mixed.AdminVariant](
		"GET", "/variant/admin", handleAdminVariant).
		Summary("Get admin variant").
		Description("Returns an AdminVariant, one of the polymorphic options in the MegaStruct.Variant oneOf field.").
		Tags("Variants").
		OnSuccess(200, "Admin variant retrieved").
		RegisterSpecAndMux(mux, spec)

	// GET /address — вспомогательная вложенная структура
	nooa.NewRoute[mixed.Address, mixed.Address](
		"GET", "/address", handleAddress).
		Summary("Get address").
		Description("Returns an Address struct, used as a nested field in MegaStruct.").
		Tags("Nested").
		OnSuccess(200, "Address retrieved").
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

// suppress unused import warning
var _ = os.File{}

package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	structtags "github.com/arkannsk/nooa/examples/models/14_struct_tags"
)

func handleJsonTag(w http.ResponseWriter, r *http.Request) {
	var req structtags.JsonTagModel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(req)
}

func handleYamlTag(w http.ResponseWriter, r *http.Request) {
	resp := structtags.YamlTagModel{
		UserName: "yaml_user",
		FullName: "YAML Full Name",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleXmlTag(w http.ResponseWriter, r *http.Request) {
	resp := structtags.XmlTagModel{
		RecordID:    "rec-001",
		RecordTitle: "XML Record",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func handleMixedTags(w http.ResponseWriter, r *http.Request) {
	resp := structtags.MixedTagsModel{
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		Address:   "123 Main St",
		Secret:    "hidden",
	}

	w.Header().Set("Content-Type", CTJSON)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "14 Struct Tags Demo",
		Version:     "1.0.0",
		Description: "Demonstrates how nooa handles struct fields with json, yaml, and xml tags. OpenAPI schema property names are derived from struct tags (json > yaml > xml priority).",
	})

	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	spec.AddTag("StructTags", "Примеры обработки struct тегов: json, yaml, xml")

	// POST /json — структура с json тегами (snake_case, camelCase, omitempty, ignored)
	nooa.NewRoute[structtags.JsonTagModel, structtags.JsonTagModel](
		"POST", "/json", handleJsonTag).
		Summary("Handle JSON tags").
		Description("Struct with various json tags: snake_case, camelCase, omitempty, and ignored fields (json:\"-\").").
		Tags("StructTags").
		OnSuccess(200, "JSON tags processed").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	// GET /yaml — структура с yaml тегами
	nooa.NewRoute[structtags.YamlTagModel, structtags.YamlTagModel](
		"GET", "/yaml", handleYamlTag).
		Summary("Handle YAML tags").
		Description("Struct with yaml tags. When both json and yaml tags are present, json takes priority for OpenAPI property names.").
		Tags("StructTags").
		OnSuccess(200, "YAML tags processed").
		RegisterSpecAndMux(mux, spec)

	// GET /xml — структура с xml тегами
	nooa.NewRoute[structtags.XmlTagModel, structtags.XmlTagModel](
		"GET", "/xml", handleXmlTag).
		Summary("Handle XML tags").
		Description("Struct with xml tags. Priority: json > yaml > xml. If only xml tag is present, it is used for the property name.").
		Tags("StructTags").
		OnSuccess(200, "XML tags processed").
		RegisterSpecAndMux(mux, spec)

	// GET /mixed — смешанные теги
	nooa.NewRoute[structtags.MixedTagsModel, structtags.MixedTagsModel](
		"GET", "/mixed", handleMixedTags).
		Summary("Handle mixed tags").
		Description("Struct combining json tags, yaml-only tags, and fields without any tags (fallback to lowercase name).").
		Tags("StructTags").
		OnSuccess(200, "Mixed tags processed").
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

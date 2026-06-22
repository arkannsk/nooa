package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/arkannsk/nooa/examples/elval-integration/models"
	val "github.com/arkannsk/nooa/examples/models/10_validators"
)

func handleStringValidators(w http.ResponseWriter, r *http.Request) {
	var req val.AllStringValidators
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

func handleNumericValidators(w http.ResponseWriter, r *http.Request) {
	var req val.AllNumericValidators
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

func handleEnumAndSliceValidators(w http.ResponseWriter, r *http.Request) {
	var req val.AllEnumAndSliceValidators
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

func handleDateAndDurationValidators(w http.ResponseWriter, r *http.Request) {
	var req val.DateAndDurationValidators
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

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "10 Validators Demo",
		Version:     "1.0.0",
		Description: "Integration example: @evl:validate annotations for string, numeric, enum, slice, and date validation",
	})

	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
	spec.AddError(http.StatusInternalServerError, new(models.APIError), "Internal server error")

	spec.AddTag("Validators", "Валидация через @evl:validate: строки, числа, enum, слайсы, даты")

	nooa.NewRoute[val.AllStringValidators, val.AllStringValidators](
		"POST", "/validate/string", handleStringValidators).
		Summary("Validate string fields").
		Description("Demonstrates string validators: required, min/max length, email, phone, uuid, custom regex, contains, starts_with, ends_with. All constraints defined via @evl:validate annotations.").
		Tags("Validators").
		OnSuccess(200, "String fields validated successfully").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[val.AllNumericValidators, val.AllNumericValidators](
		"POST", "/validate/numeric", handleNumericValidators).
		Summary("Validate numeric fields").
		Description("Demonstrates numeric validators: min, max, gt, gte, lt, lte, not-zero, eq, neq. All constraints defined via @evl:validate annotations.").
		Tags("Validators").
		OnSuccess(200, "Numeric fields validated successfully").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[val.AllEnumAndSliceValidators, val.AllEnumAndSliceValidators](
		"POST", "/validate/enum-slice", handleEnumAndSliceValidators).
		Summary("Validate enum and slice fields").
		Description("Demonstrates enum validation (allowed values) and slice constraints (required, not-zero, min/max length, fixed length). All constraints defined via @evl:validate annotations.").
		Tags("Validators").
		OnSuccess(200, "Enum and slice fields validated successfully").
		PossibleErr(http.StatusBadRequest).
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[val.DateAndDurationValidators, val.DateAndDurationValidators](
		"POST", "/validate/date", handleDateAndDurationValidators).
		Summary("Validate date and duration fields").
		Description("Demonstrates date/duration validators: after, before, not-zero. All constraints defined via @evl:validate annotations.").
		Tags("Validators").
		OnSuccess(200, "Date fields validated successfully").
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

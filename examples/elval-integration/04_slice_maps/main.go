package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	collections "github.com/arkannsk/nooa/examples/models/04_slice_maps"
)

func handleSliceVariations(w http.ResponseWriter, r *http.Request) {
	resp := collections.SliceVariations{
		Tags: []string{"a", "b", "c"},
		IDs:  []int{1, 2, 3},
		Items: []collections.Item{
			{ID: "item_1", Name: "First Item"},
			{ID: "item_2", Name: "Second Item"},
		},
		OptionalTags: &[]string{"optional"},
		PtrItems: []*collections.Item{
			{ID: "ptr_1", Name: "Pointer Item"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleMapVariations(w http.ResponseWriter, r *http.Request) {
	resp := collections.MapVariations{
		Metadata: map[string]string{
			"key": "value",
		},
		Counts: map[string]int{
			"visits": 42,
		},
		Items: map[string]collections.Item{
			"first": {ID: "item_1", Name: "First Item"},
		},
		Complex: map[string]map[string]string{
			"nested": {
				"deep": "value",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleArrayFixed(w http.ResponseWriter, r *http.Request) {
	resp := collections.ArrayFixed{
		Coords: [3]float64{40.7128, -74.0060, 10},
		Tags:   [10]string{"tag1", "tag2"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "04 Slice/Map Demo",
		Version:     "1.0.0",
		Description: "Integration example: slices, maps, and fixed arrays",
	})

	nooa.NewRoute[collections.SliceVariations, collections.SliceVariations](
		"GET", "/slices", handleSliceVariations).
		Summary("Get slice variations").
		Description("Returns a struct with various slice types including nested struct slices").
		Tags("Collections").
		OnSuccess(200, "Slices retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[collections.MapVariations, collections.MapVariations](
		"GET", "/maps", handleMapVariations).
		Summary("Get map variations").
		Description("Returns a struct with various map types including nested maps").
		Tags("Collections").
		OnSuccess(200, "Maps retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[collections.ArrayFixed, collections.ArrayFixed](
		"GET", "/arrays", handleArrayFixed).
		Summary("Get fixed arrays").
		Description("Returns a struct with fixed-size arrays and min/max constraints").
		Tags("Collections").
		OnSuccess(200, "Arrays retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)

	log.Println("Server starting on http://localhost:9093")
	log.Println("Swagger UI: http://localhost:9093/docs/")
	log.Println("Raw JSON:   http://localhost:9093/openapi.json")
	log.Fatal(http.ListenAndServe(":9093", mux))
}

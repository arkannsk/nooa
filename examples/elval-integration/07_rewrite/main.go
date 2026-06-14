package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	rw "github.com/arkannsk/nooa/examples/models/07_rewrite"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func handleRewriteType(w http.ResponseWriter, r *http.Request) {
	resp := rw.WithRewriteType{
		RawJSON:  json.RawMessage(`{"key":"value"}`),
		CustomID: rw.MyID(42),
		CustomList: rw.MyList{
			"item1", "item2", "item3",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleRewriteRef(w http.ResponseWriter, r *http.Request) {
	resp := rw.WithRewriteRef{
		Meta: rw.CommonMetadata{
			CreatedAt: "2024-01-01T00:00:00Z",
			UpdatedAt: "2024-06-15T12:00:00Z",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleGeoFeature(w http.ResponseWriter, r *http.Request) {
	resp := rw.CreateLocationRequest{
		Feature: geojson.Feature{
			Type:     "Feature",
			Geometry: orb.Point{37.6173, 55.7558},
			Properties: geojson.Properties{
				"city": "Moscow",
			},
			BBox: geojson.BBox{49.1, 55.7, 49.3, 55.8},
		},
		UserID: "user-123",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleGeoPoint(w http.ResponseWriter, r *http.Request) {
	resp := rw.CreateLocationWithPoint{
		Location: orb.Point{37.6173, 55.7558},
		Name:     "Red Square",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "07 Rewrite Demo",
		Version:     "1.0.0",
		Description: "Integration example: type and reference rewriting in OpenAPI schemas",
	})

	// Регистрируем схемы-заглушки на которые ссылаются rewrite.ref.
	// Имя должно совпадать с тем что в $ref.
	// TODO: generate without register!
	spec.RegisterModel("CommonMetadata", new(rw.CommonMetadata))
	spec.RegisterModel("FeatureDocs", new(rw.FeatureDocs))
	spec.RegisterModel("GeometryDocs", new(rw.GeometryDocs))
	spec.RegisterModel("PointDocs", new(rw.PointDocs))

	nooa.NewRoute[rw.WithRewriteType, rw.WithRewriteType](
		"GET", "/rewrite-type", handleRewriteType).
		Summary("Get struct with rewritten types").
		Description("Demonstrates rewriting Go types to different OpenAPI types (json.RawMessage→string, custom ID→integer, custom list→array)").
		Tags("Rewrite").
		OnSuccess(200, "Rewrite type struct retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[rw.WithRewriteRef, rw.WithRewriteRef](
		"GET", "/rewrite-ref", handleRewriteRef).
		Summary("Get struct with rewritten references").
		Description("Demonstrates rewriting a field reference to a local stub schema via @oa:rewrite.ref").
		Tags("Rewrite").
		OnSuccess(200, "Rewrite ref struct retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[rw.CreateLocationRequest, rw.CreateLocationRequest](
		"GET", "/geo-feature", handleGeoFeature).
		Summary("Get location with geojson Feature").
		Description("Demonstrates replacing geojson.Feature type with our own FeatureDocs schema via @oa:rewrite.ref").
		Tags("Geo").
		OnSuccess(200, "GeoJSON feature retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[rw.CreateLocationWithPoint, rw.CreateLocationWithPoint](
		"GET", "/geo-point", handleGeoPoint).
		Summary("Get location with orb Point").
		Description("Demonstrates replacing orb.Point type with our own PointDocs schema via @oa:rewrite.ref").
		Tags("Geo").
		OnSuccess(200, "GeoJSON point retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)
	nooa.RegisterScalar("", spec, mux)

	log.Println("Server starting on http://localhost:9096")
	log.Println("Swagger UI: http://localhost:9096/docs/")
	log.Println("Raw JSON:   http://localhost:9096/openapi.json")
	log.Println("Scalar UI:  http://localhost:9096/scalar/")
	log.Fatal(http.ListenAndServe(":9096", mux))
}

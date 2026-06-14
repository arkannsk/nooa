package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/arkannsk/nooa"
	poly "github.com/arkannsk/nooa/examples/models/06_polymorphism"
)

func handleContainer(w http.ResponseWriter, r *http.Request) {
	resp := poly.Container{
		Shape: poly.Shape{
			Type: "circle",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCircle(w http.ResponseWriter, r *http.Request) {
	resp := poly.CircleShape{
		Type:   "circle",
		Radius: 5.0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleRectangle(w http.ResponseWriter, r *http.Request) {
	resp := poly.RectangleShape{
		Type:   "rectangle",
		Width:  3.0,
		Height: 4.0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleOneOf(w http.ResponseWriter, r *http.Request) {
	resp := poly.OneOfExample{
		Value: poly.StringValue{Value: "hello"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()

	spec := nooa.NewSpec(nooa.Info{
		Title:       "06 Polymorphism Demo",
		Version:     "1.0.0",
		Description: "Integration example: oneOf, discriminator, and polymorphic types",
	})

	nooa.NewRoute[poly.Container, poly.Container](
		"GET", "/container", handleContainer).
		Summary("Get container with shape").
		Description("Returns a container with a polymorphic shape field using discriminator").
		Tags("Polymorphism").
		OnSuccess(200, "Container retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[poly.CircleShape, poly.CircleShape](
		"GET", "/circle", handleCircle).
		Summary("Get circle shape").
		Description("Returns a circle geometry").
		Tags("Shapes").
		OnSuccess(200, "Circle shape retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[poly.RectangleShape, poly.RectangleShape](
		"GET", "/rectangle", handleRectangle).
		Summary("Get rectangle shape").
		Description("Returns a rectangle geometry").
		Tags("Shapes").
		OnSuccess(200, "Rectangle shape retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.NewRoute[poly.OneOfExample, poly.OneOfExample](
		"GET", "/oneof", handleOneOf).
		Summary("Get oneOf example").
		Description("Returns a value that can be a string or a number (oneOf without discriminator)").
		Tags("Polymorphism").
		OnSuccess(200, "OneOf value retrieved").
		RegisterSpecAndMux(mux, spec)

	nooa.RegisterVersionedAPI("", spec, mux)
	nooa.RegisterRedoc("", spec, mux)

	log.Println("Server starting on http://localhost:9095")
	log.Println("Swagger UI: http://localhost:9095/docs/")
	log.Println("Redoc UI:   http://localhost:9095/redoc/")
	log.Println("Raw JSON:   http://localhost:9095/openapi.json")
	log.Fatal(http.ListenAndServe(":9095", mux))
}

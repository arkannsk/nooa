package rewrite

import (
	"encoding/json"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// WithRewriteType — упрощение сложных типов
// @oa:description "Struct with type rewriting"
type WithRewriteType struct {
	// @oa:rewrite.type string
	// @oa:description "JSON payload documented as string"
	RawJSON json.RawMessage

	// @oa:rewrite.type integer
	// @oa:description "Custom ID type documented as integer"
	CustomID MyID

	// @oa:rewrite.type array
	// @oa:description "Custom slice documented as array"
	CustomList MyList
}

// MyID — алиас на int64
type MyID int64

// MyList — кастомный слайс
type MyList []string

// WithRewriteRef — ссылка на локальную схему-заглушку через rewrite.ref
// @oa:description "Struct referencing local stub schema"
type WithRewriteRef struct {
	// @oa:rewrite.ref "CommonMetadata"
	// @oa:description "Reference to local stub"
	Meta CommonMetadata `json:"meta"`
}

// CommonMetadata — локальная схема-заглушка
// @oa:rewrite.ref "CommonMetadata"
// @oa:description "Reusable metadata schema"
type CommonMetadata struct {
	// @oa:description "Creation timestamp"
	CreatedAt string `json:"created_at"`
	// @oa:description "Last update timestamp"
	UpdatedAt string `json:"updated_at"`
}

// CreateLocationRequest — запрос на создание локации с гео-фичей
// @oa:description "Request to create a location with a geojson feature"
type CreateLocationRequest struct {
	// Point to local stub
	// @oa:rewrite.ref "FeatureDocs"
	Feature geojson.Feature `json:"feature"`

	// @oa:description "User ID"
	UserID string `json:"user_id"`
}

// FeatureDocs — наша документация для geojson.Feature
// @oa:rewrite.ref "FeatureDocs"
// @oa:description "GeoJSON feature with type, geometry and properties"
type FeatureDocs struct {
	// @oa:enum Feature
	// @oa:description "Must be 'Feature'"
	Type string `json:"type"`

	// @oa:rewrite.ref "GeometryDocs"
	// @oa:description "GeoJSON geometry"
	Geometry GeometryDocs `json:"geometry"`

	// @oa:rewrite.type object
	// @oa:description "Feature properties (arbitrary JSON object)"
	Properties json.RawMessage `json:"properties"`

	// @oa:description "Feature bounding box"
	BBox []float64 `json:"bbox,omitempty"`
}

// GeometryDocs — наша документация для GeoJSON geometry
// @oa:rewrite.ref "GeometryDocs"
// @oa:description "GeoJSON geometry object"
type GeometryDocs struct {
	// @oa:enum Point,LineString,Polygon,MultiPoint,MultiLineString,MultiPolygon,GeometryCollection
	// @oa:description "Geometry type"
	Type string `json:"type"`

	// @oa:rewrite.type object
	// @oa:description "Geometry coordinates"
	Coordinates json.RawMessage `json:"coordinates"`
}

// CreateLocationWithPoint — запрос с точкой orb.Point
// @oa:description "Request to create a location with an orb point"
type CreateLocationWithPoint struct {
	// Point to local stub
	// @oa:rewrite.ref "PointDocs"
	Location orb.Point `json:"location"`

	// @oa:description "Name of the location"
	Name string `json:"name"`
}

// PointDocs — наша документация для orb.Point
// @oa:rewrite.ref "PointDocs"
// @oa:description "A geographic point represented as [longitude, latitude]"
type PointDocs struct {
	// @oa:description "Longitude"
	Longitude float64 `json:"longitude"`

	// @oa:description "Latitude"
	Latitude float64 `json:"latitude"`
}

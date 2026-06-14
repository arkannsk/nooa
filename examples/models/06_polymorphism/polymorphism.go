package polymorphism

// Shape — базовый интерфейс (документируется через oneOf)
// @oa:description "Geometric shape (polymorphic)"
// @oa:discriminator.propertyName "type"
// @oa:discriminator.mapping "circle:CircleShape"
// @oa:discriminator.mapping "rectangle:RectangleShape"
// @oa:oneOf "CircleShape,RectangleShape"
type Shape struct {
	// @oa:enum circle,rectangle
	Type string `json:"type"`
}

// CircleShape — реализация круга
// @oa:description "Circle geometry"
type CircleShape struct {
	// @oa:enum "circle"
	Type string `json:"type"`
	// @oa:description "Radius in meters"
	Radius float64 `json:"radius"`
}

// RectangleShape — реализация прямоугольника
// @oa:description "Rectangle geometry"
type RectangleShape struct {
	// @oa:enum "rectangle"
	Type string `json:"type"`
	// @oa:description "Width in meters"
	Width float64 `json:"width"`
	// @oa:description "Height in meters"
	Height float64 `json:"height"`
}

// Container — структура с полиморфным полем
// @oa:description "Container with polymorphic shape"
type Container struct {
	// @oa:description "Shape variant"
	Shape Shape `json:"shape"`
}

// OneOfExample — явное oneOf без дискриминатора
// @oa:description "Value that can be string or number"
type OneOfExample struct {
	// @oa:description "Flexible value"
	// @oa:oneOf "StringValue,NumberValue"
	Value any `json:"value"`
}

type StringValue struct {
	Value string `json:"value"`
}

type NumberValue struct {
	Value float64 `json:"value"`
}

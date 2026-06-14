package basic_types

// SimplePrimitives — все базовые типы
// @oa:description "Struct with all primitive Go types"
type SimplePrimitives struct {
	// @oa:description "String field"
	// @oa:example "hello"
	Name string

	// @oa:description "Integer field"
	// @oa:example 42
	Count int

	// @oa:description "64-bit integer"
	// @oa:example 9223372036854775807
	BigNumber int64

	// @oa:description "Float value"
	// @oa:example 3.14
	Rate float64

	// @oa:description "Boolean flag"
	// @oa:example true
	Active bool

	// @oa:description "Timestamp"
	// @oa:example "2024-01-15T10:30:00Z"
	CreatedAt string
}

// WithPointers — указатели на примитивы
// @oa:description "Struct with pointer fields (nullable in OpenAPI)"
type WithPointers struct {
	// @oa:description "Optional name"
	Name *string

	// @oa:description "Optional count"
	Count *int

	// @oa:description "Optional flag"
	Active *bool
}

// WithDefaults — поля со значениями по умолчанию
// @oa:description "Struct with default values"
type WithDefaults struct {
	// @oa:description "Status with default"
	// @oa:default "active"
	Status string

	// @oa:description "Limit with default"
	// @oa:default 100
	Limit int

	// @oa:description "Enabled with default"
	// @oa:default true
	Enabled bool
}

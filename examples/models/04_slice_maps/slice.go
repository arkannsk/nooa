package collections

// SliceVariations — разные виды слайсов
// @oa:description "Struct with various slice types"
type SliceVariations struct {
	// @oa:description "Slice of strings"
	// @oa:example ["a", "b", "c"]
	Tags []string

	// @oa:description "Slice of integers"
	IDs []int

	// @oa:description "Slice of nested structs"
	Items []Item

	// @oa:description "Optional slice (nullable)"
	OptionalTags *[]string

	// @oa:description "Slice of pointers to structs"
	PtrItems []*Item
}

// Item — элемент для коллекций
// @oa:description "Generic item"
type Item struct {
	// @oa:description "Item ID"
	ID string
	// @oa:description "Item name"
	Name string
}

// MapVariations — мапы с разными типами значений
// @oa:description "Struct with map fields"
type MapVariations struct {
	// @oa:description "String-to-string map"
	// @oa:example {"key": "value"}
	Metadata map[string]string

	// @oa:description "String-to-int map"
	Counts map[string]int

	// @oa:description "String-to-struct map"
	Items map[string]Item

	// @oa:description "Nested map"
	Complex map[string]map[string]string
}

// ArrayFixed — фиксированные массивы (OpenAPI: type: array, maxItems/minItems)
// @oa:description "Struct with fixed-size arrays"
type ArrayFixed struct {
	// @oa:description "Exactly 3 coordinates"
	// @oa:minItems 3
	// @oa:maxItems 3
	Coords [3]float64

	// @oa:description "Up to 10 tags"
	// @oa:maxItems 10
	Tags [10]string
}

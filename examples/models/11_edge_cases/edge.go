package edgecases

// EmptyStruct — полностью пустая структура
// @oa:description "Empty struct (no fields)"
type EmptyStruct struct{}

// OnlyIgnored — все поля игнорируются
// @oa:ignore
type OnlyIgnored struct {
	// @oa:ignore
	Field string
}

// PointerChain — цепочка указателей
// @oa:description "Multiple levels of pointers"
type PointerChain struct {
	// @oa:description "String pointer"
	Level1 *string
	// @oa:description "Pointer to pointer"
	Level2 **string
	// @oa:description "Pointer to struct"
	Level3 *Nested
}

type Nested struct {
	Value string
}

// CircularRefA — циклическая ссылка (проверка на бесконечную рекурсию)
// @oa:description "Node A (circular reference)"
type CircularRefA struct {
	// @oa:description "Reference to B"
	B *CircularRefB
}

// CircularRefB — обратная ссылка
// @oa:description "Node B (circular reference)"
type CircularRefB struct {
	// @oa:description "Reference to A"
	A *CircularRefA
}

// StringEdgeValue — строковое значение для oneOf теста
type StringEdgeValue struct {
	Value string `json:"value"`
}

// NumberEdgeValue — числовое значение для oneOf теста
type NumberEdgeValue struct {
	Value float64 `json:"value"`
}

// WithInterface — поле интерфейса (any)
// @oa:description "Struct with interface field"
type WithInterface struct {
	// @oa:description "Flexible value"
	// @oa:oneOf "StringEdgeValue,NumberEdgeValue"
	Data any
}

// WithUntypedNil — поле, которое может быть nil
// @oa:description "Struct with nullable fields"
type WithUntypedNil struct {
	// @oa:description "Nullable string"
	Optional *string
	// @oa:description "Nullable struct"
	Child *Nested
	// @oa:description "Nullable slice"
	Items *[]string
}

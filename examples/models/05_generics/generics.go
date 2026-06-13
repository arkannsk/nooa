package generics

// Option[T] — опциональное значение (как в mo.Option)
type Option[T any] struct {
	value    T
	hasValue bool
}

func (o Option[T]) IsPresent() bool  { return o.hasValue }
func (o Option[T]) Value() (T, bool) { return o.value, o.hasValue }

// Result[T, E] — результат с ошибкой
type Result[T any, E any] struct {
	value T
	err   E
}

func (r Result[T, E]) IsOK() bool    { return &r.err == new(E) }
func (r Result[T, E]) Value() (T, E) { return r.value, r.err }

// GenericStruct — структура с дженерик-полями
// @oa:description "Struct using generic wrappers"
type GenericStruct struct {
	// @oa:description "Optional name"
	Name Option[string]

	// @oa:description "Optional age"
	Age Option[int]

	// @oa:description "Result of computation"
	Data Result[string, string]

	// @oa:description "Slice of optional items"
	Items []Option[Item]
}

// Item — вспомогательный тип
// @oa:description "Generic item"
type Item struct {
	ID   string
	Name string
}

// CustomGeneric[T] — кастомный дженерик
// @oa:rewrite.type string
type CustomGeneric[T any] struct {
	data T
}

// WithCustomGeneric — использование кастомного дженерика
// @oa:description "Struct with custom generic type"
type WithCustomGeneric struct {
	// @oa:description "Custom generic field (rewritten as string)"
	// @oa:rewrite.type string
	Value CustomGeneric[string]
}

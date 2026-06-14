package ignore

// @oa:ignore
type InternalMarker struct {
	Secret string
}

// WithIgnoredField — поле с @oa:ignore
// @oa:description "Struct with ignored field"
type WithIgnoredField struct {
	// @oa:description "Public field"
	Public string

	// @oa:ignore
	// Это поле не попадёт в схему
	Internal string
}

// IgnoredWithOverride — игнорируемый тип, но поле с @oa:file
// @oa:ignore
type IgnoredType struct {
	Data []byte
}

// WithOverride — поле ссылается на игнорируемый тип, но имеет аннотацию
// @oa:description "Field overrides ignored type"
type WithOverride struct {
	// @oa:file
	// @oa:description "This field is included despite type being @oa:ignore"
	File *IgnoredType
}

// OnlyIgnoredFields — структура, где все поля игнорируются
// @oa:description "Struct with all fields ignored"
type OnlyIgnoredFields struct {
	// @oa:ignore
	Field1 string
	// @oa:ignore
	Field2 int
}

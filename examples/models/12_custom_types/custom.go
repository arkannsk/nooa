package customtypes

// TypeAlias — алиас на примитив
// @oa:rewrite.type string
type TypeAlias string

// CustomString — алиас с валидацией
// @oa:rewrite.type string
// @oa:description "Custom string type"
type CustomString string

// WithAliases — использование алиасов
// @oa:description "Struct with type aliases"
type WithAliases struct {
	// @oa:description "Aliased string"
	Name TypeAlias

	// @oa:description "Custom string with validation"
	// @evl:validate min:3
	// @evl:validate pattern:email
	Email CustomString
}

// CustomReaderWithFile — кастомный ридер с @oa:file
type CustomReader struct {
	data []byte
}

func (c *CustomReader) Read(p []byte) (int, error) {
	n := copy(p, c.data)
	return n, nil
}

// WithCustomReader — использование кастомного ридера
// @oa:description "Struct with custom reader marked as file"
type WithCustomReader struct {
	// @oa:file
	// @oa:description "File via custom reader"
	File *CustomReader
}

// EmbedStruct — встраивание структур
// @oa:description "Base fields"
type EmbedStruct struct {
	// @oa:description "Embedded ID"
	ID string
	// @oa:description "Embedded name"
	Name string
}

// WithEmbed — структура с встраиванием
// @oa:description "Struct with embedded fields"
type WithEmbed struct {
	EmbedStruct
	// @oa:description "Additional field"
	Extra string
}

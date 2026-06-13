package rewrite

import "encoding/json"

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

// WithRewriteRef — ссылка на внешнюю/локальную схему
// @oa:description "Struct referencing external schema"
type WithRewriteRef struct {
	// @oa:rewrite.ref "github.com/arkannsk/elval/examples/07_rewrite.ExternalSchema"
	// @oa:description "Reference to external schema"
	External ExternalType

	// @oa:rewrite.ref "CommonMetadata"
	// @oa:description "Reference to local stub"
	Meta CommonMetadata
}

// ExternalType — тип из "внешнего" пакета
type ExternalType struct {
	Field string
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

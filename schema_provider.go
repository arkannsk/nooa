package nooa

import "github.com/arkannsk/elval/pkg/oa"

// SchemaProvider - интерфейс для моделей, которые предоставляют свою собственную OpenAPI схему.
// Если модель реализует этот метод, Nooa будет использовать его вместо рефлексии.
type SchemaProvider interface {
	OaSchema() *oa.Schema
}

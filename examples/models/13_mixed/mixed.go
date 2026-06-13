package mixed

import (
	"io"
	"os"
)

// MegaStruct — комбинация ВСЕХ фич
// @oa:description "Comprehensive example with all features"
// @oa:discriminator.propertyName "kind"
// @oa:discriminator.mapping "user:UserVariant"
// @oa:discriminator.mapping "admin:AdminVariant"
type MegaStruct struct {
	// === Примитивы ===
	// @oa:in path
	ID string `json:"id"`

	// @oa:description "User name"
	// @evl:validate required
	// @evl:validate min:2
	// @evl:validate max:50
	Name string `json:"name"`

	// === Файлы/Стримы ===
	// @oa:file
	// @oa:description "Avatar image"
	// @evl:validate required
	Avatar *os.File `json:"avatar"`

	// @oa:stream
	// @oa:description "Raw payload"
	Payload io.Reader `json:"payload"`

	// @oa:format:byte
	// @oa:description "Base64 thumbnail"
	Thumbnail []byte `json:"thumbnail"`

	// === Коллекции ===
	// @oa:enum "active","inactive"
	// @evl:validate required
	Status string `json:"status"`

	// @evl:validate min:1
	// @evl:validate max:10
	Tags []string `json:"tags"`

	// === Вложенные структуры ===
	// @oa:rewrite.ref "github.com/arkannsk/elval/examples/13_mixed.Address"
	Address Address `json:"address"`

	// === Дженерики ===
	// @oa:description "Optional email"
	Email Option[string] `json:"email"`

	// === Полиморфизм ===
	// @oa:oneOf "UserVariant,AdminVariant"
	Variant any `json:"variant"`

	// === HTTP-параметры ===
	// @oa:in query
	// @oa:description "Include deleted"
	IncludeDeleted bool `json:"include_deleted"`

	// @oa:in header
	// @oa:description "API version"
	APIVersion string `json:"X-API-Version"`

	// === Игнорируемые поля ===
	// @oa:ignore
	internalCache map[string]any

	// === Кастомные типы ===
	// @oa:rewrite.type string
	CustomID MyID `json:"custom_id"`
}

// Address — вспомогательная структура
// @oa:rewrite.ref "Address"
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

// Option[T] — опциональное значение
type Option[T any] struct {
	value T
	ok    bool
}

func (o Option[T]) IsPresent() bool { return o.ok }

// MyID — кастомный ID
// @oa:rewrite.type integer
type MyID int64

// UserVariant — вариант полиморфного поля
// @oa:enum "user"
type UserVariant struct {
	Kind     string `json:"kind"`
	Username string `json:"username"`
}

// AdminVariant — другой вариант
// @oa:enum "admin"
type AdminVariant struct {
	Kind       string `json:"kind"`
	AdminLevel int    `json:"admin_level"`
}

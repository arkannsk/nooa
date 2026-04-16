package models

// CreateUserRequest — тип входящего запроса.
// Теги openapi:"..." используются генератором спецификации.
type CreateUserRequest struct {
	// Name — отображаемое имя пользователя.
	// +required
	Name string `json:"name" openapi:"required,desc=Display name,example=Alice"`

	// Email — контактный email.
	// +required
	Email string `json:"email" openapi:"required,desc=Contact email,example=alice@example.com"`

	// Age — возраст в годах (опционально).
	Age *int `json:"age,omitempty" openapi:"desc=Age in years,min=0,max=120"`
}

// User — тип успешного ответа.
type User struct {
	// ID — уникальный идентификатор пользователя.
	ID string `json:"id" openapi:"desc=UUID v4"`

	// Name — отображаемое имя.
	Name string `json:"name"`

	// Email — контактный email.
	Email string `json:"email"`

	// Age — возраст (может отсутствовать).
	Age *int `json:"age,omitempty"`

	// CreatedAt — время создания (RFC3339).
	CreatedAt string `json:"created_at" openapi:"desc=Creation timestamp,format=date-time"`
}

// Error — стандартный формат ошибки (RFC 7807 compatible).
type Error struct {
	// Code — машинный код ошибки.
	Code string `json:"code" openapi:"example=VALIDATION_FAILED"`

	// Message — человекочитаемое описание.
	Message string `json:"message" openapi:"example=Email format is invalid"`

	// Details — дополнительные данные (опционально).
	Details map[string]any `json:"details,omitempty"`
}

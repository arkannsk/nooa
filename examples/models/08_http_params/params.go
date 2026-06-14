package httpparams

// QueryParams — параметры в query string
// @oa:description "Request with query parameters"
type QueryParams struct {
	// @oa:in query
	// @oa:description "Search keyword"
	// @oa:example "golang"
	Query string `json:"q"`

	// @oa:in query
	// @oa:description "Page number"
	// @oa:example 1
	Page int `json:"page"`

	// @oa:in query
	// @oa:description "Items per page"
	// @oa:example 20
	Limit int `json:"limit"`

	// @oa:in query
	// @oa:description "Filter by status"
	// @oa:enum active,inactive,pending
	Status string `json:"status"`

	// Обычное поле тела (не параметр)
	// @oa:description "Request body field"
	BodyField string `json:"body_field"`
}

// PathParams — параметры в пути
// @oa:description "Request with path parameters"
type PathParams struct {
	// @oa:in path userId
	// @oa:description "User ID"
	// @oa:example "usr_123"
	UserID string `json:"user_id"`

	// @oa:in path resource_id
	// @oa:description "Resource ID"
	ResourceID int `json:"resource_id"`

	// Поле тела
	// @oa:description "Update payload"
	Payload string `json:"payload"`
}

// HeaderParams — параметры в заголовках
// @oa:description "Request with header parameters"
type HeaderParams struct {
	// @oa:in header X-API-Key
	// @oa:description "API key"
	APIKey string

	// @oa:in header request-id
	// @oa:description "Request ID for tracing"
	RequestID string

	// Поле тела
	// @oa:description "Request content"
	Content string
}

// MixedParams — комбинация всех типов параметров
// @oa:description "Request with mixed parameter locations"
type MixedParams struct {
	// @oa:in path id
	ID string `json:"id"`

	// @oa:in query filter
	Filter string `json:"filter"`

	// @oa:in header X-Auth-Token
	AuthToken string `json:"X-Auth-Token"`

	// TODO cookie
	// SessionID string `json:"session_id"`

	// Тело запроса
	// @oa:description "Request body"
	Data string `json:"data"`
}

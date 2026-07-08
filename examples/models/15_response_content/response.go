package response_content

// UserResponse ответ, который может быть возвращён в JSON или XML
// @oa:description "Standard user response"
// @oa:response "200" "application/json,application/xml"
type UserResponse struct {
	// @oa:description "User ID"
	ID int `json:"id"`

	// @oa:description "User name"
	Name string `json:"name"`

	// @oa:description "User email"
	Email string `json:"email"`
}

// ErrorResponse ответ с ошибкой
// @oa:description "Error response"
// @oa:response "400" "application/json"
type ErrorResponse struct {
	// @oa:description "Error code"
	Code int `json:"code"`

	// @oa:description "Error message"
	Message string `json:"message"`
}

// CreateUserResponse ответ с несколькими status code
// @oa:description "User creation response"
// @oa:response "200" "application/json,application/xml"
// @oa:response "201" "application/json"
type CreateUserResponse struct {
	// @oa:description "Created user ID"
	ID int `json:"id"`

	// @oa:description "User name"
	Name string `json:"name"`

	// @oa:description "Creation timestamp"
	CreatedAt string `json:"created_at"`
}

// NoContentType структура без response аннотации (должна иметь только OaSchema)
// @oa:description "Struct without response annotation"
type NoContentType struct {
	Value string `json:"value"`
}

// NoMediaTypes структура с response, но без media types
// @oa:description "Response without media types"
// @oa:response "204" ""
type NoMediaTypes struct {
}

// MultipleResponses структура с тремя разными response
// @oa:description "Multiple response codes"
// @oa:response "200" "application/json"
// @oa:response "401" "application/json"
// @oa:response "500" "application/json"
type MultipleResponses struct {
	// @oa:description "Data payload"
	Data string `json:"data"`
}

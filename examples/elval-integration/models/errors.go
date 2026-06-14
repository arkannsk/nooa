package models

// Common 4xx error schemas for reuse across examples.
// Register these in Spec.AddError() and reference in routes via PossibleErr().

// ValidationError — 400 Bad Request. Валидация не пройдена.
type ValidationError struct {
	// @oa:description "Short summary of the problem"
	// @oa:example "Validation failed"
	Title string `json:"title"`

	// @oa:description "Detailed explanation"
	// @oa:example "Field 'email' is required"
	Detail string `json:"detail"`

	// @oa:description "HTTP status code"
	// @oa:example 400
	Status int `json:"status"`
}

// UnauthorizedError — 401 Unauthorized.
type UnauthorizedError struct {
	// @oa:description "Short summary"
	// @oa:example "Unauthorized"
	Title string `json:"title"`

	// @oa:description "Why authorization failed"
	// @oa:example "Missing or invalid API key"
	Detail string `json:"detail"`

	// @oa:description "HTTP status code"
	// @oa:example 401
	Status int `json:"status"`
}

// ForbiddenError — 403 Forbidden.
type ForbiddenError struct {
	// @oa:description "Short summary"
	// @oa:example "Forbidden"
	Title string `json:"title"`

	// @oa:description "Why access was denied"
	// @oa:example "You do not have permission to perform this action"
	Detail string `json:"detail"`

	// @oa:description "HTTP status code"
	// @oa:example 403
	Status int `json:"status"`
}

// NotFoundError — 404 Not Found.
type NotFoundError struct {
	// @oa:description "Short summary"
	// @oa:example "Not found"
	Title string `json:"title"`

	// @oa:description "What was not found"
	// @oa:example "Resource with id 'usr_123' does not exist"
	Detail string `json:"detail"`

	// @oa:description "HTTP status code"
	// @oa:example 404
	Status int `json:"status"`
}

// ConflictError — 409 Conflict.
type ConflictError struct {
	// @oa:description "Short summary"
	// @oa:example "Conflict"
	Title string `json:"title"`

	// @oa:description "Description of the conflict"
	// @oa:example "A user with this email already exists"
	Detail string `json:"detail"`

	// @oa:description "HTTP status code"
	// @oa:example 409
	Status int `json:"status"`
}

// RateLimitError — 429 Too Many Requests.
type RateLimitError struct {
	// @oa:description "Short summary"
	// @oa:example "Too many requests"
	Title string `json:"title"`

	// @oa:description "Rate limit details"
	// @oa:example "You have exceeded the rate limit. Try again in 60 seconds."
	Detail string `json:"detail"`

	// @oa:description "HTTP status code"
	// @oa:example 429
	Status int `json:"status"`
}

package models

//go:generate elval-gen gen -input . -openapi

// User представляет пользователя системы.
// @evl:validate required
// @oa:title "User Profile"
// @oa:description "Complete user profile information"
type User struct {
	// @evl:validate required
	// @evl:validate pattern:uuid
	// @oa:description "Unique identifier for the user"
	// @oa:example "550e8400-e29b-41d4-a716-446655440000"
	ID string `json:"id"`

	// @evl:validate required
	// @evl:validate min:3
	// @evl:validate max:50
	// @oa:title "Full Name"
	// @oa:description "The user's full name"
	// @oa:example "John Doe"
	Name string `json:"name"`

	// @evl:validate required
	// @evl:validate pattern:email
	// @oa:format email
	// @oa:example "john.doe@example.com"
	Email string `json:"email"`

	// @evl:validate optional
	// @evl:validate min:18
	// @evl:validate max:120
	// @oa:description "Age in years (optional)"
	Age *int `json:"age,omitempty"`

	// @evl:validate required
	// @evl:validate enum:admin,user,moderator
	// @oa:description "User role in the system"
	Role string `json:"role"`
}

// CreateUserRequest — запрос на создание пользователя.
type CreateUserRequest struct {
	// @evl:validate required
	// @evl:validate min:3
	// @oa:example "Mike Ivanov"
	// @oa:default "Mike Ivanov"
	Name string `json:"name"`

	// @evl:validate required
	// @evl:validate pattern:email
	// @oa:example "m.ivanov@example.com"
	// @oa:default "m.ivanov@example.com"
	Email string `json:"email"`

	// @evl:validate optional
	// @oa:example 25
	Age *int `json:"age,omitempty"`

	// @oa:example "user"
	Role string `json:"role"`
}

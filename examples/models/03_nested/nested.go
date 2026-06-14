package nested_structs

// Address — вложенная структура
// @oa:description "Postal address"
type Address struct {
	// @oa:description "Street address"
	// @oa:example "123 Main St"
	Street string

	// @oa:description "City name"
	// @oa:example "New York"
	City string

	// @oa:description "Postal code"
	// @oa:example "12345"
	ZipCode string

	// @oa:description "Country code (ISO 3166-1 alpha-2)"
	// @oa:example "US"
	Country string
}

// UserWithAddress — структура с вложенным объектом
// @oa:description "User with nested address"
type UserWithAddress struct {
	// @oa:description "User ID"
	// @oa:example "user_001"
	ID string

	// @oa:description "User name"
	// @oa:example "John Doe"
	Name string

	// @oa:description "Billing address"
	// @oa:example {"street": "123 Main St", "city": "New York", "zipCode": "12345", "country": "US"}
	Billing Address

	// @oa:description "Shipping address (optional)"
	// @oa:example {"street": "456 Side St", "city": "Los Angeles", "zipCode": "90001", "country": "US"}
	Shipping *Address
}

// RecursiveNode — рекурсивная структура (проверка на циклические ссылки)
// @oa:description "Tree node with self-reference"
type RecursiveNode struct {
	// @oa:description "Node value"
	// @oa:example "root"
	Value string

	// @oa:description "Child nodes"
	// @oa:example [{"value": "child1"}]
	Children []RecursiveNode

	// @oa:description "Parent reference (nullable)"
	// @oa:example null
	Parent *RecursiveNode
}

// DeepNesting — глубокое вложение (5+ уровней)
// @oa:description "Deeply nested structure"
type DeepNesting struct {
	Level1 struct {
		Level2 struct {
			Level3 struct {
				Level4 struct {
					Level5 struct {
						// @oa:description "Deep value"
						// @oa:example "deep_val"
						Value string
					} `json:"level5"`
				} `json:"level4"`
			} `json:"level3"`
		} `json:"level2"`
	} `json:"level1"`
}

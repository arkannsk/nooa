package nested_structs

// Address — вложенная структура
// @oa:description "Postal address"
type Address struct {
	// @oa:description "Street address"
	// @oa:example "123 Main St"
	Street string

	// @oa:description "City name"
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
	ID string

	// @oa:description "User name"
	Name string

	// @oa:description "Billing address"
	Billing Address

	// @oa:description "Shipping address (optional)"
	Shipping *Address
}

// RecursiveNode — рекурсивная структура (проверка на циклические ссылки)
// @oa:description "Tree node with self-reference"
type RecursiveNode struct {
	// @oa:description "Node value"
	Value string

	// @oa:description "Child nodes"
	Children []RecursiveNode

	// @oa:description "Parent reference (nullable)"
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
						Value string
					} `json:"level5"`
				} `json:"level4"`
			} `json:"level3"`
		} `json:"level2"`
	} `json:"level1"`
}

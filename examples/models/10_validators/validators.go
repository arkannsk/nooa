package validators

// AllStringValidators — все валидаторы для string
// @oa:description "String field with all validators"
type AllStringValidators struct {
	// @evl:validate required
	// @evl:validate min:3
	// @evl:validate max:50
	// @oa:description "Name with length constraints"
	Name string

	// @evl:validate pattern:email
	// @oa:description "Email field"
	Email string

	// @evl:validate pattern:phone
	// @oa:description "Phone field"
	Phone string

	// @evl:validate pattern:uuid
	// @oa:description "UUID field"
	ID string

	// @evl:validate pattern:^https://
	// @oa:description "Custom regex pattern"
	CustomPattern string

	// @evl:validate contains:admin
	// @oa:description "Must contain substring"
	Role string

	// @evl:validate starts_with:https://
	// @oa:description "Must start with prefix"
	URL string

	// @evl:validate ends_with:.com
	// @oa:description "Must end with suffix"
	Domain string
}

// AllNumericValidators — все валидаторы для numeric
// @oa:description "Numeric field with all validators"
type AllNumericValidators struct {
	// @evl:validate min:0
	// @evl:validate max:100
	// @oa:description "Percentage"
	Percent int

	// @evl:validate gt:0
	// @oa:description "Strictly positive"
	Positive int

	// @evl:validate gte:18
	// @oa:description "Age requirement"
	Age int

	// @evl:validate lt:1000
	// @oa:description "Less than threshold"
	Count int

	// @evl:validate lte:999
	// @oa:description "Up to limit"
	Limit int

	// @evl:validate not-zero
	// @oa:description "Non-zero value"
	Amount float64

	// @evl:validate eq:42
	// @oa:description "Must equal specific value"
	Answer int

	// @evl:validate neq:0
	// @oa:description "Must not equal zero"
	Code int
}

// AllEnumAndSliceValidators — enum и slice валидаторы
// @oa:description "Enum and slice validators"
type AllEnumAndSliceValidators struct {
	// @evl:validate enum:active,inactive,pending
	// @oa:description "Status from allowed list"
	Status string

	// @evl:validate required
	// @evl:validate not-zero
	// @evl:validate min:1
	// @evl:validate max:10
	// @oa:description "Tags with constraints"
	Tags []string

	// @evl:validate len:3
	// @oa:description "Exactly 3 items"
	FixedList []string
}

// DateAndDurationValidators — валидаторы для time.Time и time.Duration
// @oa:description "Date and duration validators"
type DateAndDurationValidators struct {
	// @evl:validate
	// @oa:description "Date after threshold"
	StartDate string // или time.Time

	// @evl:validate
	// @oa:description "Date before deadline"
	EndDate string

	// @evl:validate not-zero
	// @oa:description "Non-zero timestamp"
	Timestamp string
}

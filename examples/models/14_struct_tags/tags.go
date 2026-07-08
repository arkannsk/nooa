package structtags

// JsonTagModel — проверка тегов json
// @oa:description "Struct with various json tags"
type JsonTagModel struct {
	// @oa:description "Field with snake_case json tag"
	UserID string `json:"user_id"`

	// @oa:description "Field with camelCase json tag"
	FirstName string `json:"firstName"`

	// @oa:description "Field with json omitempty"
	Email string `json:"email,omitempty"`

	// @oa:description "Field without json tag (fallback to lowercase)"
	NoTag string

	// @oa:description "Ignored field"
	Ignored string `json:"-"`
}

// YamlTagModel — проверка тегов yaml (приоритет json > yaml)
// @oa:description "Struct with yaml tags"
// @oa:response 200 application/json
type YamlTagModel struct {
	// @oa:description "Only yaml tag (no json)"
	UserName string `yaml:"user_name"`

	// @oa:description "Both json and yaml — json wins"
	FullName string `json:"full_name" yaml:"full-name"`
}

// XmlTagModel — проверка тегов xml (приоритет json > yaml > xml)
// @oa:description "Struct with xml tags"
// @oa:response 200 application/json
type XmlTagModel struct {
	// @oa:description "Only xml tag (no json, no yaml)"
	RecordID string `xml:"record_id"`

	// @oa:description "All three tags — json wins"
	RecordTitle string `json:"title" yaml:"record_title" xml:"RecordTitle"`
}

// MixedTagsModel — смешанные теги и без тегов
// @oa:description "Mixed struct with various tag combinations"
type MixedTagsModel struct {
	// @oa:description "json tag"
	FirstName string `json:"first_name"`

	// @oa:description "no tag — fallback"
	LastName string

	// @oa:description "yaml only"
	Age int `yaml:"age"`

	// @oa:description "json with omitempty"
	Address string `json:"address,omitempty"`

	// @oa:description "ignored"
	Secret string `json:"-"`
}

package nooa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oa "github.com/arkannsk/elval/pkg/openapi"
)

// ---------------------------------------------------------------------------
// Test helpers — mock-типы, реализующие schemaProvider
// ---------------------------------------------------------------------------

// mockItem — простая структура с OaSchema
type mockItem struct {
	ID   string
	Name string
}

func (m *mockItem) OaSchema() *oa.Schema {
	return &oa.Schema{
		Type: "object",
		Properties: map[string]*oa.Schema{
			"id":   {Type: "string"},
			"name": {Type: "string"},
		},
	}
}

func (m *mockItem) GlobalRef() string {
	return "#/components/schemas/test.MockItem"
}

// mockContainer — структура, содержащая вложенный mockItem
type mockContainer struct {
	Inner *mockItem
}

func (m *mockContainer) OaSchema() *oa.Schema {
	return &oa.Schema{
		Type: "object",
		Properties: map[string]*oa.Schema{
			"inner": {Ref: "#/components/schemas/test.MockItem"},
		},
	}
}

func (m *mockContainer) GlobalRef() string {
	return "#/components/schemas/test.MockContainer"
}

// ---------------------------------------------------------------------------
// Generic wrapper без OaSchema (симулирует Option[T] из elval-gen)
// ---------------------------------------------------------------------------

type mockOption[T any] struct {
	value    T
	hasValue bool
}

type mockResult[T any, E any] struct {
	value T
	err   E
}

// ---------------------------------------------------------------------------
// Тестовая структура с generic-полями
// ---------------------------------------------------------------------------

type testGenericStruct struct {
	Name  mockOption[string]
	Age   mockOption[int]
	Data  mockResult[string, string]
	Items []mockOption[mockItem]
}

func (t *testGenericStruct) OaSchema() *oa.Schema {
	return &oa.Schema{
		Type: "object",
		Properties: map[string]*oa.Schema{
			"name":  {Type: "string"},
			"age":   {Type: "integer"},
			"data":  {Type: "string"},
			"items": {Type: "array", Items: &oa.Schema{Ref: "#/components/schemas/test.MockItem"}},
		},
	}
}

func (t *testGenericStruct) GlobalRef() string {
	return "#/components/schemas/test.TestGenericStruct"
}

// ---------------------------------------------------------------------------
// Вложенная generic обёртка: Option[Result[Item, Error]]
// ---------------------------------------------------------------------------

type mockError struct {
	Code int
}

type deepNestedStruct struct {
	Value mockOption[mockResult[mockItem, mockError]]
}

func (d *deepNestedStruct) OaSchema() *oa.Schema {
	return &oa.Schema{Type: "object"}
}

func (d *deepNestedStruct) GlobalRef() string {
	return "#/components/schemas/test.DeepNested"
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRegisterModel_Basic(t *testing.T) {
	schemas := make(map[string]*oa.Schema)
	registerModelInternal(schemas, "test.MockItem", new(mockItem))

	s := schemas["test.MockItem"]
	require.NotNil(t, s, "test.MockItem should be registered")
	assert.Equal(t, "object", s.Type)
	assert.Empty(t, s.Ref, "Ref should be cleared on registration")
}

func TestRegisterModel_Duplicate(t *testing.T) {
	schemas := make(map[string]*oa.Schema)
	registerModelInternal(schemas, "test.MockItem", new(mockItem))
	registerModelInternal(schemas, "test.MockItem", new(mockItem))

	assert.Len(t, schemas, 1, "duplicate registration should not add a new schema")
}

func TestRegisterModel_NestedDiscovery(t *testing.T) {
	schemas := make(map[string]*oa.Schema)
	registerModelInternal(schemas, "test.MockContainer", new(mockContainer))

	// Контейнер зарегистрирован
	require.NotNil(t, schemas["test.MockContainer"])
	// mockItem обнаружен автоматически через экспортированное поле Inner
	require.NotNil(t, schemas["test.MockItem"], "nested mockItem should be auto-discovered")
}

func TestRegisterModel_GenericUnwrap(t *testing.T) {
	schemas := make(map[string]*oa.Schema)
	registerModelInternal(schemas, "test.TestGenericStruct", new(testGenericStruct))

	// Саму структуру зарегистрировали
	require.NotNil(t, schemas["test.TestGenericStruct"])
	// mockItem обнаружен через generic обёртку mockOption[mockItem]
	require.NotNil(t, schemas["test.MockItem"], "mockItem should be discovered through generic wrapper")
}

func TestRegisterModel_DeepNestedGenerics(t *testing.T) {
	schemas := make(map[string]*oa.Schema)
	registerModelInternal(schemas, "test.DeepNested", new(deepNestedStruct))

	// DeepNested зарегистрирован
	require.NotNil(t, schemas["test.DeepNested"])
	// mockItem обнаружен через mockOption[mockResult[mockItem, mockError]]
	require.NotNil(t, schemas["test.MockItem"],
		"mockItem should be discovered through nested generic wrappers (Option[Result[Item, Error]])")
}

func TestRegisterModel_NonProviderIgnored(t *testing.T) {
	schemas := make(map[string]*oa.Schema)

	type plainStruct struct{ Field string }
	registerModelInternal(schemas, "test.PlainStruct", new(plainStruct))

	assert.Empty(t, schemas, "type without schemaProvider should not be registered")
}

func TestRegisterModel_PointerUnwrap(t *testing.T) {
	schemas := make(map[string]*oa.Schema)
	registerModelInternal(schemas, "test.MockItem", &mockItem{})

	require.NotNil(t, schemas["test.MockItem"], "pointer should be unwrapped")
}

func TestRegisterSchema_Idempotent(t *testing.T) {
	schemas := make(map[string]*oa.Schema)

	s1 := &oa.Schema{Type: "object", Ref: "#/components/schemas/foo"}
	ok1 := registerSchema(schemas, "foo", s1)
	assert.True(t, ok1, "first registration should succeed")
	assert.Empty(t, s1.Ref, "Ref should be cleared")

	s2 := &oa.Schema{Type: "array"}
	ok2 := registerSchema(schemas, "foo", s2)
	assert.False(t, ok2, "second registration of same name should fail")
	assert.Equal(t, "object", schemas["foo"].Type, "first schema should be preserved")
}

func TestExtractShortName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"#/components/schemas/github.com/arkannsk/nooa/examples/models/03_nested.Address",
			"03_nested.Address",
		},
		{"#/components/schemas/05_generics.Item", "05_generics.Item"},
		{"github.com/foo/bar.Baz", "bar.Baz"},
		{"SimpleName", "SimpleName"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, extractShortName(tc.input))
		})
	}
}

func TestSpec_RegisterModel(t *testing.T) {
	spec := NewSpec(Info{Title: "test", Version: "1.0"})
	spec.RegisterModel("test.MockItem", new(mockItem))

	// Генерируем JSON и проверяем, что схема присутствует
	data := spec.generate()
	assert.Contains(t, string(data), `"test.MockItem"`)
}

func TestSpec_RegisterModelWithGenericDiscovery(t *testing.T) {
	spec := NewSpec(Info{Title: "test", Version: "1.0"})
	spec.RegisterModel("test.TestGenericStruct", new(testGenericStruct))

	data := spec.generate()
	json := string(data)

	assert.Contains(t, json, `"test.TestGenericStruct"`)
	assert.Contains(t, json, `"test.MockItem"`, "MockItem should be auto-discovered through generic wrapper")
}

func TestGlobalRegisterModel(t *testing.T) {
	// Сохраняем и восстанавливаем глобальный реестр
	old := make(map[string]*oa.Schema)
	for k, v := range globalSchemas {
		old[k] = v
	}
	globalSchemas = make(map[string]*oa.Schema)
	defer func() { globalSchemas = old }()

	RegisterModel("test.MockItem", new(mockItem))

	s := globalSchemas["test.MockItem"]
	require.NotNil(t, s)
	assert.Equal(t, "object", s.Type)
}

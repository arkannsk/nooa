package main

import (
	"embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed testdata/*.json
var testFiles embed.FS

func loadSpec(filename string) (map[string]any, error) {
	data, err := testFiles.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func TestParseSpec(t *testing.T) {
	raw, err := loadSpec("testdata/test_spec.json")
	if err != nil {
		t.Fatal(err)
	}

	spec, err := parseSpec(raw)
	if err != nil {
		t.Fatalf("parseSpec: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("title: got %q, want %q", spec.Info.Title, "Test API")
	}
	if spec.Info.Version != "1.0.0" {
		t.Errorf("version: got %q, want %q", spec.Info.Version, "1.0.0")
	}

	// Check paths
	if len(spec.Paths) != 3 {
		t.Errorf("paths: got %d, want 3", len(spec.Paths))
	}

	// Check operations
	usersGet := spec.Paths["/users"].Operations["get"]
	if usersGet.ID != "ListUsers" {
		t.Errorf("ListUsers operationId: got %q, want %q", usersGet.ID, "ListUsers")
	}

	usersPost := spec.Paths["/users"].Operations["post"]
	if usersPost.RequestBody == nil {
		t.Error("CreateUser should have request body")
	}

	// Check schemas
	if len(spec.Schemas) != 2 {
		t.Errorf("schemas: got %d, want 2", len(spec.Schemas))
	}

	// Check total operations
	totalOps := 0
	for _, pi := range spec.Paths {
		totalOps += len(pi.Operations)
	}
	if totalOps != 5 {
		t.Errorf("operations: got %d, want 5", totalOps)
	}
}

func TestGenerate(t *testing.T) {
	raw, err := loadSpec("testdata/test_spec.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		SpecFile:   "test.json",
		Output:     "client.go",
		Package:    "client",
		BaseImport: "github.com/example/models",
	}

	g := NewGenerator(raw, cfg, "Test API", "1.0.0")
	out, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	code := string(out)

	// Check basic structure
	mustContain(t, code, "package client")
	mustContain(t, code, "type Client struct")
	mustContain(t, code, "type HTTPClient interface")
	mustContain(t, code, "func New(baseURL string")

	// Check generated methods
	mustContain(t, code, "func (c *Client) ListUsers(ctx context.Context)")
	mustContain(t, code, "func (c *Client) CreateUser(ctx context.Context")
	mustContain(t, code, "func (c *Client) GetUser(ctx context.Context")
	mustContain(t, code, "func (c *Client) DeleteUser(ctx context.Context")
	mustContain(t, code, "func (c *Client) SearchUsers(ctx context.Context")

	// Check request/response types
	mustContain(t, code, "type CreateUserRequest struct")
	mustContain(t, code, "type CreateUserResponse struct")
	mustContain(t, code, "type SearchUsersRequest struct")

	// Check method signatures
	mustContain(t, code, "ListUsers(ctx context.Context) (*ListUsersResponse, error)")
	mustContain(t, code, "CreateUser(ctx context.Context, input *CreateUserRequest) (*CreateUserResponse, error)")
	mustContain(t, code, "DeleteUser(ctx context.Context, input *DeleteUserRequest) error")

	// Check path param handling
	mustContain(t, code, `strings.ReplaceAll(u, "{id}"`)

	// Check query param handling
	mustContain(t, code, "query := url.Values{}")
	mustContain(t, code, `query.Set("q"`)
	mustContain(t, code, `query.Set("limit"`)

	// Check request body marshaling
	mustContain(t, code, "json.Marshal(input.Body)")
	mustContain(t, code, `Content-Type", "application/json"`)

	// Check response unmarshaling
	mustContain(t, code, "json.Unmarshal(raw")
	mustContain(t, code, "StatusCode")
	mustContain(t, code, "RawBody")

	// Check auth
	mustContain(t, code, "c.TokenPrefix")
	mustContain(t, code, "Authorization")

	// Check no broken returns
	mustNotContain(t, code, "return,")
	mustNotContain(t, code, "returnnil")
}

func TestSanitizeMethodName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"GETUsers", "GETUsers"},
		{"POSTUsers", "POSTUsers"},
		{"DELETEUsers:id", "DELETEUsersid"},
		{"get-users", "Getusers"},
		{"Get_User_By_ID", "GetUserByID"},
		{"My_Operation", "MyOperation"},
		{"", "Operation"},
	}

	for _, tc := range tests {
		got := sanitizeMethodName(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeMethodName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToGoType(t *testing.T) {
	raw, err := loadSpec("testdata/test_spec.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{BaseImport: "github.com/example/models"}
	g := NewGenerator(raw, cfg, "Test", "1.0")

	tests := []struct {
		schema *Schema
		want   string
	}{
		{&Schema{Ref: "#/components/schemas/User"}, "User"},
		{&Schema{Type: "string"}, "string"},
		{&Schema{Type: "integer"}, "int"},
		{&Schema{Type: "integer", Format: "int64"}, "int64"},
		{&Schema{Type: "number"}, "float64"},
		{&Schema{Type: "boolean"}, "bool"},
		{&Schema{Type: "array", Items: &Schema{Ref: "#/components/schemas/User"}}, "[]User"},
		{&Schema{Type: "object"}, "map[string]any"},
		{nil, "any"},
	}

	for _, tc := range tests {
		got := g.schemaToGoType(tc.schema)
		if got != tc.want {
			t.Errorf("toGoType(%v) = %q, want %q", tc.schema, got, tc.want)
		}
	}
}

func TestToImportPath(t *testing.T) {
	raw, err := loadSpec("testdata/test_spec.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{BaseImport: "github.com/example/models"}
	g := NewGenerator(raw, cfg, "Test", "1.0")

	tests := []struct {
		schemaName string
		want       string
	}{
		{"pkg.User", "github.com/example/models/pkg"},
		{"01_basic_types.User", "github.com/example/models/01_basic_types"},
		{"User", ""}, // no package prefix
	}

	for _, tc := range tests {
		got := g.toImportPath(&Schema{Name: tc.schemaName})
		if got != tc.want {
			t.Errorf("toImportPath(%q) = %q, want %q", tc.schemaName, got, tc.want)
		}
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"user_name", "UserName"},
		{"simple", "Simple"},
		{"a_b_c", "ABC"},
		{"", ""},
	}

	for _, tc := range tests {
		got := camelCase(tc.in)
		if got != tc.want {
			t.Errorf("camelCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMultipleResponseCodes(t *testing.T) {
	// Spec with multiple success response codes
	data, _ := testFiles.ReadFile("testdata/multi_response.json")
	var raw map[string]any
	json.Unmarshal(data, &raw)

	cfg := &Config{Package: "client", BaseImport: "github.com/example/models"}
	g := NewGenerator(raw, cfg, "Multi Response API", "1.0.0")
	out, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	code := string(out)

	// Response container should have both Status200 and Status201
	mustContain(t, code, "Status200 Item")
	mustContain(t, code, "Status201 Item")
	mustContain(t, code, "resp.StatusCode == 200")
	mustContain(t, code, "resp.StatusCode == 201")
}

func TestNoRequestBody(t *testing.T) {
	data, _ := testFiles.ReadFile("testdata/simple.json")
	var raw map[string]any
	json.Unmarshal(data, &raw)

	cfg := &Config{Package: "client", BaseImport: "github.com/example/models"}
	g := NewGenerator(raw, cfg, "Simple API", "1.0.0")
	out, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	code := string(out)

	// Should not have request struct
	mustNotContain(t, code, "HealthCheckRequest")
	// Should return error (no response body)
	mustContain(t, code, "func (c *Client) HealthCheck(ctx context.Context) error")
}

func mustContain(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected code to contain %q", substr)
	}
}

func mustNotContain(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected code to NOT contain %q", substr)
	}
}

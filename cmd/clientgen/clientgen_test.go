package main

import (
	"strings"
	"testing"
)

func TestParsePackage_BasicTypes(t *testing.T) {
	info, err := ParsePackage("./../../examples/elval-integration/01_basic_types")
	if err != nil {
		t.Fatalf("ParsePackage failed: %v", err)
	}

	if info.Title == "" {
		t.Error("expected non-empty title")
	}
	if len(info.Routes) == 0 {
		t.Error("expected at least one route")
	}
	if len(info.Imports) == 0 {
		t.Error("expected at least one import")
	}

	// Check that nooa is filtered out
	for alias, path := range info.Imports {
		if path == "github.com/arkannsk/nooa" {
			t.Errorf("nooa should be filtered out, but found: %s -> %s", alias, path)
		}
	}
}

func TestParsePackage_HttpParams(t *testing.T) {
	info, err := ParsePackage("./../../examples/elval-integration/08_http_params")
	if err != nil {
		t.Fatalf("ParsePackage failed: %v", err)
	}

	// Should have path params
	hasPathParams := false
	for _, route := range info.Routes {
		if strings.Contains(route.Path, "{") {
			hasPathParams = true
			break
		}
	}
	if !hasPathParams {
		t.Error("expected at least one route with path parameters")
	}
}

func TestGenerate_BasicTypes(t *testing.T) {
	info, err := ParsePackage("./../../examples/elval-integration/01_basic_types")
	if err != nil {
		t.Fatalf("ParsePackage failed: %v", err)
	}

	cfg := &Config{
		Package: "basictypesdemo",
	}
	gen := NewGenerator(info, cfg)
	resultBytes, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	result := string(resultBytes)

	// Check basic structure
	if !strings.Contains(result, "package basictypesdemo") {
		t.Error("expected package declaration")
	}
	if !strings.Contains(result, "type V1 struct") {
		t.Error("expected V1 struct")
	}
	if !strings.Contains(result, "type HTTPClient interface") {
		t.Error("expected HTTPClient interface")
	}
	if !strings.Contains(result, "func (c *V1) POSTDefaults") {
		t.Error("expected POSTDefaults method")
	}
	if !strings.Contains(result, "func (c *V1) GETPointers") {
		t.Error("expected GETPointers method")
	}
	if !strings.Contains(result, "func (c *V1) POSTPrimitives") {
		t.Error("expected POSTPrimitives method")
	}

	// Check imports
	if !strings.Contains(result, `basictypes "github.com/arkannsk/nooa/examples/models/01_basic_types"`) {
		t.Error("expected basictypes import with alias")
	}
	// Should NOT import nooa
	if strings.Contains(result, `"github.com/arkannsk/nooa"`) {
		t.Error("should not import nooa library")
	}
	// Check new style: imports nooa/client for Codec
	if !strings.Contains(result, `"github.com/arkannsk/nooa/client"`) {
		t.Error("expected nooa/client import")
	}
	// Check new style: status field (lowercase, unexported)
	if !strings.Contains(result, "status200") {
		t.Error("expected unexported status200 field")
	}
	// Check new style: StatusOk method
	if !strings.Contains(result, "StatusOk()") {
		t.Error("expected StatusOk() method")
	}
}

func TestGenerate_HttpParams(t *testing.T) {
	info, err := ParsePackage("./../../examples/elval-integration/08_http_params")
	if err != nil {
		t.Fatalf("ParsePackage failed: %v", err)
	}

	cfg := &Config{
		Package: "httpparams",
	}
	gen := NewGenerator(info, cfg)
	resultBytes, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	result := string(resultBytes)

	// RequestOption and helpers are now shared via github.com/arkannsk/nooa/client
	if !strings.Contains(result, "github.com/arkannsk/nooa/client") {
		t.Error("expected nooa/client import")
	}
	// Methods with model accept the model directly, not a generated *Request
	if !strings.Contains(result, "*httpparams.HeaderParams") {
		t.Error("expected model type in POSTHeaderdemo method")
	}
	if !strings.Contains(result, "*httpparams.QueryParams") {
		t.Error("expected model type in GETSearch method")
	}
}

func TestSanitizeMethodName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GETUsers", "GETUsers"},
		{"POSTItems", "POSTItems"},
		{"GET_Users", "GETUsers"},
		{"GET-Users", "GETUsers"},
		{"GET:Users", "GETUsers"},
		{"GET/Users", "GETUsers"},
		{"X-API-Key", "XAPIKey"},
		{"PUTUsersUserIdResourcesResourceid", "PUTUsersUserIdResourcesResourceid"},
	}

	for _, tt := range tests {
		result := sanitizeMethodName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeMethodName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizePackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"01_basic_types", "basictypes"},
		{"simple", "simple"},
		{"my&package", "mypackage"},
		{"a|b*c", "abc"},
		{"123abc", "abc"},
		{"a?b!c#d", "abcd"},
	}

	for _, tt := range tests {
		result := sanitizePackageName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizePackageName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

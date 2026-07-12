package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNDJSONCodec_Unmarshal(t *testing.T) {
	ndjson := `{"name":"alice","age":30}
{"name":"bob","age":25}
{"name":"charlie","age":35}`

	var result []any
	if err := (&NDJSONCodec{}).Unmarshal([]byte(ndjson), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	alice := result[0].(map[string]any)
	if alice["name"] != "alice" {
		t.Fatalf("expected name=alice, got %v", alice["name"])
	}
	if alice["age"].(float64) != 30 {
		t.Fatalf("expected age=30, got %v", alice["age"])
	}
}

func TestNDJSONCodec_Unmarshal_EmptyLines(t *testing.T) {
	ndjson := `{"a":1}

{"b":2}

`
	var result []any
	if err := (&NDJSONCodec{}).Unmarshal([]byte(ndjson), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items (empty lines skipped), got %d", len(result))
	}
}

func TestNDJSONCodec_Unmarshal_Empty(t *testing.T) {
	var result []any
	if err := (&NDJSONCodec{}).Unmarshal([]byte(""), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 items, got %d", len(result))
	}
}

func TestNDJSONCodec_Unmarshal_InvalidJSON(t *testing.T) {
	ndjson := `{"a":1}
{invalid json}
{"b":2}`

	var result []any
	err := (&NDJSONCodec{}).Unmarshal([]byte(ndjson), &result)
	if err == nil {
		t.Fatal("expected error for invalid JSON line")
	}
}

func TestNDJSONCodec_Unmarshal_WrongTarget(t *testing.T) {
	var result map[string]any
	err := (&NDJSONCodec{}).Unmarshal([]byte(`{"a":1}`), &result)
	if err == nil {
		t.Fatal("expected error: target must be *[]any")
	}
}

func TestNDJSONCodec_Marshal(t *testing.T) {
	data := []any{
		map[string]any{"name": "alice", "age": float64(30)},
		map[string]any{"name": "bob", "age": float64(25)},
	}

	b, err := (&NDJSONCodec{}).Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Unmarshal back
	var result []any
	if err := (&NDJSONCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal roundtrip failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
}

func TestNDJSONCodec_Marshal_WrongType(t *testing.T) {
	_, err := (&NDJSONCodec{}).Marshal("not a slice")
	if err == nil {
		t.Fatal("expected error for non-slice type")
	}
}

func TestUnmarshalBody_NDJSON(t *testing.T) {
	ndjson := `{"id":1,"msg":"hello"}
{"id":2,"msg":"world"}`

	var result []any
	if err := UnmarshalBody([]byte(ndjson), "application/x-ndjson", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	first := result[0].(map[string]any)
	if first["msg"] != "hello" {
		t.Fatalf("expected msg=hello, got %v", first["msg"])
	}
}

func TestUnmarshalBody_NDJSON_AltContentType(t *testing.T) {
	ndjson := `{"x":42}`

	var result []any
	if err := UnmarshalBody([]byte(ndjson), "application/ndjson", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
}

func TestUnmarshalResponse_NDJSON(t *testing.T) {
	ndjson := `{"a":1}
{"a":2}
{"a":3}`

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(ndjson)),
		Header: map[string][]string{
			"Content-Type": {"application/x-ndjson"},
		},
	}

	var result []any
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("UnmarshalResponse failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
}

func TestUnmarshalBody_NDJSON_CustomCodec(t *testing.T) {
	ndjson := `{"v":10}`

	// Custom codec overrides default
	customCodec := &JSONCodec{} // wrong codec to verify it's used
	codecs := map[string]Codec{
		"application/x-ndjson": customCodec,
	}

	var result map[string]any
	if err := UnmarshalBody([]byte(ndjson), "application/x-ndjson", codecs, &result); err != nil {
		t.Fatalf("UnmarshalBody with custom codec failed: %v", err)
	}

	if result["v"].(float64) != 10 {
		t.Fatalf("expected v=10, got %v", result["v"])
	}
}

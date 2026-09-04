package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// --- OctetStreamCodec tests ---

func TestOctetStreamCodec_Unmarshal(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0xff, 0xfe}

	var result []byte
	if err := (&OctetStreamCodec{}).Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != 5 {
		t.Fatalf("expected 5 bytes, got %d", len(result))
	}
	if result[0] != 0x00 || result[3] != 0xff {
		t.Fatalf("byte mismatch: got %v", result)
	}
}

func TestOctetStreamCodec_Unmarshal_WrongTarget(t *testing.T) {
	var result string
	err := (&OctetStreamCodec{}).Unmarshal([]byte("hello"), &result)
	if err == nil {
		t.Fatal("expected error: target must be *[]byte")
	}
}

func TestOctetStreamCodec_Marshal(t *testing.T) {
	data := []byte{1, 2, 3}

	b, err := (&OctetStreamCodec{}).Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if len(b) != 3 || b[0] != 1 {
		t.Fatalf("marshal mismatch: got %v", b)
	}
}

func TestOctetStreamCodec_Marshal_String(t *testing.T) {
	b, err := (&OctetStreamCodec{}).Marshal("hello")
	if err != nil {
		t.Fatalf("Marshal string failed: %v", err)
	}
	if string(b) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(b))
	}
}

func TestOctetStreamCodec_Marshal_WrongType(t *testing.T) {
	_, err := (&OctetStreamCodec{}).Marshal(42)
	if err == nil {
		t.Fatal("expected error for int type")
	}
}

func TestUnmarshalBody_OctetStream(t *testing.T) {
	data := []byte{0x48, 0x45, 0x4c, 0x4c, 0x4f} // "HELLO"

	var result []byte
	if err := UnmarshalBody(data, "application/octet-stream", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody failed: %v", err)
	}

	if string(result) != "HELLO" {
		t.Fatalf("expected HELLO, got %s", string(result))
	}
}

func TestUnmarshalResponse_OctetStream(t *testing.T) {
	data := []byte{10, 20, 30}

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(data))),
		Header: map[string][]string{
			"Content-Type": {"application/octet-stream"},
		},
	}

	var result []byte
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("UnmarshalResponse failed: %v", err)
	}

	if len(result) != 3 || result[0] != 10 {
		t.Fatalf("expected [10,20,30], got %v", result)
	}
}

// --- PlainTextCodec tests ---

func TestPlainTextCodec_Unmarshal(t *testing.T) {
	var result string
	if err := (&PlainTextCodec{}).Unmarshal([]byte("hello world"), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestPlainTextCodec_Unmarshal_WrongTarget(t *testing.T) {
	var result []byte
	err := (&PlainTextCodec{}).Unmarshal([]byte("hello"), &result)
	if err == nil {
		t.Fatal("expected error: target must be *string")
	}
}

func TestPlainTextCodec_Marshal(t *testing.T) {
	b, err := (&PlainTextCodec{}).Marshal("test")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if string(b) != "test" {
		t.Fatalf("expected 'test', got %q", string(b))
	}
}

func TestPlainTextCodec_Marshal_Bytes(t *testing.T) {
	b, err := (&PlainTextCodec{}).Marshal([]byte("bytes"))
	if err != nil {
		t.Fatalf("Marshal bytes failed: %v", err)
	}
	if string(b) != "bytes" {
		t.Fatalf("expected 'bytes', got %q", string(b))
	}
}

func TestPlainTextCodec_Marshal_WrongType(t *testing.T) {
	_, err := (&PlainTextCodec{}).Marshal(42)
	if err == nil {
		t.Fatal("expected error for int type")
	}
}

func TestUnmarshalBody_PlainText(t *testing.T) {
	var result string
	if err := UnmarshalBody([]byte("plain text content"), "text/plain", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody failed: %v", err)
	}
	if result != "plain text content" {
		t.Fatalf("expected 'plain text content', got %q", result)
	}
}

func TestUnmarshalResponse_PlainText(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("line1\nline2")),
		Header: map[string][]string{
			"Content-Type": {"text/plain"},
		},
	}

	var result string
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("UnmarshalResponse failed: %v", err)
	}
	if result != "line1\nline2" {
		t.Fatalf("expected multiline, got %q", result)
	}
}

// --- Fallback tests ---

func TestUnmarshalBody_Fallback_JSON(t *testing.T) {
	// Unknown content type falls back to JSON
	var result map[string]any
	if err := UnmarshalBody([]byte(`{"x":1}`), "application/unknown", nil, &result); err != nil {
		t.Fatalf("fallback JSON failed: %v", err)
	}
	if result["x"].(float64) != 1 {
		t.Fatalf("expected x=1, got %v", result["x"])
	}
}

// --- Roundtrip tests ---

func TestOctetStream_Roundtrip(t *testing.T) {
	original := []byte{0, 1, 255, 128, 64}

	b, err := (&OctetStreamCodec{}).Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result []byte
	if err := (&OctetStreamCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result) != len(original) {
		t.Fatalf("length mismatch: %d vs %d", len(result), len(original))
	}
	for i := range original {
		if result[i] != original[i] {
			t.Fatalf("byte mismatch at %d: %d vs %d", i, result[i], original[i])
		}
	}
}

func TestPlainText_Roundtrip(t *testing.T) {
	original := "hello\nworld\ttab"

	b, err := (&PlainTextCodec{}).Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result string
	if err := (&PlainTextCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result != original {
		t.Fatalf("roundtrip mismatch: %q vs %q", result, original)
	}
}

package client

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

// --- MultipartCodec tests ---

func TestMultipartCodec_Unmarshal(t *testing.T) {
	// Build a multipart body
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	_ = w.WriteField("name", "alice")
	_ = w.WriteField("age", "30")

	fw, _ := w.CreateFormFile("file", "file.txt")
	_, _ = fw.Write([]byte("file content here"))

	_ = w.Close()

	body := buf.Bytes()

	var result MultipartFormData
	if err := (&MultipartCodec{}).Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Fields["name"] != "alice" {
		t.Fatalf("expected name=alice, got %q", result.Fields["name"])
	}
	if result.Fields["age"] != "30" {
		t.Fatalf("expected age=30, got %q", result.Fields["age"])
	}
	if string(result.Files["file"]) != "file content here" {
		t.Fatalf("expected 'file content here', got %q", string(result.Files["file"]))
	}
}

func TestMultipartCodec_Unmarshal_FieldsOnly(t *testing.T) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	_ = w.WriteField("key1", "val1")
	_ = w.WriteField("key2", "val2")
	_ = w.Close()

	var result MultipartFormData
	if err := (&MultipartCodec{}).Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Fields["key1"] != "val1" || result.Fields["key2"] != "val2" {
		t.Fatalf("unexpected fields: %v", result.Fields)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected no files, got %d", len(result.Files))
	}
}

func TestMultipartCodec_Unmarshal_WrongTarget(t *testing.T) {
	var result string
	err := (&MultipartCodec{}).Unmarshal([]byte("boundary"), &result)
	if err == nil {
		t.Fatal("expected error: target must be *MultipartFormData")
	}
}

func TestMultipartCodec_Marshal(t *testing.T) {
	data := &MultipartFormData{
		Fields: map[string]string{
			"name": "bob",
			"age":  "25",
		},
		Files: map[string][]byte{
			"doc": []byte("document content"),
		},
		FileNames: map[string]string{
			"doc": "doc.txt",
		},
	}

	b, err := (&MultipartCodec{}).Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result MultipartFormData
	if err := (&MultipartCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal roundtrip failed: %v", err)
	}

	if result.Fields["name"] != "bob" {
		t.Fatalf("expected name=bob, got %q", result.Fields["name"])
	}
	if string(result.Files["doc"]) != "document content" {
		t.Fatalf("expected 'document content', got %q", string(result.Files["doc"]))
	}
}

func TestMultipartCodec_Marshal_WrongType(t *testing.T) {
	_, err := (&MultipartCodec{}).Marshal("not multipart")
	if err == nil {
		t.Fatal("expected error for non-MultipartFormData type")
	}
}

func TestUnmarshalBody_Multipart(t *testing.T) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	_ = w.WriteField("status", "ok")
	_ = w.Close()

	var result MultipartFormData
	if err := UnmarshalBody(buf.Bytes(), "multipart/form-data", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody failed: %v", err)
	}

	if result.Fields["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", result.Fields["status"])
	}
}

func TestUnmarshalResponse_Multipart(t *testing.T) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	_ = w.WriteField("token", "abc123")

	fw, _ := w.CreateFormFile("data", "data.bin")
	_, _ = fw.Write([]byte("binary payload"))

	_ = w.Close()

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
		Header: map[string][]string{
			"Content-Type": {"multipart/form-data"},
		},
	}

	var result MultipartFormData
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("UnmarshalResponse failed: %v", err)
	}

	if result.Fields["token"] != "abc123" {
		t.Fatalf("expected token=abc123, got %q", result.Fields["token"])
	}
	if string(result.Files["data"]) != "binary payload" {
		t.Fatalf("expected 'binary payload', got %q", string(result.Files["data"]))
	}
}

// --- Roundtrip ---

func TestMultipart_Roundtrip(t *testing.T) {
	original := &MultipartFormData{
		Fields: map[string]string{
			"action":  "upload",
			"version": "1.0",
		},
		Files: map[string][]byte{
			"avatar": []byte{0x89, 0x50, 0x4e, 0x47}, // fake PNG header
		},
		FileNames: map[string]string{
			"avatar": "avatar.png",
		},
	}

	b, err := (&MultipartCodec{}).Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result MultipartFormData
	if err := (&MultipartCodec{}).Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Fields["action"] != "upload" {
		t.Fatalf("expected action=upload, got %q", result.Fields["action"])
	}
	if string(result.Files["avatar"]) != string(original.Files["avatar"]) {
		t.Fatalf("file content mismatch")
	}
}

// --- Edge cases ---

func TestMultipartCodec_Unmarshal_MultipleFilesSameField(t *testing.T) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	// First file
	fw1, _ := w.CreateFormFile("upload", "first.txt")
	_, _ = fw1.Write([]byte("first"))

	// Second file overwrites first in the map
	fw2, _ := w.CreateFormFile("upload", "second.txt")
	_, _ = fw2.Write([]byte("second"))

	_ = w.Close()

	var result MultipartFormData
	if err := (&MultipartCodec{}).Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Last file wins (map behavior)
	if string(result.Files["upload"]) != "second" {
		t.Fatalf("expected 'second', got %q", string(result.Files["upload"]))
	}
}

func TestUnmarshalBody_CSV_AltContentType(t *testing.T) {
	// application/csv is also valid per RFC 7111
	csvData := `a,b
1,2`

	var result [][]string
	if err := UnmarshalBody([]byte(csvData), "application/csv", nil, &result); err != nil {
		t.Fatalf("UnmarshalBody with application/csv failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
}

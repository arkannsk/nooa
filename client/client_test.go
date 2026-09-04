package client

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"testing"
)

// --- StreamReader tests ---

func TestStreamReader_DefaultBufferSize(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("data"))
	s := NewStreamReader(rc)
	if s.bufSize != 32*1024 {
		t.Fatalf("expected default bufSize 32KB, got %d", s.bufSize)
	}
}

func TestStreamReader_Read(t *testing.T) {
	data := "test data for streaming"
	rc := io.NopCloser(strings.NewReader(data))
	s := NewStreamReader(rc)

	result := &bytes.Buffer{}
	for {
		buf := make([]byte, 5)
		n, err := s.Read(buf)
		result.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if result.String() != data {
		t.Fatalf("expected %q, got %q", data, result.String())
	}
}

func TestStreamReader_Progress(t *testing.T) {
	data := strings.Repeat("x", 100)
	var progressCalls []int64
	rc := io.NopCloser(strings.NewReader(data))
	s := NewStreamReader(rc,
		WithBufferSize(10),
		WithTotal(100),
		WithProgress(func(n int64) {
			progressCalls = append(progressCalls, n)
		}),
	)

	buf := make([]byte, 10)
	for {
		_, err := s.Read(buf)
		if err == io.EOF {
			break
		}
	}

	if len(progressCalls) == 0 {
		t.Fatal("progress callback was never called")
	}
	if progressCalls[len(progressCalls)-1] != 100 {
		t.Fatalf("expected final progress 100, got %d", progressCalls[len(progressCalls)-1])
	}

	p := s.Progress()
	if p != 1.0 {
		t.Fatalf("expected progress 1.0, got %f", p)
	}
}

func TestStreamReader_Progress_UnknownTotal(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("data"))
	s := NewStreamReader(rc)

	if s.Progress() != -1 {
		t.Fatalf("expected -1 for unknown total, got %f", s.Progress())
	}
}

func TestStreamReader_PipeTo(t *testing.T) {
	data := strings.Repeat("pipe test data\n", 500) // ~9 KB
	rc := io.NopCloser(strings.NewReader(data))
	s := NewStreamReader(rc, WithBufferSize(4096))

	dst := &bytes.Buffer{}
	n, err := s.PipeTo(dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if dst.String() != data {
		t.Fatalf("data mismatch")
	}
}

func TestStreamReader_PipeTo_Progress(t *testing.T) {
	data := strings.Repeat("x", 200)
	var progressCalls []int64
	rc := io.NopCloser(strings.NewReader(data))
	s := NewStreamReader(rc,
		WithBufferSize(20),
		WithTotal(200),
		WithProgress(func(n int64) {
			progressCalls = append(progressCalls, n)
		}),
	)

	dst := &bytes.Buffer{}
	n, err := s.PipeTo(dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if progressCalls[len(progressCalls)-1] != 200 {
		t.Fatalf("expected final progress 200, got %d", progressCalls[len(progressCalls)-1])
	}
}

func TestStreamReader_BytesRead(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("1234567890"))
	s := NewStreamReader(rc, WithBufferSize(3))

	buf := make([]byte, 3)
	s.Read(buf) // 3
	s.Read(buf) // 3
	s.Read(buf) // 3

	if s.BytesRead() != 9 {
		t.Fatalf("expected 9 bytes read, got %d", s.BytesRead())
	}
}

func TestStreamReader_Close(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("data"))
	s := NewStreamReader(rc)

	err := s.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStreamReader_WithBufferSize(t *testing.T) {
	rc := io.NopCloser(strings.NewReader("data"))
	s := NewStreamReader(rc, WithBufferSize(64*1024))

	if s.bufSize != 64*1024 {
		t.Fatalf("expected bufSize 64KB, got %d", s.bufSize)
	}
}

func TestStreamReader_MultipleReadsAfterPipeTo(t *testing.T) {
	data := "hello"
	rc := io.NopCloser(strings.NewReader(data))
	s := NewStreamReader(rc)

	dst := &bytes.Buffer{}
	n, err := s.PipeTo(dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}

	// Further read should be EOF
	buf := make([]byte, 10)
	_, err = s.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

// --- UnmarshalResponse / UnmarshalBody tests ---

func TestUnmarshalResponse_JSON(t *testing.T) {
	body := `{"name":"test","value":42}`
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: map[string][]string{"Content-Type": {"application/json"}},
	}

	var result map[string]interface{}
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["name"] != "test" {
		t.Fatalf("expected name=test, got %v", result["name"])
	}
	if result["value"].(float64) != 42 {
		t.Fatalf("expected value=42, got %v", result["value"])
	}
}

func TestUnmarshalResponse_XML(t *testing.T) {
	type Item struct {
		XMLName xml.Name `xml:"item"`
		Name    string   `xml:"name"`
	}

	body := `<item><name>xml-test</name></item>`
	resp := &http.Response{
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: map[string][]string{"Content-Type": {"application/xml"}},
	}

	var result Item
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "xml-test" {
		t.Fatalf("expected name=xml-test, got %v", result.Name)
	}
}

func TestUnmarshalBody(t *testing.T) {
	body := []byte(`{"key":"val"}`)

	var result map[string]interface{}
	if err := UnmarshalBody(body, "application/json", nil, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "val" {
		t.Fatalf("expected key=val, got %v", result["key"])
	}
}

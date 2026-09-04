package client

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// generatedResponse mimics a generated response struct to verify
// that Stream(), Body(), readBody() work correctly together.
type generatedResponse struct {
	resp     *http.Response
	codec    map[string]Codec
	bodyOnce sync.Once
}

func (r *generatedResponse) readBody() error {
	var err error
	r.bodyOnce.Do(func() {
		data, readErr := io.ReadAll(r.resp.Body)
		if readErr != nil {
			err = readErr
			return
		}
		r.resp.Body = NewCachedBodyReader(data)
	})
	return err
}

func (r *generatedResponse) Body() io.ReadCloser {
	if r.resp == nil {
		return nil
	}
	if seeker, ok := r.resp.Body.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}
	return r.resp.Body
}

func (r *generatedResponse) Stream(opts ...StreamOption) *StreamReader {
	if r.resp == nil {
		return nil
	}
	opts = append(opts, WithTotal(r.resp.ContentLength))
	return NewStreamReader(r.resp.Body, opts...)
}

// TestStreamDirect — Stream() reads directly from resp.Body without caching.
func TestStreamDirect(t *testing.T) {
	data := strings.Repeat("streaming data\n", 100)

	resp := &http.Response{
		StatusCode:    200,
		ContentLength: int64(len(data)),
		Body:          io.NopCloser(strings.NewReader(data)),
		Header: map[string][]string{
			"Content-Type": {"application/octet-stream"},
		},
	}

	gr := &generatedResponse{resp: resp}

	var progressCalls []int64
	stream := gr.Stream(
		WithBufferSize(256),
		WithProgress(func(n int64) {
			progressCalls = append(progressCalls, n)
		}),
	)
	defer stream.Close()

	dst := &bytes.Buffer{}
	n, err := stream.PipeTo(dst)
	if err != nil {
		t.Fatalf("PipeTo failed: %v", err)
	}
	if n != int64(len(data)) {
		t.Fatalf("expected %d bytes, got %d", len(data), n)
	}
	if dst.String() != data {
		t.Fatalf("data mismatch")
	}

	// After Stream(), resp.Body is exhausted and not cached.
	bodyData, _ := io.ReadAll(gr.Body())
	if len(bodyData) != 0 {
		t.Fatalf("expected empty body after Stream, got %d bytes", len(bodyData))
	}

	if len(progressCalls) == 0 {
		t.Fatal("progress callback was never called")
	}
}

// TestReadBodyThenBody — readBody caches, then Body() returns reset reader each time.
func TestReadBodyThenBody(t *testing.T) {
	data := "cached body data"

	resp := &http.Response{
		StatusCode:    200,
		ContentLength: int64(len(data)),
		Body:          io.NopCloser(strings.NewReader(data)),
		Header: map[string][]string{
			"Content-Type": {"text/plain"},
		},
	}

	gr := &generatedResponse{resp: resp}

	// readBody caches the body
	if err := gr.readBody(); err != nil {
		t.Fatalf("readBody failed: %v", err)
	}

	// Body() returns cached reader, Seek(0) resets position
	body1 := gr.Body()
	bodyData1, err := io.ReadAll(body1)
	if err != nil {
		t.Fatalf("Body() failed: %v", err)
	}
	if string(bodyData1) != data {
		t.Fatalf("Body() mismatch: got %q", string(bodyData1))
	}

	// Body() again — Seek(0) resets, fresh data
	body2 := gr.Body()
	bodyData2, err := io.ReadAll(body2)
	if err != nil {
		t.Fatalf("Body() second call failed: %v", err)
	}
	if string(bodyData2) != data {
		t.Fatalf("Body() second call mismatch: got %q", string(bodyData2))
	}
}

// TestUnmarshalThenBody — UnmarshalResponse reads from cached resp.Body,
// then Body() seeks back and returns full data.
func TestUnmarshalThenBody(t *testing.T) {
	jsonData := `{"name":"test","value":42}`

	resp := &http.Response{
		StatusCode:    200,
		ContentLength: int64(len(jsonData)),
		Body:          io.NopCloser(strings.NewReader(jsonData)),
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
	}

	gr := &generatedResponse{resp: resp}

	// readBody (simulates StatusOk)
	if err := gr.readBody(); err != nil {
		t.Fatalf("readBody failed: %v", err)
	}

	// UnmarshalResponse reads from cached resp.Body
	var result map[string]interface{}
	if err := UnmarshalResponse(resp, nil, &result); err != nil {
		t.Fatalf("UnmarshalResponse failed: %v", err)
	}
	if result["name"] != "test" {
		t.Fatalf("expected name=test, got %v", result["name"])
	}

	// Body() seeks back to 0 and returns full data
	body := gr.Body()
	bodyData, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Body() failed: %v", err)
	}
	if string(bodyData) != jsonData {
		t.Fatalf("Body() mismatch: got %q", string(bodyData))
	}
}

// TestBodyThenStream — Body() first (original), then Stream() (body exhausted).
func TestBodyThenStream(t *testing.T) {
	data := "body first"

	resp := &http.Response{
		StatusCode:    200,
		ContentLength: int64(len(data)),
		Body:          io.NopCloser(strings.NewReader(data)),
		Header: map[string][]string{
			"Content-Type": {"text/plain"},
		},
	}

	gr := &generatedResponse{resp: resp}

	// Body() returns original resp.Body
	body := gr.Body()
	bodyData, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Body() failed: %v", err)
	}
	if string(bodyData) != data {
		t.Fatalf("Body() mismatch: got %q", string(bodyData))
	}

	// Stream() now reads from exhausted resp.Body
	stream := gr.Stream()
	defer stream.Close()
	dst := &bytes.Buffer{}
	n, _ := stream.PipeTo(dst)
	if n != 0 {
		t.Fatalf("expected 0 bytes from Stream after Body consumed, got %d", n)
	}
}

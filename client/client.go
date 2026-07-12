// Package client provides shared utilities for generated HTTP clients.
//
// It defines the Codec interface for response body decoding and ready-made
// codecs for JSON, XML, and YAML. Generated clients import this package to
// avoid duplicating unmarshaling logic across generated files.
package client

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// Codec defines the interface for marshaling/unmarshaling response bodies.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(b []byte, v any) error
}

// --- JSON ---

// JSONCodec is the default Codec for application/json.
type JSONCodec struct{}

func (j *JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JSONCodec) Unmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}

// --- XML ---

// XMLCodec is a Codec for application/xml and text/xml.
type XMLCodec struct{}

func (x *XMLCodec) Marshal(v any) ([]byte, error) {
	return xml.Marshal(v)
}

func (x *XMLCodec) Unmarshal(b []byte, v any) error {
	return xml.Unmarshal(b, v)
}

// --- YAML ---

// YAMLCodec is a Codec for application/x-yaml and text/yaml.
type YAMLCodec struct{}

func (y *YAMLCodec) Marshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func (y *YAMLCodec) Unmarshal(b []byte, v any) error {
	return yaml.Unmarshal(b, v)
}

// --- Default codecs registry ---

// defaultCodecs holds the built-in codecs indexed by media type.
var defaultCodecs = map[string]Codec{
	"application/json": &JSONCodec{},
	"application/xml":  &XMLCodec{},
	"text/xml":         &XMLCodec{},
	"application/x-yaml": &YAMLCodec{},
	"text/yaml":        &YAMLCodec{},
}

// UnmarshalResponse reads the response body and decodes it using the
// appropriate Codec based on the Content-Type header.
//
// If codecs is nil or does not contain an entry for the detected media type,
// the built-in registry (JSON, XML, YAML) is consulted. If no codec matches,
// JSON decoding is used as a final fallback.
func UnmarshalResponse(resp *http.Response, codecs map[string]Codec, v any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	return UnmarshalBody(body, resp.Header.Get("Content-Type"), codecs, v)
}

// UnmarshalBody decodes a pre-read body into v using the appropriate Codec
// based on the Content-Type header value.
//
// If codecs is nil or does not contain an entry for the detected media type,
// the built-in registry (JSON, XML, YAML) is consulted. If no codec matches,
// JSON decoding is used as a final fallback.
func UnmarshalBody(body []byte, contentType string, codecs map[string]Codec, v any) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "application/json"
	}

	var codec Codec
	if codecs != nil {
		if co, ok := codecs[mediaType]; ok {
			codec = co
		}
	}
	if codec == nil {
		codec = defaultCodecs[mediaType]
	}
	if codec == nil {
		codec = &JSONCodec{}
	}

	if err := codec.Unmarshal(body, v); err != nil {
		return fmt.Errorf("unmarshal response (%s): %w", mediaType, err)
	}
	return nil
}

// --- Request option pattern ---

// RequestOption modifies an HTTP request before it is sent.
// Use WithBody, WithQuery, WithHeader, WithPath to configure a request.
type RequestOption func(*http.Request)

// WithBody sets the request body (JSON-encoded).
func WithBody(v any) RequestOption {
	return func(r *http.Request) {
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	}
}

// WithQuery adds a query parameter.
func WithQuery(key string, value any) RequestOption {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set(key, fmt.Sprintf("%v", value))
		r.URL.RawQuery = q.Encode()
	}
}

// WithHeader sets a request header.
func WithHeader(key, value string) RequestOption {
	return func(r *http.Request) {
		r.Header.Set(key, value)
	}
}

// WithPath substitutes a path placeholder {key} with the given value.
func WithPath(key string, value any) RequestOption {
	return func(r *http.Request) {
		r.URL.Path = strings.ReplaceAll(r.URL.Path, "{"+key+"}", fmt.Sprintf("%v", value))
	}
}

// --- Streaming ---

// StreamReader wraps an io.ReadCloser with buffered reading and optional
// progress callbacks. It is useful for streaming large responses (e.g. files)
// to another writer such as a file or Minio upload.
type StreamReader struct {
	reader     io.Reader
	closer     io.Closer
	bufSize    int
	progressFn func(bytesRead int64)
	total      int64
	read       int64
}

// StreamOption configures a StreamReader.
type StreamOption func(*StreamReader)

// WithBufferSize sets the buffer size for streaming reads.
// Default is 32 KB.
func WithBufferSize(size int) StreamOption {
	return func(s *StreamReader) {
		s.bufSize = size
	}
}

// WithProgress sets a callback invoked after each buffer read.
// The callback receives the cumulative number of bytes read so far.
func WithProgress(fn func(bytesRead int64)) StreamOption {
	return func(s *StreamReader) {
		s.progressFn = fn
	}
}

// WithTotal sets the expected total size (from Content-Length).
// Used for progress reporting.
func WithTotal(total int64) StreamOption {
	return func(s *StreamReader) {
		s.total = total
	}
}

// NewStreamReader creates a new StreamReader from an io.ReadCloser.
// The default buffer size is 32 KB.
func NewStreamReader(rc io.ReadCloser, opts ...StreamOption) *StreamReader {
	s := &StreamReader{
		reader:  rc,
		closer:  rc,
		bufSize: 32 * 1024, // 32 KB default
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Read implements io.Reader. It reads into the provided buffer,
// updating the internal byte counter and invoking the progress callback.
func (s *StreamReader) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	if n > 0 {
		s.read += int64(n)
		if s.progressFn != nil {
			s.progressFn(s.read)
		}
	}
	return n, err
}

// Close implements io.Closer.
func (s *StreamReader) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// PipeTo copies the entire stream to the given writer using the configured
// buffer size. It returns the total number of bytes copied.
// After calling PipeTo, the stream is exhausted and should be closed.
func (s *StreamReader) PipeTo(w io.Writer) (int64, error) {
	buf := make([]byte, s.bufSize)
	var total int64
	for {
		n, err := s.reader.Read(buf)
		if n > 0 {
			s.read += int64(n)
			total += int64(n)
			if s.progressFn != nil {
				s.progressFn(s.read)
			}
			_, wErr := w.Write(buf[:n])
			if wErr != nil {
				return total, wErr
			}
		}
		if err != nil {
			return total, err
		}
	}
}

// BytesRead returns the cumulative number of bytes read so far.
func (s *StreamReader) BytesRead() int64 {
	return s.read
}

// Progress returns the fraction of data read so far (0.0–1.0).
// If total size is unknown (not set via WithTotal), returns -1.
func (s *StreamReader) Progress() float64 {
	if s.total <= 0 {
		return -1
	}
	return float64(s.read) / float64(s.total)
}

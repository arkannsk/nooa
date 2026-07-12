// Package client provides shared utilities for generated HTTP clients.
//
// It defines the Codec interface for response body decoding and ready-made
// codecs for JSON, XML, and YAML. Generated clients import this package to
// avoid duplicating unmarshaling logic across generated files.
package client

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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

// --- NDJSON (Newline Delimited JSON) ---

// NDJSONCodec decodes newline-delimited JSON into a slice.
// Each line is parsed as a separate JSON object and appended to the slice.
// The target must be a pointer to a slice (e.g. *[]MyType or *[]map[string]any).
type NDJSONCodec struct{}

func (n *NDJSONCodec) Marshal(v any) ([]byte, error) {
	switch data := v.(type) {
	case []any:
		var lines []string
		for _, item := range data {
			b, err := json.Marshal(item)
			if err != nil {
				return nil, fmt.Errorf("marshal ndjson item: %w", err)
			}
			lines = append(lines, string(b))
		}
		return []byte(strings.Join(lines, "\n")), nil
	default:
		return nil, fmt.Errorf("ndjson marshal: unsupported type %T", v)
	}
}

func (n *NDJSONCodec) Unmarshal(b []byte, v any) error {
	slicePtr, ok := v.(*[]any)
	if !ok {
		return fmt.Errorf("ndjson unmarshal: target must be *[]any, got %T", v)
	}

	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	var items []any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item any
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return fmt.Errorf("ndjson unmarshal line: %w", err)
		}
		items = append(items, item)
	}
	*slicePtr = items
	return nil
}

// --- Octet Stream (raw bytes) ---

// OctetStreamCodec decodes raw binary data into a []byte.
// The target must be a pointer to a byte slice (e.g. *[]byte).
type OctetStreamCodec struct{}

func (o *OctetStreamCodec) Marshal(v any) ([]byte, error) {
	switch data := v.(type) {
	case []byte:
		return data, nil
	case string:
		return []byte(data), nil
	default:
		return nil, fmt.Errorf("octet-stream marshal: unsupported type %T", v)
	}
}

func (o *OctetStreamCodec) Unmarshal(b []byte, v any) error {
	slicePtr, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("octet-stream unmarshal: target must be *[]byte, got %T", v)
	}
	*slicePtr = append([]byte(nil), b...)
	return nil
}

// --- Plain Text ---

// PlainTextCodec decodes plain text into a string.
// The target must be a pointer to a string (e.g. *string).
type PlainTextCodec struct{}

func (p *PlainTextCodec) Marshal(v any) ([]byte, error) {
	switch data := v.(type) {
	case string:
		return []byte(data), nil
	case []byte:
		return data, nil
	default:
		return nil, fmt.Errorf("text/plain marshal: unsupported type %T", v)
	}
}

func (p *PlainTextCodec) Unmarshal(b []byte, v any) error {
	strPtr, ok := v.(*string)
	if !ok {
		return fmt.Errorf("text/plain unmarshal: target must be *string, got %T", v)
	}
	*strPtr = string(b)
	return nil
}

// --- CSV ---

// CSVCodec decodes CSV data into a [][]string.
// The target must be a pointer to a string slice slice (e.g. *[][]string).
type CSVCodec struct{}

func (c *CSVCodec) Marshal(v any) ([]byte, error) {
	data, ok := v.([][]string)
	if !ok {
		return nil, fmt.Errorf("csv marshal: target must be [][]string, got %T", v)
	}

	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	if err := w.WriteAll(data); err != nil {
		return nil, fmt.Errorf("csv marshal: %w", err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv marshal flush: %w", err)
	}
	return buf.Bytes(), nil
}

func (c *CSVCodec) Unmarshal(b []byte, v any) error {
	slicePtr, ok := v.(*[][]string)
	if !ok {
		return fmt.Errorf("csv unmarshal: target must be *[][]string, got %T", v)
	}

	r := csv.NewReader(bytes.NewReader(b))
	records, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("csv unmarshal: %w", err)
	}
	*slicePtr = records
	return nil
}

// extractMultipartBoundary extracts the boundary string from raw multipart body bytes.
// The first line of a multipart body is "--<boundary>".
func extractMultipartBoundary(b []byte) (string, error) {
	lineEnd := bytes.IndexByte(b, '\n')
	if lineEnd <= 0 {
		return "", fmt.Errorf("multipart: cannot find boundary in body")
	}
	firstLine := string(b[:lineEnd])
	// Strip optional \r
	firstLine = strings.TrimRight(firstLine, "\r")
	// Remove leading "--" and trailing "--" (if closing delimiter)
	if !strings.HasPrefix(firstLine, "--") {
		return "", fmt.Errorf("multipart: invalid body, no boundary prefix")
	}
	boundary := strings.TrimPrefix(firstLine, "--")
	boundary = strings.TrimSuffix(boundary, "--")
	if boundary == "" {
		return "", fmt.Errorf("multipart: empty boundary")
	}
	return boundary, nil
}

// --- Multipart ---

// MultipartFormData represents a parsed multipart/form-data response.
type MultipartFormData struct {
	// Fields maps field names to their text values.
	Fields map[string]string
	// Files maps field names to their binary data.
	Files map[string][]byte
	// FileNames maps field names to original filenames (if available).
	FileNames map[string]string
}

// MultipartCodec decodes multipart/form-data responses.
// The target must be *MultipartFormData.
type MultipartCodec struct{}

func (m *MultipartCodec) Marshal(v any) ([]byte, error) {
	data, ok := v.(*MultipartFormData)
	if !ok {
		return nil, fmt.Errorf("multipart marshal: target must be *MultipartFormData, got %T", v)
	}

	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for name, value := range data.Fields {
		if err := w.WriteField(name, value); err != nil {
			return nil, fmt.Errorf("multipart marshal field %q: %w", name, err)
		}
	}
	for name, value := range data.Files {
		filename := data.FileNames[name]
		if filename == "" {
			filename = name
		}
		fw, err := w.CreateFormFile(name, filename)
		if err != nil {
			return nil, fmt.Errorf("multipart marshal file %q: %w", name, err)
		}
		_, err = fw.Write(value)
		if err != nil {
			return nil, fmt.Errorf("multipart marshal file %q write: %w", name, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("multipart marshal close: %w", err)
	}
	return buf.Bytes(), nil
}

func (m *MultipartCodec) Unmarshal(b []byte, v any) error {
	target, ok := v.(*MultipartFormData)
	if !ok {
		return fmt.Errorf("multipart unmarshal: target must be *MultipartFormData, got %T", v)
	}

	boundary, err := extractMultipartBoundary(b)
	if err != nil {
		return fmt.Errorf("multipart unmarshal: %w", err)
	}

	reader := multipart.NewReader(bytes.NewReader(b), boundary)
	forms, err := reader.ReadForm(32 << 20) // 32 MB max memory
	if err != nil {
		return fmt.Errorf("multipart unmarshal: %w", err)
	}

	target.Fields = make(map[string]string)
	for k, vals := range forms.Value {
		target.Fields[k] = vals[0]
	}

	target.Files = make(map[string][]byte)
	target.FileNames = make(map[string]string)
	for k, files := range forms.File {
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				return fmt.Errorf("multipart open file %q: %w", k, err)
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				return fmt.Errorf("multipart read file %q: %w", k, err)
			}
			target.Files[k] = data
			target.FileNames[k] = fh.Filename
		}
	}

	return nil
}

// --- Default codecs registry ---

// defaultCodecs holds the built-in codecs indexed by media type.
var defaultCodecs = map[string]Codec{
	"application/json":           &JSONCodec{},
	"application/xml":            &XMLCodec{},
	"text/xml":                   &XMLCodec{},
	"application/x-yaml":         &YAMLCodec{},
	"text/yaml":                  &YAMLCodec{},
	"application/x-ndjson":       &NDJSONCodec{},
	"application/ndjson":         &NDJSONCodec{},
	"application/octet-stream":   &OctetStreamCodec{},
	"text/plain":                 &PlainTextCodec{},
	"text/csv":                   &CSVCodec{},
	"application/csv":          &CSVCodec{},
	"multipart/form-data":        &MultipartCodec{},
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

// --- Cached body reader ---

// cachedBodyReader wraps a *bytes.Reader to implement io.ReadCloser
// while preserving io.Seeker, so Body() can seek back to position 0.
type cachedBodyReader struct {
	*bytes.Reader
}

func (cachedBodyReader) Close() error { return nil }

// NewCachedBodyReader creates an io.ReadCloser backed by a *bytes.Reader.
// Unlike io.NopCloser, it preserves the io.Seeker interface so the reader
// can be rewound via Seek(0, io.SeekStart).
func NewCachedBodyReader(data []byte) io.ReadCloser {
	return &cachedBodyReader{Reader: bytes.NewReader(data)}
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
			if err == io.EOF {
				return total, nil
			}
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

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
	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
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

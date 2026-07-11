package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type testStruct struct {
	Name  string `json:"name" xml:"name" yaml:"name"`
	Value int    `json:"value" xml:"value" yaml:"value"`
}

func newTestResponse(body string, contentType string) *http.Response {
	return &http.Response{
		Header: map[string][]string{"Content-Type": {contentType}},
		Body:   http.MaxBytesReader(nil, http.NoBody, 0),
	}
}

func newResponseWithBody(body string, contentType string) *http.Response {
	return &http.Response{
		Header: map[string][]string{"Content-Type": {contentType}},
		Body:   http.MaxBytesReader(nil, http.NoBody, 0),
	}
}

func TestJSONCodec(t *testing.T) {
	c := &JSONCodec{}

	// Marshal
	b, err := c.Marshal(testStruct{Name: "Alice", Value: 42})
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"name":"Alice","value":42}`
	if string(b) != expected {
		t.Fatalf("got %s, want %s", b, expected)
	}

	// Unmarshal
	var v testStruct
	if err := c.Unmarshal([]byte(expected), &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Alice" || v.Value != 42 {
		t.Fatalf("got %+v", v)
	}
}

func TestXMLCodec(t *testing.T) {
	c := &XMLCodec{}

	// Marshal
	b, err := c.Marshal(testStruct{Name: "Alice", Value: 42})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty xml output")
	}

	// Unmarshal
	var v testStruct
	xmlData := []byte(`<testStruct><name>Alice</name><value>42</value></testStruct>`)
	if err := c.Unmarshal(xmlData, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Alice" || v.Value != 42 {
		t.Fatalf("got %+v", v)
	}
}

func TestYAMLCodec(t *testing.T) {
	c := &YAMLCodec{}

	// Marshal
	b, err := c.Marshal(testStruct{Name: "Alice", Value: 42})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty yaml output")
	}

	// Unmarshal
	var v testStruct
	yamlData := []byte("name: Alice\nvalue: 42\n")
	if err := c.Unmarshal(yamlData, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Alice" || v.Value != 42 {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalResponse_JSON(t *testing.T) {
	body := `{"name":"Bob","value":99}`
	resp := httptest.NewRecorder().Result()
	resp.Body = http.MaxBytesReader(nil, http.NoBody, 0)

	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteString(body)
	resp = rec.Result()

	var v testStruct
	if err := UnmarshalResponse(resp, nil, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Bob" || v.Value != 99 {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalResponse_XML(t *testing.T) {
	body := `<testStruct><name>Bob</name><value>99</value></testStruct>`
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/xml")
	rec.WriteString(body)
	resp := rec.Result()

	var v testStruct
	if err := UnmarshalResponse(resp, nil, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Bob" || v.Value != 99 {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalResponse_YAML(t *testing.T) {
	body := "name: Bob\nvalue: 99\n"
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/x-yaml")
	rec.WriteString(body)
	resp := rec.Result()

	var v testStruct
	if err := UnmarshalResponse(resp, nil, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Bob" || v.Value != 99 {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalResponse_CustomCodec(t *testing.T) {
	body := `{"name":"Custom","value":1}`
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteString(body)
	resp := rec.Result()

	var v testStruct
	codecs := map[string]Codec{"application/json": &JSONCodec{}}
	if err := UnmarshalResponse(resp, codecs, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Custom" || v.Value != 1 {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalResponse_FallbackToJSON(t *testing.T) {
	body := `{"name":"Fallback","value":7}`
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/octet-stream")
	rec.WriteString(body)
	resp := rec.Result()

	var v testStruct
	if err := UnmarshalResponse(resp, nil, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Fallback" || v.Value != 7 {
		t.Fatalf("got %+v", v)
	}
}

func TestUnmarshalResponse_TextXML(t *testing.T) {
	body := `<testStruct><name>Text</name><value>5</value></testStruct>`
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "text/xml")
	rec.WriteString(body)
	resp := rec.Result()

	var v testStruct
	if err := UnmarshalResponse(resp, nil, &v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "Text" || v.Value != 5 {
		t.Fatalf("got %+v", v)
	}
}

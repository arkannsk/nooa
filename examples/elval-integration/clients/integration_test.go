package clients_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	basictypesdemo "github.com/arkannsk/nooa/examples/elval-integration/clients/01_basic_types"
	nesteddemo "github.com/arkannsk/nooa/examples/elval-integration/clients/03_nested"
	collectionsdemo "github.com/arkannsk/nooa/examples/elval-integration/clients/04_slice_maps"
	httpparamsdemo "github.com/arkannsk/nooa/examples/elval-integration/clients/08_http_params"
	responsecontentdemo "github.com/arkannsk/nooa/examples/elval-integration/clients/15_response_content"
	basictypes "github.com/arkannsk/nooa/examples/models/01_basic_types"
	collections "github.com/arkannsk/nooa/examples/models/04_slice_maps"
	httpparams "github.com/arkannsk/nooa/examples/models/08_http_params"
	nested "github.com/arkannsk/nooa/examples/models/03_nested"
	responsecontent "github.com/arkannsk/nooa/examples/models/15_response_content"
)

// --- 01 Basic Types ---

func TestBasicTypes_POSTPrimitives(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req basictypes.SimplePrimitives
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(req)
	}))
	defer server.Close()

	client := basictypesdemo.New(server.URL, nil)
	input := &basictypes.SimplePrimitives{
		Name:      "test",
		Count:     42,
		BigNumber: 999,
		Rate:      3.14,
		Active:    true,
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	resp, err := client.POSTPrimitives(context.Background(), input)
	if err != nil {
		t.Fatalf("POSTPrimitives failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusCreated()
	if err != nil {
		t.Fatalf("StatusCreated failed: %v", err)
	}

	if got.Name != input.Name || got.Count != input.Count {
		t.Fatalf("mismatch: got %+v, want %+v", got, input)
	}
}

func TestBasicTypes_GETPointers(t *testing.T) {
	name := "optional_name"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := basictypes.WithPointers{Name: &name}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := basictypesdemo.New(server.URL, nil)
	resp, err := client.GETPointers(context.Background(), nil)
	if err != nil {
		t.Fatalf("GETPointers failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if got.Name == nil || *got.Name != name {
		t.Fatalf("expected name=%q, got %v", name, got.Name)
	}
}

func TestBasicTypes_POSTDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req basictypes.WithDefaults
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(req)
	}))
	defer server.Close()

	client := basictypesdemo.New(server.URL, nil)
	input := &basictypes.WithDefaults{Status: "active", Limit: 50, Enabled: false}

	resp, err := client.POSTDefaults(context.Background(), input)
	if err != nil {
		t.Fatalf("POSTDefaults failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusCreated()
	if err != nil {
		t.Fatalf("StatusCreated failed: %v", err)
	}

	if got.Status != input.Status || got.Limit != input.Limit {
		t.Fatalf("mismatch: got %+v, want %+v", got, input)
	}
}

// --- 03 Nested ---

func TestNested_GETUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := nested.UserWithAddress{
			ID:   "user_001",
			Name: "Alice",
			Billing: nested.Address{
				Street:  "Main St",
				City:    "Springfield",
				ZipCode: "12345",
				Country: "US",
			},
			Shipping: &nested.Address{
				Street:  "456 Side St",
				City:    "Los Angeles",
				ZipCode: "90001",
				Country: "US",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := nesteddemo.New(server.URL, nil)
	resp, err := client.GETUser(context.Background(), nil)
	if err != nil {
		t.Fatalf("GETUser failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if got.Name != "Alice" || got.Billing.City != "Springfield" {
		t.Fatalf("mismatch: got %+v", got)
	}
	if got.Shipping == nil || got.Shipping.City != "Los Angeles" {
		t.Fatalf("shipping mismatch: got %+v", got.Shipping)
	}
}

// --- 04 Slice/Maps ---

func TestCollections_GETSlices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := collections.SliceVariations{
			Tags: []string{"a", "b", "c"},
			IDs:  []int{1, 2, 3},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := collectionsdemo.New(server.URL, nil)
	resp, err := client.GETSlices(context.Background(), nil)
	if err != nil {
		t.Fatalf("GETSlices failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if len(got.Tags) != 3 || got.Tags[0] != "a" {
		t.Fatalf("tags mismatch: %v", got.Tags)
	}
	if len(got.IDs) != 3 || got.IDs[0] != 1 {
		t.Fatalf("ids mismatch: %v", got.IDs)
	}
}

func TestCollections_GETMaps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := collections.MapVariations{
			Metadata: map[string]string{"key": "value"},
			Counts:   map[string]int{"count": 42},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := collectionsdemo.New(server.URL, nil)
	resp, err := client.GETMaps(context.Background(), nil)
	if err != nil {
		t.Fatalf("GETMaps failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if got.Metadata["key"] != "value" || got.Counts["count"] != 42 {
		t.Fatalf("maps mismatch: %v", got)
	}
}

// --- 08 HTTP Params ---

func TestHTTPParams_GETSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// client sends "query" param, server echoes it back
		res := httpparams.QueryParams{
			Query:     q.Get("query"),
			Page:      1,
			Limit:     20,
			Status:    q.Get("status"),
			BodyField: "echo",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := httpparamsdemo.New(server.URL, nil)
	input := &httpparams.QueryParams{
		Query:     "test",
		Page:      1,
		Limit:     20,
		Status:    "active",
		BodyField: "echo",
	}

	resp, err := client.GETSearch(context.Background(), input)
	if err != nil {
		t.Fatalf("GETSearch failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if got.Query != "test" {
		t.Fatalf("query mismatch: got %q", got.Query)
	}
}

func TestHTTPParams_PUTPathParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := httpparams.PathParams{
			UserID:     "user123",
			ResourceID: 456,
			Payload:    "update",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := httpparamsdemo.New(server.URL, nil)
	input := &httpparams.PathParams{
		UserID:     "user123",
		ResourceID: 456,
		Payload:    "update",
	}

	resp, err := client.PUTUsersUserIdResourcesResourceid(context.Background(), input)
	if err != nil {
		t.Fatalf("PUTUsersUserIdResourcesResourceid failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if got.UserID != "user123" || got.ResourceID != 456 {
		t.Fatalf("path params mismatch: got %+v", got)
	}
}

// --- 15 Response Content ---

func TestResponseContent_GETUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := responsecontent.UserResponse{
			ID:    1,
			Name:  "Alice",
			Email: "alice@example.com",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := responsecontentdemo.New(server.URL, nil)
	resp, err := client.Responses.GETUser(context.Background())
	if err != nil {
		t.Fatalf("GETUser failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusOk()
	if err != nil {
		t.Fatalf("StatusOk failed: %v", err)
	}

	if got.ID != 1 || got.Name != "Alice" {
		t.Fatalf("mismatch: got %+v", got)
	}
}

func TestResponseContent_POSTCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req responsecontent.CreateUserResponse
		json.NewDecoder(r.Body).Decode(&req)
		req.CreatedAt = "2024-01-01T00:00:00Z"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(req)
	}))
	defer server.Close()

	client := responsecontentdemo.New(server.URL, nil)
	input := &responsecontent.CreateUserResponse{
		ID:   42,
		Name: "Bob",
	}

	resp, err := client.Responses.POSTCreate(context.Background(), input)
	if err != nil {
		t.Fatalf("POSTCreate failed: %v", err)
	}
	defer resp.Close()

	got, err := resp.StatusCreated()
	if err != nil {
		t.Fatalf("StatusCreated failed: %v", err)
	}

	if got.ID != 42 || got.Name != "Bob" {
		t.Fatalf("mismatch: got %+v", got)
	}
}

// --- Body caching / streaming ---

func TestBodyCaching_StatusOkThenBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := basictypes.SimplePrimitives{Name: "cached", Count: 100}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := basictypesdemo.New(server.URL, nil)
	resp, err := client.POSTPrimitives(context.Background(), &basictypes.SimplePrimitives{})
	if err != nil {
		t.Fatalf("POSTPrimitives failed: %v", err)
	}
	defer resp.Close()

	// First: deserialize
	got, err := resp.StatusCreated()
	if err != nil {
		t.Fatalf("StatusCreated failed: %v", err)
	}
	if got.Name != "cached" {
		t.Fatalf("expected cached, got %q", got.Name)
	}

	// Second: read body again (should be cached and seekable)
	body := resp.Body()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Body read failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Body is empty after StatusOk — caching not working")
	}
}

func TestBodyCaching_BodyThenStatusOk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := basictypes.SimplePrimitives{Name: "streamed", Count: 200}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := basictypesdemo.New(server.URL, nil)
	resp, err := client.POSTPrimitives(context.Background(), &basictypes.SimplePrimitives{})
	if err != nil {
		t.Fatalf("POSTPrimitives failed: %v", err)
	}
	defer resp.Close()

	// First: call StatusCreated which caches the body
	got, err := resp.StatusCreated()
	if err != nil {
		t.Fatalf("StatusCreated failed: %v", err)
	}
	if got.Name != "streamed" {
		t.Fatalf("expected streamed, got %q", got.Name)
	}

	// Second: read body again via Body()
	body := resp.Body()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Body read failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Body is empty after StatusOk — caching not working")
	}

	// Third: read body a third time
	body2 := resp.Body()
	data2, err := io.ReadAll(body2)
	if err != nil {
		t.Fatalf("Body second read failed: %v", err)
	}
	if !bytes.Equal(data, data2) {
		t.Fatal("Body reads don't match — seek not working")
	}
}

func TestStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := basictypesdemo.New(server.URL, nil)
	resp, err := client.POSTDefaults(context.Background(), &basictypes.WithDefaults{})
	if err != nil {
		t.Fatalf("POSTDefaults failed: %v", err)
	}
	defer resp.Close()

	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode())
	}
}

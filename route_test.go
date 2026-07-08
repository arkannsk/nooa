package nooa

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type Req struct{}
type Res struct {
	ID string `json:"id"`
}

func handlerOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func TestNewRoute_TwoTypes(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	b := NewRoute[Req, Res]("POST", "/users", handlerOK).
		Summary("Create user").
		Tags("users").
		OnSuccess(201, "Created").
		PossibleErr(400, 500)

	spec := b.Spec()

	if spec.Method != "POST" || spec.Path != "/users" {
		t.Fatalf("unexpected method/path: %s %s", spec.Method, spec.Path)
	}
	if spec.Summary != "Create user" {
		t.Fatalf("unexpected summary: %s", spec.Summary)
	}
	if len(spec.Tags) != 1 || spec.Tags[0] != "users" {
		t.Fatalf("unexpected tags: %v", spec.Tags)
	}
	if len(spec.Responses) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(spec.Responses))
	}

	// Проверка 2xx
	r := spec.Responses[0]
	if r.Status != 201 || r.Description != "Created" || r.IsError {
		t.Errorf("bad success response: %+v", r)
	}
	if len(r.ContentTypes) != 1 || r.ContentTypes[0] != CTJSON {
		t.Errorf("bad content type: %v", r.ContentTypes)
	}

	// Проверка 4xx (должен быть problem+json по дефолту)
	e := spec.Responses[1]
	if e.Status != 400 || !e.IsError {
		t.Errorf("bad client error: %+v", e)
	}
	if len(e.ContentTypes) != 1 || e.ContentTypes[0] != CTProblemJSON {
		t.Errorf("client error should default to problem+json, got %v", e.ContentTypes)
	}
}

func TestRegisterAndGlobalRegistry(t *testing.T) {
	ClearRegistry()
	defer ClearRegistry()

	mux := http.NewServeMux()

	// Вариант 1: Цепочка (теперь работает!)
	NewRoute[Req, Res]("POST", "/b", handlerOK).
		Summary("B").
		Register(mux)

	// Вариант 2: Раздельные вызовы (тоже работает)
	NewRoute[Req, Res]("GET", "/a", handlerOK).
		Summary("A").
		Register(mux)

	routes := RegisteredRoutes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	// Проверка путей (порядок может быть любым, так как мы добавили два маршрута)
	paths := map[string]bool{routes[0].Path: true, routes[1].Path: true}
	if !paths["/a"] || !paths["/b"] {
		t.Fatalf("unexpected paths in registry: %v", paths)
	}

	// Проверка, что mux тоже работает (проверяем маршрут /b)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/b", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from mux, got %d", rec.Code)
	}
}

func TestMiddlewareChaining(t *testing.T) {
	var order []string

	mw1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1")
			next(w, r)
		}
	}
	mw2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2")
			next(w, r)
		}
	}
	mw3 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw3")
			next(w, r)
		}
	}

	ClearRegistry()
	defer ClearRegistry()

	mux := http.NewServeMux()

	NewRoute[Req, Res]("GET", "/mw", handlerOK).
		Use(mw1, mw2).
		Use(mw3).
		Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/mw", nil)
	mux.ServeHTTP(rec, req)

	// mw1 и mw2 добавлены первыми — они самые внешние.
	// mw3 добавлен последним — он ближайший к handler.
	// Порядок вызова: mw1 -> mw2 -> mw3 -> handler
	want := []string{"mw1", "mw2", "mw3"}
	if len(order) != len(want) {
		t.Fatalf("expected %d middleware calls, got %d: %v", len(want), len(order), order)
	}
	for i, w := range want {
		if order[i] != w {
			t.Errorf("order[%d] = %q, want %q", i, order[i], w)
		}
	}
}

func TestOperationIDGeneration(t *testing.T) {
	tests := []struct {
		method, path, expected string
	}{
		{"GET", "/", "GETRoot"},
		{"GET", "/users/{id}", "GETUsersId"},
		{"POST", "/api/v1/orders", "POSTApiV1Orders"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := defaultOperationID(tt.method, tt.path); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

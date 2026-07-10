package ginAdapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arkannsk/nooa"
	"github.com/gin-gonic/gin"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	route := nooa.NewRoute[struct{}, struct{}]("GET", "/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}).Spec()

	Register(r, route)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/hello", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected 'ok', got %q", w.Body.String())
	}
}

func TestRegisterSpecAndMux(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	spec := nooa.NewSpec(nooa.Info{Title: "test", Version: "1.0"})
	spec.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Spec-Mw", "1")
			next(w, r)
		}
	})

	route := nooa.NewRoute[struct{}, struct{}]("GET", "/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Spec()

	RegisterSpecAndMux(spec, r, route)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Spec-Mw") != "1" {
		t.Error("expected spec middleware to run")
	}
}

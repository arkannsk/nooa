package fiberAdapter

import (
	"io"
	"net/http"
	"testing"

	"github.com/arkannsk/nooa"
	"github.com/gofiber/fiber/v2"
)

func TestRegister(t *testing.T) {
	app := fiber.New()

	route := nooa.NewRoute[struct{}, struct{}]("GET", "/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}).Spec()

	Register(app, route)

	req, _ := http.NewRequest("GET", "/hello", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok" {
		t.Fatalf("expected 'ok', got %q", body)
	}
}

func TestRegisterSpecAndMux(t *testing.T) {
	app := fiber.New()

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

	RegisterSpecAndMux(spec, app, route)

	req, _ := http.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Spec-Mw") != "1" {
		t.Error("expected spec middleware to run")
	}
	resp.Body.Close()
}

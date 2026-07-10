// Package fiber provides an adapter to register nooa routes with gofiber/fiber.
//
// Usage:
//
//	app := fiber.New()
//	fiber.Register(app, nooa.NewRoute[Req, Res]("GET", "/users", handler).Spec())
package fiberAdapter

import (
	"bytes"
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/gofiber/fiber/v2"
)

// fw wraps fiber.Ctx to satisfy http.ResponseWriter.
// Headers are collected in a buffer and committed to the fiber response after the handler returns.
type fw struct {
	c       *fiber.Ctx
	written bool
	headers http.Header
}

func newFW(c *fiber.Ctx) *fw {
	return &fw{c: c, headers: make(http.Header)}
}

func (w *fw) Header() http.Header {
	return w.headers
}

func (w *fw) Write(p []byte) (int, error) {
	if !w.written {
		w.c.Status(http.StatusOK)
		w.written = true
	}
	_, _ = w.c.Write(p)
	return len(p), nil
}

func (w *fw) WriteHeader(status int) {
	if !w.written {
		w.c.Status(status)
		w.written = true
	}
}

// WrapHandler wraps a standard http.HandlerFunc for fiber.Ctx.
// Use it to mount any http.Handler (e.g. nooa.RedocUIHandler) in Fiber.
func WrapHandler(h http.HandlerFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		w := newFW(c)
		req, err := http.NewRequest(c.Method(), c.OriginalURL(), bytes.NewReader(c.Body()))
		if err != nil {
			return err
		}
		req.Host = c.Hostname()
		req.RemoteAddr = c.IP()
		for k, vs := range c.GetReqHeaders() {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}

		h(w, req)
		w.commitHeaders()
		return nil
	}
}

// commitHeaders copies the collected headers to the fiber response.
func (w *fw) commitHeaders() {
	for k, vs := range w.headers {
		for _, v := range vs {
			w.c.Set(k, v)
		}
	}
}

// Register registers a single RouteSpec with a fiber.App.
func Register(app *fiber.App, route nooa.RouteSpec) {
	handled := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handled = route.Middlewares[i](handled)
	}
	app.Add(route.Method, route.Path, WrapHandler(handled))
}

// RegisterSpecAndMux registers multiple routes from a Spec into a fiber.App.
// It also adds each route to the Spec so that OpenAPI generation includes them.
func RegisterSpecAndMux(spec *nooa.Spec, app *fiber.App, routes ...nooa.RouteSpec) {
	specMws := spec.Middlewares()

	for _, route := range routes {
		spec.AddRoute(route)

		handled := route.Handler
		for i := len(route.Middlewares) - 1; i >= 0; i-- {
			handled = route.Middlewares[i](handled)
		}
		for i := len(specMws) - 1; i >= 0; i-- {
			handled = specMws[i](handled)
		}
		app.Add(route.Method, route.Path, WrapHandler(handled))
	}
}

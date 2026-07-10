// Package gin provides an adapter to register nooa routes with gin-gonic/gin.
//
// Usage:
//
//	r := gin.Default()
//	gin.RegisterSpecAndMux(spec, r,
//	    nooa.NewRoute[Req, Res]("GET", "/users", handler).Spec(),
//	    nooa.NewRoute[Req, Res]("POST", "/users", handler).Spec(),
//	)
package ginAdapter

import (
	"net/http"

	"github.com/arkannsk/nooa"
	"github.com/gin-gonic/gin"
)

// WrapHandler wraps a standard http.HandlerFunc for gin.Context.
// Use it to mount any http.Handler (e.g. nooa.RedocUIHandler) in Gin.
func WrapHandler(h http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(c.Writer, c.Request)
	}
}

// Register registers a single RouteSpec with a gin.Engine.
// Middleware from the spec is applied in order: spec middlewares → route middlewares → handler.
func Register(r *gin.Engine, route nooa.RouteSpec) {
	handled := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handled = route.Middlewares[i](handled)
	}
	r.Handle(route.Method, route.Path, WrapHandler(handled))
}

// RegisterSpecAndMux registers multiple routes from a Spec into a gin.Engine.
// It applies spec-level middleware as the outermost layer.
// It also adds each route to the Spec so that OpenAPI generation includes them.
func RegisterSpecAndMux(spec *nooa.Spec, r *gin.Engine, routes ...nooa.RouteSpec) {
	for _, route := range routes {
		spec.AddRoute(route)

		handled := route.Handler
		// Route middlewares (inner)
		for i := len(route.Middlewares) - 1; i >= 0; i-- {
			handled = route.Middlewares[i](handled)
		}
		// Spec middlewares (outer)
		specMws := spec.Middlewares()
		for i := len(specMws) - 1; i >= 0; i-- {
			handled = specMws[i](handled)
		}
		r.Handle(route.Method, route.Path, WrapHandler(handled))
	}
}

// Package nooa provides a type-safe, annotation-free HTTP router
// with compile-time OpenAPI 3.0 specification generation support.
package nooa

import (
	"net/http"
	"strings"
	"sync"
)

// Content-Type константы
const (
	CTJSON        = "application/json"
	CTProblemJSON = "application/problem+json" // RFC 7807
	CTXML         = "application/xml"
	CTForm        = "application/x-www-form-urlencoded"
	CTMultipart   = "multipart/form-data"
	CTOctetStream = "application/octet-stream"
	CTPNG         = "image/png"
	CTHTML        = "text/html"
	CTPlainText   = "text/plain"
	CTCSV         = "text/csv" // <--- ДОБАВИТЬ ЭТУ СТРОКУ
)

// ResponseSpec хранит метаданные ответа
type ResponseSpec struct {
	Status       int
	Description  string
	ContentTypes []string
	IsError      bool
}

// SecurityRequirement для OpenAPI
type SecurityRequirement struct {
	Scheme string
	Scopes []string
}

// RouteSpec — публичное описание маршрута (используется в рантайме/startup)
type RouteSpec struct {
	Method             string
	Path               string
	OperationID        string
	Summary            string
	Description        string
	Tags               []string
	Deprecated         bool
	Security           []SecurityRequirement
	RequestContentType []string
	Responses          []ResponseSpec
	Handler            http.HandlerFunc
}

// RouteBuilder — fluent API. Req и Res используются генератором (AST),
// в рантайме типы стираются, оверхед отсутствует.
type RouteBuilder[Req, Res any] struct {
	method             string
	path               string
	summary            string
	description        string
	operationID        string
	tags               []string
	security           []SecurityRequirement
	requestContentType []string
	deprecated         bool
	handler            http.HandlerFunc
	responses          []ResponseSpec
	spec               *RouteSpec
}

// NewRoute создаёт билдер маршрута.
// Req — тип входящего запроса (body/params)
// Res — тип успешного ответа (2xx)
func NewRoute[Req, Res any](method, path string, handler http.HandlerFunc) *RouteBuilder[Req, Res] {
	b := &RouteBuilder[Req, Res]{
		method:             strings.ToUpper(method),
		path:               path,
		handler:            handler,
		operationID:        defaultOperationID(method, path),
		requestContentType: []string{CTJSON},
	}
	b.syncSpec()
	return b
}

func defaultOperationID(method, path string) string {
	method = strings.ToUpper(method)
	path = strings.Trim(path, "/ ")
	if path == "" {
		return method + "Root"
	}
	var sb strings.Builder
	sb.WriteString(method)
	for _, p := range strings.Split(path, "/") {
		p = strings.Trim(p, "{} ")
		if p == "" {
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			sb.WriteString(p[1:])
		}
	}
	return sb.String()
}

func (b *RouteBuilder[Req, Res]) syncSpec() {
	if b.spec == nil {
		b.spec = &RouteSpec{}
	}
	b.spec.Method = b.method
	b.spec.Path = b.path
	b.spec.OperationID = b.operationID
	b.spec.Summary = b.summary
	b.spec.Description = b.description
	b.spec.Tags = append([]string(nil), b.tags...)
	b.spec.Deprecated = b.deprecated
	b.spec.Security = append([]SecurityRequirement(nil), b.security...)
	b.spec.RequestContentType = append([]string(nil), b.requestContentType...)
	b.spec.Responses = append([]ResponseSpec(nil), b.responses...)
	b.spec.Handler = b.handler
}

// === METADATA ===

func (b *RouteBuilder[Req, Res]) Summary(s string) *RouteBuilder[Req, Res] {
	b.summary = s
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) Description(s string) *RouteBuilder[Req, Res] {
	b.description = s
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) Tags(tags ...string) *RouteBuilder[Req, Res] {
	b.tags = append(b.tags, tags...)
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) OperationID(id string) *RouteBuilder[Req, Res] {
	b.operationID = id
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) Deprecated() *RouteBuilder[Req, Res] {
	b.deprecated = true
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) Secure(scheme string, scopes ...string) *RouteBuilder[Req, Res] {
	b.security = append(b.security, SecurityRequirement{Scheme: scheme, Scopes: scopes})
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) RequestContentType(cts ...string) *RouteBuilder[Req, Res] {
	if len(cts) > 0 {
		b.requestContentType = cts
		b.syncSpec()
	}
	return b
}

// === RESPONSES (без дженериков, типы берутся из Res или глобальных моделей ошибок) ===

// OnSuccess документирует 2xx ответ.
func (b *RouteBuilder[Req, Res]) OnSuccess(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	b.addResponse(status, desc, ct, false)
	return b
}

// OnClientErr документирует 4xx ответ. Дефолтный CT: application/problem+json.
func (b *RouteBuilder[Req, Res]) OnClientErr(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	if len(ct) == 0 {
		ct = []string{CTProblemJSON}
	}
	b.addResponse(status, desc, ct, true)
	return b
}

// OnServerErr документирует 5xx ответ.
func (b *RouteBuilder[Req, Res]) OnServerErr(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	if len(ct) == 0 {
		ct = []string{CTProblemJSON}
	}
	b.addResponse(status, desc, ct, true)
	return b
}

// OnNoContent для ответов без тела
func (b *RouteBuilder[Req, Res]) OnNoContent(status int, desc string) *RouteBuilder[Req, Res] {
	b.responses = append(b.responses, ResponseSpec{
		Status:      status,
		Description: desc,
		IsError:     false,
	})
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) addResponse(status int, desc string, ct []string, isError bool) {
	if len(ct) == 0 {
		ct = []string{CTJSON}
	}
	b.responses = append(b.responses, ResponseSpec{
		Status:       status,
		Description:  desc,
		ContentTypes: ct,
		IsError:      isError,
	})
	b.syncSpec()
}

// === TERMINAL ===

// Register регистрирует хендлер в ServeMux и возвращает билдер для продолжения цепочки.
// Это позволяет писать: NewRoute(...).Register(mux).RegisterGlobal()
func (b *RouteBuilder[Req, Res]) Register(mux *http.ServeMux) *RouteBuilder[Req, Res] {
	if b.handler == nil {
		panic("nooa: handler cannot be nil")
	}
	if b.path == "" {
		panic("nooa: path cannot be empty")
	}
	mux.HandleFunc(b.method+" "+b.path, b.handler)
	return b
}

// Spec возвращает копию метаданных
func (b *RouteBuilder[Req, Res]) Spec() RouteSpec {
	if b.spec == nil {
		b.syncSpec()
	}
	spec := *b.spec
	spec.Tags = append([]string(nil), b.spec.Tags...)
	spec.Security = append([]SecurityRequirement(nil), b.spec.Security...)
	spec.RequestContentType = append([]string(nil), b.spec.RequestContentType...)
	spec.Responses = append([]ResponseSpec(nil), b.spec.Responses...)
	return spec
}

// GLOBAL REGISTRY
var (
	registryMu    sync.RWMutex
	routeRegistry []RouteSpec
)

// RegisterGlobal добавляет маршрут в глобальный реестр и возвращает билдер.
func (b *RouteBuilder[Req, Res]) RegisterGlobal() *RouteBuilder[Req, Res] {
	registryMu.Lock()
	defer registryMu.Unlock()
	routeRegistry = append(routeRegistry, b.Spec())
	return b
}

// RegisteredRoutes возвращает копию всех зарегистрированных маршрутов.
func RegisteredRoutes() []RouteSpec {
	registryMu.RLock()
	defer registryMu.RUnlock()
	result := make([]RouteSpec, len(routeRegistry))
	copy(result, routeRegistry)
	return result
}

// ClearRegistry очищает реестр (полезно для тестов).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	routeRegistry = nil
}

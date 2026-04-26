package nooa

import (
	"net/http"
	"path"
	"reflect"
	"strings"
)

const (
	CTJSON        = "application/json"
	CTProblemJSON = "application/problem+json"
	CTXML         = "application/xml"
	CTForm        = "application/x-www-form-urlencoded"
	CTMultipart   = "multipart/form-data"
	CTOctetStream = "application/octet-stream"
	CTPNG         = "image/png"
	CTHTML        = "text/html"
	CTPlainText   = "text/plain"
	CTCSV         = "text/csv"
)

type ResponseSpec struct {
	Status       int
	Description  string
	ContentTypes []string
	IsError      bool
}

type SecurityRequirement struct {
	Scheme string
	Scopes []string
}

type RouteSpec struct {
	Method                string
	Path                  string
	OperationID           string
	Summary               string
	Description           string
	Tags                  []string
	Deprecated            bool
	Security              []SecurityRequirement
	RequestContentType    []string
	Responses             []ResponseSpec
	Handler               http.HandlerFunc
	Extensions            map[string]any
	RequestBodySchemaName string
	ResponseSchemaNames   map[int]string // [Status Code] -> Schema Name
}

type RouteBuilder[Req, Res any] struct {
	method                string
	path                  string
	summary               string
	description           string
	operationID           string
	tags                  []string
	security              []SecurityRequirement
	requestContentType    []string
	deprecated            bool
	handler               http.HandlerFunc
	responses             []ResponseSpec
	spec                  *RouteSpec
	extensions            map[string]any
	requestBodySchemaName string
	responseSchemaNames   map[int]string
}

func NewRoute[Req, Res any](method, path string, handler http.HandlerFunc) *RouteBuilder[Req, Res] {
	b := &RouteBuilder[Req, Res]{
		method:             strings.ToUpper(method),
		path:               path,
		handler:            handler,
		operationID:        defaultOperationID(method, path),
		requestContentType: []string{CTJSON},
	}
	reqSchemaName := getSchemaName[Req]()
	resSchemaName := getSchemaName[Res]()

	RegisterModel(reqSchemaName, new(Req))
	RegisterModel(resSchemaName, new(Res))

	if method != "GET" && method != "HEAD" && method != "DELETE" {
		b.RequestBodySchema(reqSchemaName)
	}
	// Автоматически привязываем основной ответ к 200/201
	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[200] = resSchemaName
	b.responseSchemaNames[201] = resSchemaName

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
		b.spec.ResponseSchemaNames = make(map[int]string)
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
	b.spec.RequestBodySchemaName = b.requestBodySchemaName

	// Копирование расширений (исправлено)
	if len(b.extensions) > 0 {
		if b.spec.Extensions == nil {
			b.spec.Extensions = make(map[string]any)
		}
		for k, v := range b.extensions {
			b.spec.Extensions[k] = v
		}
	}

	if b.responseSchemaNames != nil {
		b.spec.ResponseSchemaNames = make(map[int]string)
		for k, v := range b.responseSchemaNames {
			b.spec.ResponseSchemaNames[k] = v
		}
	}
}

// Вспомогательное поле для накопления расширений до syncSpec
var extensionsTemp map[string]any

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

func (b *RouteBuilder[Req, Res]) Extension(key string, value any) *RouteBuilder[Req, Res] {
	if b.extensions == nil {
		b.extensions = make(map[string]any)
	}
	b.extensions[key] = value
	// Обновляем spec, чтобы изменения были видны сразу при вызове Spec()
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) RequestBodySchema(name string) *RouteBuilder[Req, Res] {
	b.requestBodySchemaName = name
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) ResponseSchema(status int, schemaName string) *RouteBuilder[Req, Res] {
	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[status] = schemaName
	b.syncSpec()
	return b
}

// Prefix добавляет префикс к пути (например, /api/v1)
func (b *RouteBuilder[Req, Res]) Prefix(prefix string) *RouteBuilder[Req, Res] {
	prefix = strings.TrimRight(prefix, "/")
	currentPath := strings.TrimLeft(b.path, "/")

	if prefix != "" && currentPath != "" {
		b.path = prefix + "/" + currentPath
	} else if prefix != "" {
		b.path = prefix
	}
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) OnSuccess(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	b.addResponse(status, desc, ct, false)
	return b
}

func (b *RouteBuilder[Req, Res]) OnClientErr(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	if len(ct) == 0 {
		ct = []string{CTProblemJSON}
	}
	b.addResponse(status, desc, ct, true)
	return b
}

func (b *RouteBuilder[Req, Res]) OnServerErr(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	if len(ct) == 0 {
		ct = []string{CTProblemJSON}
	}
	b.addResponse(status, desc, ct, true)
	return b
}

func (b *RouteBuilder[Req, Res]) OnNoContent(status int, desc string) *RouteBuilder[Req, Res] {
	b.responses = append(b.responses, ResponseSpec{Status: status, Description: desc, IsError: false})
	b.syncSpec()
	return b
}

func (b *RouteBuilder[Req, Res]) addResponse(status int, desc string, ct []string, isError bool) {
	if len(ct) == 0 {
		ct = []string{CTJSON}
	}
	b.responses = append(b.responses, ResponseSpec{
		Status: status, Description: desc, ContentTypes: ct, IsError: isError,
	})
	b.syncSpec()
}
func (b *RouteBuilder[Req, Res]) Register(mux *http.ServeMux) *RouteBuilder[Req, Res] {
	if b.handler == nil {
		panic("nooa: handler cannot be nil")
	}
	if b.path == "" {
		panic("nooa: path cannot be empty")
	}
	mux.HandleFunc(b.method+" "+b.path, b.handler)
	b.registerGlobal()
	return b
}

func (b *RouteBuilder[Req, Res]) Spec() RouteSpec {
	if b.spec == nil {
		b.syncSpec()
	}
	// Глубокая копия для безопасности
	spec := *b.spec
	spec.Tags = append([]string(nil), b.spec.Tags...)
	spec.Security = append([]SecurityRequirement(nil), b.spec.Security...)
	spec.RequestContentType = append([]string(nil), b.spec.RequestContentType...)
	spec.Responses = append([]ResponseSpec(nil), b.spec.Responses...)
	if b.spec.Extensions != nil {
		spec.Extensions = make(map[string]any)
		for k, v := range b.spec.Extensions {
			spec.Extensions[k] = v
		}
	}
	return spec
}

func (b *RouteBuilder[Req, Res]) registerGlobal() *RouteBuilder[Req, Res] {
	addToRegistryInternal(b.Spec())
	return b
}

func WithResponse[Req, Res, T any](b *RouteBuilder[Req, Res], status int) *RouteBuilder[Req, Res] {
	schemaName := getSchemaName[T]()
	RegisterModel(schemaName, new(T))

	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[status] = schemaName
	b.syncSpec()
	return b
}

func getSchemaName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	pkgPath := t.PkgPath()
	typeName := t.Name()

	// Для main-пакета или пустого пути берём только имя
	if pkgPath == "" || pkgPath == "main" {
		return typeName
	}

	// Берём имя последней папки (models, errors, dto, api и т.д.)
	pkgName := path.Base(pkgPath)
	return pkgName + "." + typeName
}

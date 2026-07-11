package nooa

import (
	"maps"
	"net/http"
	"path"
	"reflect"
	"strings"

	oa "github.com/arkannsk/elval/pkg/openapi"
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
	Middlewares           []func(http.HandlerFunc) http.HandlerFunc
	Extensions            map[string]any
	RequestBodySchemaName string
	ResponseSchemaNames   map[int]string       // [Status Code] -> Schema Name
	ErrorStatuses         []int                // статусы, для которых подтягиваются глобальные схемы ошибок из Spec
	Parameters            []*oa.Parameter      // OpenAPI параметры из OaParams() моделей
	ModelResponses        map[int]*oa.Response // OpenAPI ответы из OaResponses() моделей
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
	middlewares           []func(http.HandlerFunc) http.HandlerFunc
	extensions            map[string]any
	requestBodySchemaName string
	responseSchemaNames   map[int]string
	errorStatuses         []int
	parameters            []*oa.Parameter
	modelResponses        map[int]*oa.Response
}

func NewRoute[Req, Res any](method, path string, handler http.HandlerFunc) *RouteBuilder[Req, Res] {
	b := &RouteBuilder[Req, Res]{
		method:              strings.ToUpper(method),
		path:                path,
		handler:             handler,
		operationID:         defaultOperationID(method, path),
		requestContentType:  []string{CTJSON},
		responseSchemaNames: map[int]string{},
		modelResponses:      make(map[int]*oa.Response),
	}
	reqSchemaName := getSchemaName[Req]()
	resSchemaName := getSchemaName[Res]()

	reqInstance := new(Req)
	resInstance := new(Res)

	RegisterModel(reqSchemaName, reqInstance)
	RegisterModel(resSchemaName, resInstance)

	// Собираем параметры из OaParams() если модели поддерживают
	collectParams(reqInstance, &b.parameters)
	collectParams(resInstance, &b.parameters)

	// Собираем ответы из OaResponses() если модели поддерживают
	collectResponses(reqInstance, b.modelResponses)
	collectResponses(resInstance, b.modelResponses)

	if method != "GET" && method != "HEAD" && method != "DELETE" {
		b.requestBodySchemaName = reqSchemaName
	}
	// Привязываем схему ответа к статус-кодам, объявленным в модели через @oa:response.
	// Если модель не объявила статусы — дефолт 200.
	b.responseSchemaNames = make(map[int]string)
	if len(b.modelResponses) > 0 {
		for status := range b.modelResponses {
			b.responseSchemaNames[status] = resSchemaName
		}
	} else {
		b.responseSchemaNames[200] = resSchemaName
	}

	return b
}

// RouteBuilderMultiResp is defined in route_multi.go.

func defaultOperationID(method, path string) string {
	method = strings.ToUpper(method)
	path = strings.Trim(path, "/ ")
	if path == "" {
		return method + "Root"
	}
	var sb strings.Builder
	sb.WriteString(method)
	for p := range strings.SplitSeq(path, "/") {
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

// collectParams извлекает параметры из инстанса если он реализует paramsProvider.
// Параметры добавляются к destination, дубликаты по (name, in) пропускаются.
func collectParams(instance any, destination *[]*oa.Parameter) {
	pp, ok := any(instance).(paramsProvider)
	if !ok {
		return
	}

	// Строим set существующих параметров для дедупликации
	existing := make(map[string]bool)
	for _, p := range *destination {
		existing[p.Name+"/"+string(p.In)] = true
	}

	for _, p := range pp.OaParams() {
		if !existing[p.Name+"/"+string(p.In)] {
			*destination = append(*destination, p)
			existing[p.Name+"/"+string(p.In)] = true
		}
	}
}

// collectResponses извлекает ответы из инстанса если он реализует responsesProvider.
// Ответы добавляются к destination; при конфликте статусного кода приоритет у новых данных.
func collectResponses(instance any, destination map[int]*oa.Response) {
	rp, ok := any(instance).(responsesProvider)
	if !ok {
		return
	}

	maps.Copy(destination, rp.OaResponses())
}

func (b *RouteBuilder[Req, Res]) Summary(s string) *RouteBuilder[Req, Res] {
	b.summary = s
	return b
}

func (b *RouteBuilder[Req, Res]) Description(s string) *RouteBuilder[Req, Res] {
	b.description = s
	return b
}

func (b *RouteBuilder[Req, Res]) Tags(tags ...string) *RouteBuilder[Req, Res] {
	b.tags = append(b.tags, tags...)
	return b
}

func (b *RouteBuilder[Req, Res]) OperationID(id string) *RouteBuilder[Req, Res] {
	b.operationID = id
	return b
}

func (b *RouteBuilder[Req, Res]) Deprecated() *RouteBuilder[Req, Res] {
	b.deprecated = true
	return b
}

func (b *RouteBuilder[Req, Res]) Secure(scheme string, scopes ...string) *RouteBuilder[Req, Res] {
	b.security = append(b.security, SecurityRequirement{Scheme: scheme, Scopes: scopes})
	return b
}

func (b *RouteBuilder[Req, Res]) RequestContentType(cts ...string) *RouteBuilder[Req, Res] {
	if len(cts) > 0 {
		b.requestContentType = cts
	}
	return b
}

func (b *RouteBuilder[Req, Res]) Extension(key string, value any) *RouteBuilder[Req, Res] {
	if b.extensions == nil {
		b.extensions = make(map[string]any)
	}
	b.extensions[key] = value
	return b
}

func (b *RouteBuilder[Req, Res]) RequestBodySchema(name string) *RouteBuilder[Req, Res] {
	b.requestBodySchemaName = name
	return b
}

func (b *RouteBuilder[Req, Res]) ResponseSchema(status int, schemaName string) *RouteBuilder[Req, Res] {
	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[status] = schemaName
	return b
}

// Response добавляет произвольный ответ для указанного status code.
// Схема берётся из зарегистрированных моделей (spec.RegisterModel).
// Полезно когда роут возвращает разные типы для разных статусов:
//
//	Response(201, "CreatedUser", "User created").
//	Response(202, "AcceptedJob", "Processing...", nooa.CTJSON)
func (b *RouteBuilder[Req, Res]) Response(status int, schemaName string, desc string, ct ...string) *RouteBuilder[Req, Res] {
	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[status] = schemaName
	b.addResponse(status, desc, ct, false)
	return b
}

// Use добавляет middleware к роуту. Middleware применяются в порядке добавления
// (первый добавленный — самый внешний). Поддерживает стандартную сигнатуру
// func(http.HandlerFunc) http.HandlerFunc.
func (b *RouteBuilder[Req, Res]) Use(middlewares ...func(http.HandlerFunc) http.HandlerFunc) *RouteBuilder[Req, Res] {
	b.middlewares = append(b.middlewares, middlewares...)
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
	return b
}

func (b *RouteBuilder[Req, Res]) OnSuccess(status int, desc string, ct ...string) *RouteBuilder[Req, Res] {
	b.addResponse(status, desc, ct, false)
	return b
}

func (b *RouteBuilder[Req, Res]) OnNoContent(status int, desc string) *RouteBuilder[Req, Res] {
	b.responses = append(b.responses, ResponseSpec{Status: status, Description: desc, IsError: false})
	return b
}

// PossibleErr привязывает зарегистрированные в Spec ошибки к роуту.
// Статусы должны быть предварительно зарегистрированы через Spec.AddError или Spec.AddErrorSchema.
// Рекомендуется использовать константы из пакета net/http (http.StatusBadRequest и т.д.).
func (b *RouteBuilder[Req, Res]) PossibleErr(statuses ...int) *RouteBuilder[Req, Res] {
	for _, status := range statuses {
		b.errorStatuses = append(b.errorStatuses, status)
		b.addResponse(status, "Error", []string{CTProblemJSON}, true)
	}
	return b
}

// copyMap — shallow copy map[int]string.
func copyMap(m map[int]string) map[int]string {
	if m == nil {
		return nil
	}
	c := make(map[int]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// copyAnyMap — shallow copy map[string]any.
func copyAnyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	c := make(map[string]any, len(m))
	maps.Copy(c, m)
	return c
}

// copyResponseMap — shallow copy map[int]*oa.Response.
func copyResponseMap(m map[int]*oa.Response) map[int]*oa.Response {
	if m == nil {
		return nil
	}
	c := make(map[int]*oa.Response, len(m))
	maps.Copy(c, m)
	return c
}

func (b *RouteBuilder[Req, Res]) addResponse(status int, desc string, ct []string, isError bool) {
	if len(ct) == 0 {
		ct = []string{CTJSON}
	}
	b.responses = append(b.responses, ResponseSpec{
		Status: status, Description: desc, ContentTypes: ct, IsError: isError,
	})
}

func (b *RouteBuilder[Req, Res]) Register(mux *http.ServeMux) *RouteBuilder[Req, Res] {
	if b.handler == nil {
		panic("nooa: handler cannot be nil")
	}
	if b.path == "" {
		panic("nooa: path cannot be empty")
	}
	handled := b.handler
	// Оборачиваем handler middleware-ами в обратном порядке,
	// чтобы первый добавленный middleware был самым внешним.
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		handled = b.middlewares[i](handled)
	}
	mux.HandleFunc(b.method+" "+b.path, handled)
	b.registerGlobal()
	return b
}

// Spec возвращает глубокую копию RouteSpec, собранную из полей билдера.
// Собирается лениво, при первом вызове.
func (b *RouteBuilder[Req, Res]) Spec() RouteSpec {
	return RouteSpec{
		Method:                b.method,
		Path:                  b.path,
		OperationID:           b.operationID,
		Summary:               b.summary,
		Description:           b.description,
		Tags:                  append([]string(nil), b.tags...),
		Deprecated:            b.deprecated,
		Security:              append([]SecurityRequirement(nil), b.security...),
		RequestContentType:    append([]string(nil), b.requestContentType...),
		Responses:             append([]ResponseSpec(nil), b.responses...),
		Handler:               b.handler,
		Middlewares:           append([]func(http.HandlerFunc) http.HandlerFunc(nil), b.middlewares...),
		RequestBodySchemaName: b.requestBodySchemaName,
		ResponseSchemaNames:   copyMap(b.responseSchemaNames),
		ErrorStatuses:         append([]int(nil), b.errorStatuses...),
		Parameters:            append([]*oa.Parameter(nil), b.parameters...),
		Extensions:            copyAnyMap(b.extensions),
		ModelResponses:        copyResponseMap(b.modelResponses),
	}
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
	return b
}

func getSchemaName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Pointer {
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

func (b *RouteBuilder[Req, Res]) RegisterSpec(spec *Spec) *RouteBuilder[Req, Res] {
	if spec != nil {
		reqName := getSchemaName[Req]()
		resName := getSchemaName[Res]()
		spec.RegisterModel(reqName, new(Req))
		if reqName != resName {
			spec.RegisterModel(resName, new(Res))
		}
		spec.AddRoute(b.Spec())
	}
	return b
}

// RegisterSpecAndMux привязывает к Spec И регистрирует хендлер в mux.
// Если Spec задан, middleware из Spec применяются как самые внешние.
func (b *RouteBuilder[Req, Res]) RegisterSpecAndMux(mux *http.ServeMux, spec *Spec) *RouteBuilder[Req, Res] {
	if spec != nil {
		spec.RegisterModel(getSchemaName[Req](), new(Req))
		resName := getSchemaName[Res]()
		reqName := getSchemaName[Req]()
		if reqName != resName {
			spec.RegisterModel(resName, new(Res))
		}
		spec.RegisterMux(mux, b.Spec())
	} else {
		b.Register(mux)
	}
	return b
}

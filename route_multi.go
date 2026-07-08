package nooa

import (
	"net/http"
	"reflect"
	"strings"

	oa "github.com/arkannsk/elval/pkg/openapi"
)

// ResponseEntry — один тип ответа для NewRouteMultiResp.
// Status — HTTP status code, Instance — нулевой инстанс модели (для регистрации схемы).
type ResponseEntry struct {
	Status       int
	Instance     any
	Desc         string
	ContentTypes []string
}

// NewRouteMultiResp создаёт роут с несколькими типами ответов.
// Каждый entry регистрирует свою схему и привязывает её к указанному статусу.
//
//	NewRouteMultiResp("POST", "/items", handler,
//		ResponseEntry{Status: 201, Instance: new(CreatedItem), Desc: "Created"},
//		ResponseEntry{Status: 202, Instance: new(AcceptedJob), Desc: "Processing"},
//	).RegisterSpecAndMux(mux, spec)
func NewRouteMultiResp[Req any](method, path string, handler http.HandlerFunc, responses ...ResponseEntry) *RouteBuilderMultiResp {
	b := &RouteBuilderMultiResp{
		method:              strings.ToUpper(method),
		path:                path,
		handler:             handler,
		operationID:         defaultOperationID(method, path),
		requestContentType:  []string{CTJSON},
		responseSchemaNames: make(map[int]string),
		modelResponses:      make(map[int]*oa.Response),
		responses:           make([]ResponseSpec, 0, len(responses)),
	}

	// Регистрируем request модель
	reqSchemaName := getSchemaName[Req]()
	reqInstance := new(Req)
	RegisterModel(reqSchemaName, reqInstance)
	collectParams(reqInstance, &b.parameters)
	collectResponses(reqInstance, b.modelResponses)

	if method != "GET" && method != "HEAD" && method != "DELETE" {
		b.requestBodySchemaName = reqSchemaName
	}

	// Регистрируем каждую модель ответа
	for _, resp := range responses {
		inst := resp.Instance
		if inst == nil {
			continue
		}
		schemaName := multiRespSchemaName(inst)
		RegisterModel(schemaName, inst)
		collectParams(inst, &b.parameters)
		collectResponses(inst, b.modelResponses)

		b.responseSchemaNames[resp.Status] = schemaName
		ct := resp.ContentTypes
		if len(ct) == 0 {
			ct = []string{CTJSON}
		}
		desc := resp.Desc
		if desc == "" {
			desc = http.StatusText(resp.Status)
		}
		b.responses = append(b.responses, ResponseSpec{
			Status: resp.Status, Description: desc, ContentTypes: ct, IsError: false,
		})
	}

	return b
}

// RouteBuilderMultiResp — билдер для роутов с несколькими типами ответов.
// API идентичен RouteBuilder, но ответные модели задаются через ResponseEntry.
type RouteBuilderMultiResp struct {
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

func (b *RouteBuilderMultiResp) Summary(s string) *RouteBuilderMultiResp {
	b.summary = s
	return b
}

func (b *RouteBuilderMultiResp) Description(s string) *RouteBuilderMultiResp {
	b.description = s
	return b
}

func (b *RouteBuilderMultiResp) Tags(tags ...string) *RouteBuilderMultiResp {
	b.tags = append(b.tags, tags...)
	return b
}

func (b *RouteBuilderMultiResp) OperationID(id string) *RouteBuilderMultiResp {
	b.operationID = id
	return b
}

func (b *RouteBuilderMultiResp) Deprecated() *RouteBuilderMultiResp {
	b.deprecated = true
	return b
}

func (b *RouteBuilderMultiResp) Secure(scheme string, scopes ...string) *RouteBuilderMultiResp {
	b.security = append(b.security, SecurityRequirement{Scheme: scheme, Scopes: scopes})
	return b
}

func (b *RouteBuilderMultiResp) RequestContentType(cts ...string) *RouteBuilderMultiResp {
	if len(cts) > 0 {
		b.requestContentType = cts
	}
	return b
}

func (b *RouteBuilderMultiResp) Extension(key string, value any) *RouteBuilderMultiResp {
	if b.extensions == nil {
		b.extensions = make(map[string]any)
	}
	b.extensions[key] = value
	return b
}

func (b *RouteBuilderMultiResp) RequestBodySchema(name string) *RouteBuilderMultiResp {
	b.requestBodySchemaName = name
	return b
}

func (b *RouteBuilderMultiResp) ResponseSchema(status int, schemaName string) *RouteBuilderMultiResp {
	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[status] = schemaName
	return b
}

// Response добавляет произвольный ответ для указанного status code.
// Схема берётся из зарегистрированных моделей (spec.RegisterModel).
func (b *RouteBuilderMultiResp) Response(status int, schemaName string, desc string, ct ...string) *RouteBuilderMultiResp {
	if b.responseSchemaNames == nil {
		b.responseSchemaNames = make(map[int]string)
	}
	b.responseSchemaNames[status] = schemaName
	b.addResponse(status, desc, ct, false)
	return b
}

// Use добавляет middleware к роуту.
func (b *RouteBuilderMultiResp) Use(middlewares ...func(http.HandlerFunc) http.HandlerFunc) *RouteBuilderMultiResp {
	b.middlewares = append(b.middlewares, middlewares...)
	return b
}

// Prefix добавляет префикс к пути (например, /api/v1)
func (b *RouteBuilderMultiResp) Prefix(prefix string) *RouteBuilderMultiResp {
	prefix = strings.TrimRight(prefix, "/")
	currentPath := strings.TrimLeft(b.path, "/")

	if prefix != "" && currentPath != "" {
		b.path = prefix + "/" + currentPath
	} else if prefix != "" {
		b.path = prefix
	}
	return b
}

func (b *RouteBuilderMultiResp) OnSuccess(status int, desc string, ct ...string) *RouteBuilderMultiResp {
	b.addResponse(status, desc, ct, false)
	return b
}

func (b *RouteBuilderMultiResp) OnNoContent(status int, desc string) *RouteBuilderMultiResp {
	b.responses = append(b.responses, ResponseSpec{Status: status, Description: desc, IsError: false})
	return b
}

// PossibleErr привязывает зарегистрированные в Spec ошибки к роуту.
func (b *RouteBuilderMultiResp) PossibleErr(statuses ...int) *RouteBuilderMultiResp {
	for _, status := range statuses {
		b.errorStatuses = append(b.errorStatuses, status)
		b.addResponse(status, "Error", []string{CTProblemJSON}, true)
	}
	return b
}

func (b *RouteBuilderMultiResp) addResponse(status int, desc string, ct []string, isError bool) {
	if len(ct) == 0 {
		ct = []string{CTJSON}
	}
	b.responses = append(b.responses, ResponseSpec{
		Status: status, Description: desc, ContentTypes: ct, IsError: isError,
	})
}

func (b *RouteBuilderMultiResp) Register(mux *http.ServeMux) *RouteBuilderMultiResp {
	if b.handler == nil {
		panic("nooa: handler cannot be nil")
	}
	if b.path == "" {
		panic("nooa: path cannot be empty")
	}
	handled := b.handler
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		handled = b.middlewares[i](handled)
	}
	mux.HandleFunc(b.method+" "+b.path, handled)
	b.registerGlobal()
	return b
}

func (b *RouteBuilderMultiResp) Spec() RouteSpec {
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

func (b *RouteBuilderMultiResp) registerGlobal() *RouteBuilderMultiResp {
	addToRegistryInternal(b.Spec())
	return b
}

func (b *RouteBuilderMultiResp) RegisterSpec(spec *Spec) *RouteBuilderMultiResp {
	if spec != nil {
		spec.AddRoute(b.Spec())
	}
	return b
}

func (b *RouteBuilderMultiResp) RegisterSpecAndMux(mux *http.ServeMux, spec *Spec) *RouteBuilderMultiResp {
	if spec != nil {
		spec.RegisterMux(mux, b.Spec())
	} else {
		b.Register(mux)
	}
	return b
}

// multiRespSchemaName вычисляет имя схемы для инстанса модели в NewRouteMultiResp.
func multiRespSchemaName(instance any) string {
	typ := reflect.TypeOf(instance)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	pkgPath := typ.PkgPath()
	typeName := typ.Name()

	if pkgPath == "" || pkgPath == "main" {
		return typeName
	}

	pkgName := pkgPath[strings.LastIndex(pkgPath, "/")+1:]
	return pkgName + "." + typeName
}

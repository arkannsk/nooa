package nooa

import (
	"encoding/json"
	"net/http"
	"path"
	"reflect"
	"sync"

	oa "github.com/arkannsk/elval/pkg/openapi"
)

// errorSchema — внутреннее представление зарегистрированной ошибки.
type errorSchema struct {
	schemaName  string // имя схемы в components/schemas
	description string // описание ошибки
}

// Spec — изолированный генератор OpenAPI спецификации для одной версии/группы.
type Spec struct {
	routes          []RouteSpec
	schemas         map[string]*oa.Schema
	errors          map[int]*errorSchema
	tags            map[string]string // name -> description
	securitySchemes []SecurityScheme
	defaultSecurity []SecurityRequirement
	info            Info
	transformers    []SpecTransformer
	mu              sync.RWMutex
	specJSON        []byte
	generated       bool
}

// NewSpec создает новый изолированный генератор спецификации.
func NewSpec(info Info) *Spec {
	return &Spec{
		schemas:         make(map[string]*oa.Schema),
		errors:          make(map[int]*errorSchema),
		tags:            make(map[string]string),
		securitySchemes: nil,
		defaultSecurity: nil,
		info:            info,
	}
}

// RegisterModel регистрирует модель в рамках этой спецификации.
func (s *Spec) RegisterModel(name string, instance any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	registerModelInternal(s.schemas, name, instance)
	s.generated = false
}

// AddRoute добавляет маршрут в спецификацию.
func (s *Spec) AddRoute(r RouteSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes = append(s.routes, r)
	s.generated = false // инвалидируем кэш
}

// ServeHTTP отдаёт сгенерированный JSON. Можно монтировать как http.Handler.
func (s *Spec) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := s.generate()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(data)
}

// generate строит JSON один раз при первом запросе (thread-safe).
func (s *Spec) generate() []byte {
	if !s.generated {
		specMap := buildSpecFromData(s.info, s.routes, s.schemas, s.errors, s.tags, s.securitySchemes, s.defaultSecurity)
		for _, t := range s.transformers {
			specMap = t(specMap)
		}
		data, err := json.MarshalIndent(specMap, "", "  ")
		if err != nil {
			data = []byte(`{"openapi":"3.0.3","info":{"title":"Error","version":"0.0"}}`)
		}
		s.mu.Lock()
		s.specJSON = data
		s.generated = true
		s.mu.Unlock()
	}
	return s.specJSON
}

// SetTransformers добавляет трансформеры спецификации.
func (s *Spec) SetTransformers(transformers ...SpecTransformer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transformers = transformers
	s.generated = false
}

// AddError регистрирует ошибку на уровне спецификации.
// Модель автоматически добавляется в schemas; имя схемы вычисляется из типа.
// Статус коды рекомендуется брать из констант net/http (http.StatusBadRequest и т.д.).
// После регистрации можно привязать ошибку к роуту через Route.PossibleErr(status...).
//
//	spec.AddError(http.StatusBadRequest, new(models.ValidationError), "Validation failed")
func (s *Spec) AddError(status int, model any, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Определяем имя схемы из типа модели
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	pkgPath := t.PkgPath()
	typeName := t.Name()

	var schemaName string
	if pkgPath == "" || pkgPath == "main" {
		schemaName = typeName
	} else {
		schemaName = path.Base(pkgPath) + "." + typeName
	}

	// Регистрируем модель в схемах
	registerModelInternal(s.schemas, schemaName, model)

	s.errors[status] = &errorSchema{
		schemaName:  schemaName,
		description: description,
	}
	s.generated = false
}

// LookupError возвращает зарегистрированную ошибку по статусу.
func (s *Spec) LookupError(status int) *errorSchema {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.errors[status]
}

// AddTag регистрирует тег с описанием.
// Теги используются для группировки маршрутов в OpenAPI спецификации.
// Описание отображается в Swagger/Redoc/Scalar UI.
//
//	spec.AddTag("Users", "Операции с пользователями")
//	spec.AddTag("Auth", "Аутентификация и авторизация")
func (s *Spec) AddTag(name, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tags[name] = description
	s.generated = false
}

// GetTags возвращает копию зарегистрированных тегов.
func (s *Spec) GetTags() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make(map[string]string, len(s.tags))
	for k, v := range s.tags {
		copy[k] = v
	}
	return copy
}

// AddSecurityScheme регистрирует схему безопасности в components/securitySchemes.
// Схема становится доступна для использования в Route.Secure().
//
// Примеры:
//
//	// Bearer token (JWT)
//	spec.AddSecurityScheme("bearerAuth", nooa.SecuritySchemeBearer("JWT authorization"))
//
//	// API key в заголовке
//	spec.AddSecurityScheme("apiKey", nooa.SecuritySchemeAPIKey("X-API-Key", "header", "API key"))
//
//	// Basic auth
//	spec.AddSecurityScheme("basicAuth", nooa.SecuritySchemeBasic("HTTP Basic auth"))
func (s *Spec) AddSecurityScheme(name string, scheme SecurityScheme) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Если имя не задано, используем нормализованное имя из схемы
	if name == "" {
		name = scheme.Name
	}
	scheme.Name = name
	s.securitySchemes = append(s.securitySchemes, scheme)
	s.generated = false
}

// DefaultSecurity задаёт глобальные требования безопасности для всех операций.
// Если маршрут не указывает своё Security через Route.Secure(), используется этот блок.
// Для отмены глобального security на конкретном маршруте — передайте пустой список.
//
// Пример:
//
//	spec.DefaultSecurity(noa.SecurityRequirement{Scheme: "bearerAuth", Scopes: []string{"read", "write"}})
func (s *Spec) DefaultSecurity(reqs ...SecurityRequirement) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultSecurity = append([]SecurityRequirement(nil), reqs...)
	s.generated = false
}

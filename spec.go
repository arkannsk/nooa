package nooa

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/arkannsk/elval/pkg/oa"
)

// Spec — изолированный генератор OpenAPI спецификации для одной версии/группы.
type Spec struct {
	routes       []RouteSpec
	schemas      map[string]*oa.Schema
	info         Info
	transformers []SpecTransformer
	mu           sync.RWMutex
	specJSON     []byte
	generated    bool
}

// NewSpec создает новый изолированный генератор спецификации.
func NewSpec(info Info) *Spec {
	return &Spec{
		schemas: make(map[string]*oa.Schema),
		info:    info,
	}
}

// RegisterModel регистрирует модель в рамках этой спецификации.
func (s *Spec) RegisterModel(name string, instance any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	registerModelInternal(s.schemas, name, instance)
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
		specMap := buildSpecFromData(s.info, s.routes, s.schemas)
		for _, t := range s.transformers {
			specMap = t(specMap)
		}
		data, err := json.MarshalIndent(specMap, "", "  ")
		if err != nil {
			log.Printf("❌ Error generating spec for %s: %v", s.info.Title, err)
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

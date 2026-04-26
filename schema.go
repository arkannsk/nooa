package nooa

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"

	"github.com/arkannsk/elval/pkg/oa"
)

// RegisterModel регистрирует модель, используя рефлексию для поиска метода OaSchema.
// Это позволяет работать со структурами, у которых метод возвращает конкретный тип (*oa.Schema),
// а не интерфейс any.
// modelSchemas хранит схемы напрямую в типизированном виде
var modelSchemas = make(map[string]*oa.Schema)

// RegisterModel регистрирует модель, вызывая её метод OaSchema().
// Работает напрямую с *oa.Schema. Если по какой-то причине возвращается не *oa.Schema,
// используется фоллбэк через json.Unmarshal (как вы просили).
func RegisterModel(name string, instance any) {
	// Корректно извлекаем тип для fallback-имени
	t := reflect.TypeOf(instance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Подготовка для вызова OaSchema()
	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var method reflect.Value
	if m := val.MethodByName("OaSchema"); m.IsValid() {
		method = m
	} else if val.CanAddr() {
		if m := val.Addr().MethodByName("OaSchema"); m.IsValid() {
			method = m
		}
	} else {
		ptr := reflect.New(val.Type())
		ptr.Elem().Set(val)
		if m := ptr.MethodByName("OaSchema"); m.IsValid() {
			method = m
		}
	}

	if !method.IsValid() {
		log.Printf("⚠️ OaSchema not found on %T", instance)
		return
	}

	results := method.Call(nil)
	if len(results) == 0 || results[0].IsNil() {
		log.Printf("❌ OaSchema returned nil for %T", instance)
		return
	}

	// Извлекаем *oa.Schema (прямой путь или fallback)
	var schema *oa.Schema
	if s, ok := results[0].Interface().(*oa.Schema); ok {
		schema = s
	} else {
		// Fallback через JSON
		data, _ := json.Marshal(results[0].Interface())
		schema = &oa.Schema{}
		json.Unmarshal(data, schema)
		log.Printf("OaSchema fallback used for %T", instance)
	}

	//  Проверка на коллизию ключей
	if _, exists := modelSchemas[name]; exists {
		fullRef := t.PkgPath() + "." + t.Name()
		name = sanitizeGlobalRefForOpenAPI(fullRef)
		log.Printf("Collision resolved: %s → %s", name, name)
	}

	// 5. Очищаем $ref перед сохранением в компоненты
	schema.Ref = ""
	modelSchemas[name] = schema
}

// GetRegisteredSchemas возвращает все зарегистрированные схемы
func GetRegisteredSchemas() map[string]*oa.Schema {
	return modelSchemas
}

func sanitizeGlobalRefForOpenAPI(ref string) string {
	// Заменяем слеши на точки, убираем ведущие точки/пустоты
	s := strings.ReplaceAll(ref, "/", ".")
	s = strings.TrimLeft(s, ".")
	return s
}

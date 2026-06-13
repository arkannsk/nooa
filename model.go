package nooa

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"

	oa "github.com/arkannsk/elval/pkg/openapi"
)

// Глобальная мапа для обратной совместимости со старым API (SpecMiddleware)
var modelSchemas = make(map[string]*oa.Schema)

// registerModelInternal — универсальная логика регистрации схемы в переданную мапу.
func registerModelInternal(schemas map[string]*oa.Schema, name string, instance any) {
	t := reflect.TypeOf(instance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

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

	var schema *oa.Schema
	if s, ok := results[0].Interface().(*oa.Schema); ok {
		schema = s
	} else {
		data, _ := json.Marshal(results[0].Interface())
		schema = &oa.Schema{}
		json.Unmarshal(data, schema)
	}

	// Проверка на коллизию ключей внутри конкретной мапы схем
	if _, exists := schemas[name]; exists {
		fullRef := t.PkgPath() + "." + t.Name()
		name = sanitizeGlobalRefForOpenAPI(fullRef)
	}

	schema.Ref = "" // Очищаем $ref для компонентов
	schemas[name] = schema
}

// RegisterModel (глобальная функция) для обратной совместимости
func RegisterModel(name string, instance any) {
	registerModelInternal(modelSchemas, name, instance)
}

func sanitizeGlobalRefForOpenAPI(ref string) string {
	s := strings.ReplaceAll(ref, "/", ".")
	s = strings.TrimLeft(s, ".")
	return s
}

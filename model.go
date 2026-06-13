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
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Pointer {
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

	var schema *oa.Schema

	if !method.IsValid() {
		log.Printf("⚠️ OaSchema not found on %T, falling back to reflection", instance)
		schema = schemaFromType(t)
		if schema == nil {
			return
		}
	} else {
		results := method.Call(nil)
		if len(results) == 0 || results[0].IsNil() {
			log.Printf("❌ OaSchema returned nil for %T", instance)
			return
		}

		if s, ok := results[0].Interface().(*oa.Schema); ok {
			schema = s
		} else {
			data, _ := json.Marshal(results[0].Interface())
			schema = &oa.Schema{}
			json.Unmarshal(data, schema)
		}
	}

	// Если схема с таким именем уже зарегистрирована — пропускаем.
	// Это предотвращает дубликаты при повторной регистрации (например, когда
	// RegisterSpecAndMux и RegisterSpec вызываются для одного роута).
	if _, exists := schemas[name]; exists {
		return
	}

	schema.Ref = "" // Очищаем $ref для компонентов
	schemas[name] = schema
}

// schemaFromType генерирует базовую OpenAPI-схему через рефлексию.
// Используется как fallback, когда у модели нет метода OaSchema().
func schemaFromType(t reflect.Type) *oa.Schema {
	if t.Kind() == reflect.Ptr {
		return schemaFromType(t.Elem())
	}

	switch t.Kind() {
	case reflect.String:
		return &oa.Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &oa.Schema{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &oa.Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &oa.Schema{Type: "number"}
	case reflect.Bool:
		return &oa.Schema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		return &oa.Schema{
			Type:  "array",
			Items: schemaFromType(t.Elem()),
		}
	case reflect.Map:
		// oa.Schema не поддерживает AdditionalProperties, возвращаем object
		return &oa.Schema{Type: "object"}
	case reflect.Struct:
		schema := &oa.Schema{
			Type:       "object",
			Properties: make(map[string]*oa.Schema),
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = field.Name
			} else {
				jsonTag = strings.Split(jsonTag, ",")[0]
			}
			if jsonTag == "-" {
				continue
			}

			schema.Properties[jsonTag] = schemaFromType(field.Type)
		}
		return schema
	default:
		log.Printf("⚠️ Unsupported type %v for schema generation", t.Kind())
		return nil
	}
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

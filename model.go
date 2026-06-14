package nooa

import (
	"reflect"
	"strings"

	oa "github.com/arkannsk/elval/pkg/openapi"
)

// globalSchemas — глобальный реестр схем для обратной совместимости.
var globalSchemas = make(map[string]*oa.Schema)

// schemaProvider — интерфейс, реализуемый типами сгенерированными elval-gen.
type schemaProvider interface {
	OaSchema() *oa.Schema
	GlobalRef() string
}

// RegisterModel регистрирует модель в глобальном реестре схем.
// Эта функция используется в NewRoute для автоматической регистрации request/response моделей.
func RegisterModel(name string, instance any) {
	registerModelInternal(globalSchemas, name, instance)
}

// registerModelInternal регистрирует модель в указанной карте схем.
// Если instance реализует schemaProvider, вызывается OaSchema() и результат сохраняется.
// Затем рекурсивно регистрируются вложенные типы из полей структуры.
func registerModelInternal(schemas map[string]*oa.Schema, name string, instance any) {
	// Проверяем, реализует ли instance интерфейс schemaProvider
	sp, ok := any(instance).(schemaProvider)
	if !ok {
		return
	}

	schema := sp.OaSchema()
	if schema == nil {
		return
	}

	// Если схема уже зарегистрирована — пропускаем
	if _, exists := schemas[name]; exists {
		return
	}

	if !registerSchema(schemas, name, schema) {
		return
	}

	// Рекурсивно регистрируем вложенные типы
	registerNestedTypes(schemas, reflect.TypeOf(instance), name)
}

// registerNestedTypes рекурсивно находит и регистрирует вложенные типы из полей структуры.
func registerNestedTypes(schemas map[string]*oa.Schema, typ reflect.Type, parentName string) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		ft := f.Type

		// Пропускаем вложенные (неэкспортированные) поля
		if !f.IsExported() {
			continue
		}

		walkType(schemas, ft, parentName)
	}
}

// walkType обходит тип поля и регистрирует вложенные структуры.
// Если тип — generic struct (например Option[T]), рекурсивно обходит его поля
// чтобы обнаружить скрытые schemaProvider'ы в конкретизированных type arguments.
func walkType(schemas map[string]*oa.Schema, typ reflect.Type, parentName string) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		// Проверяет, реализует ли этот тип schemaProvider
		var zeroVal any = reflect.New(typ).Interface()
		sp, isSP := zeroVal.(schemaProvider)
		if isSP {
			ref := sp.GlobalRef()
			shortName := extractShortName(ref)
			schema := sp.OaSchema()
			if registerSchema(schemas, shortName, schema) {
				registerNestedTypes(schemas, typ, shortName)
			}
		}

		// Если тип не реализует schemaProvider (например generic обёртка Option[T]),
		// обходим его поля, чтобы обнаружить вложенные schemaProvider'ы.
		// Это позволяет автоматически регистрировать типы, скрытые внутри generics.
		if !isSP {
			walkStructFields(schemas, typ, parentName)
		}

	case reflect.Slice, reflect.Array:
		elem := typ.Elem()
		// Сначала рекурсивно обходим элемент
		walkType(schemas, elem, parentName)

		// Затем регистрируем элемент, если он struct с OaSchema
		unwrapped := elem
		if unwrapped.Kind() == reflect.Ptr {
			unwrapped = unwrapped.Elem()
		}
		if unwrapped.Kind() == reflect.Struct {
			zeroVal := reflect.New(unwrapped).Interface()
			sp, isSP := zeroVal.(schemaProvider)
			if isSP {
				ref := sp.GlobalRef()
				shortName := extractShortName(ref)
				schema := sp.OaSchema()
				if registerSchema(schemas, shortName, schema) {
					registerNestedTypes(schemas, unwrapped, shortName)
				}
			}
			// Если элемент — generic обёртка без schemaProvider, обходим его поля
			if !isSP {
				walkStructFields(schemas, unwrapped, parentName)
			}
		}

	case reflect.Map:
		val := typ.Elem()
		// Рекурсивно обходим значение
		walkType(schemas, val, parentName)

		// Регистрируем значение, если оно struct с OaSchema
		unwrapped := val
		if unwrapped.Kind() == reflect.Ptr {
			unwrapped = unwrapped.Elem()
		}
		if unwrapped.Kind() == reflect.Struct {
			zeroVal := reflect.New(unwrapped).Interface()
			sp, isSP := zeroVal.(schemaProvider)
			if isSP {
				ref := sp.GlobalRef()
				shortName := extractShortName(ref)
				schema := sp.OaSchema()
				if registerSchema(schemas, shortName, schema) {
					registerNestedTypes(schemas, unwrapped, shortName)
				}
			}
			// Если значение — generic обёртка без schemaProvider, обходим его поля
			if !isSP {
				walkStructFields(schemas, unwrapped, parentName)
			}
		}
	}
}

// walkStructFields рекурсивно обходит поля структуры, вызывая walkType для каждого.
// В отличие от registerNestedTypes, обходит все поля (включая неэкспортированные),
// потому что в generic обёртках тип-аргументы часто хранятся в неэкспортированных полях
// (например value T в Option[T]).
func walkStructFields(schemas map[string]*oa.Schema, typ reflect.Type, parentName string) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		walkType(schemas, f.Type, parentName)
	}
}

// registerSchema регистрирует схему, очищая Ref, если она ещё не зарегистрирована.
// Возвращает true если регистрация произошла, false если схема уже существовала.
func registerSchema(schemas map[string]*oa.Schema, name string, schema *oa.Schema) bool {
	if _, exists := schemas[name]; exists {
		return false
	}
	schema.Ref = ""
	schemas[name] = schema
	return true
}

// extractShortName извлекает короткое имя из полного $ref.
// github.com/arkannsk/nooa/examples/models/03_nested.Address -> 03_nested.Address
func extractShortName(ref string) string {
	// Убираем префикс $ref
	pathPart := strings.TrimPrefix(ref, "#/components/schemas/")

	// Заменяем / на .
	dotted := strings.ReplaceAll(pathPart, "/", ".")

	// Берём последние два сегмента (пакет.Тип)
	parts := strings.Split(dotted, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}
	return pathPart
}

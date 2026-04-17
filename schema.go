package nooa

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
)

// modelSchemas хранит схемы.
var modelSchemas = make(map[string]any)

// RegisterModel регистрирует модель, используя рефлексию для поиска метода OaSchema.
// Это позволяет работать со структурами, у которых метод возвращает конкретный тип (*oa.Schema),
// а не интерфейс any.
func RegisterModel[T any](name string) {
	var zero T
	val := reflect.ValueOf(&zero)

	// Ищем метод OaSchema
	method := val.MethodByName("OaSchema")
	if !method.IsValid() {
		log.Printf("⚠️  Type %T does not have a method named 'OaSchema'. Did you run 'go generate'?", zero)
		return
	}

	// Вызываем метод (он не принимает аргументов)
	results := method.Call(nil)
	if len(results) == 0 {
		log.Printf("❌ Method OaSchema for %T returned no values", zero)
		return
	}

	schemaObj := results[0].Interface()
	if schemaObj == nil {
		log.Printf("️  OaSchema() returned nil for %T", zero)
		return
	}

	// Конвертируем результат в map[string]any через JSON
	data, err := json.Marshal(schemaObj)
	if err != nil {
		log.Printf("❌ Failed to marshal schema for %T: %v", zero, err)
		// Попробуем вывести тип, чтобы понять, что пришло
		log.Printf("   Schema type received: %T", schemaObj)
		return
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(data, &schemaMap); err != nil {
		log.Printf("❌ Failed to unmarshal schema for %T: %v", zero, err)
		return
	}

	if len(schemaMap) == 0 {
		log.Printf("️  Resulting schema map is empty for %T. Check if the struct has json tags.", zero)
	} else {
		modelSchemas[name] = schemaMap
		fmt.Printf("✅ Registered schema for '%s' (%d properties)\n", name, len(schemaMap))
		if props, ok := schemaMap["properties"].(map[string]any); ok {
			fmt.Printf("   Fields: %v\n", reflect.ValueOf(props).MapKeys())
		}
	}
}

func GetRegisteredSchemas() map[string]any {
	result := make(map[string]any, len(modelSchemas))
	for k, v := range modelSchemas {
		result[k] = v
	}
	return result
}

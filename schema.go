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

	// 1. Поиск метода
	method := val.MethodByName("OaSchema")
	if !method.IsValid() {
		log.Printf("❌ FAIL: Method OaSchema not found on %T", zero)
		return
	}

	// 2. Вызов метода
	results := method.Call(nil)
	if len(results) == 0 {
		log.Printf("❌ FAIL: OaSchema returned no values")
		return
	}

	schemaObj := results[0].Interface()
	if schemaObj == nil {
		log.Printf(" FAIL: OaSchema returned nil")
		return
	}

	// 3. Маршалинг
	data, err := json.Marshal(schemaObj)
	if err != nil {
		log.Printf("❌ FAIL: Marshal error: %v", err)
		// Попробуем вывести тип ошибки подробнее
		fmt.Printf("   Schema object content: %+v\n", schemaObj)
		return
	}

	// 4. Размаршалинг в map
	var schemaMap map[string]any
	if err := json.Unmarshal(data, &schemaMap); err != nil {
		log.Printf("❌ FAIL: Unmarshal error: %v", err)
		return
	}

	if len(schemaMap) == 0 {
		return
	}

	// 5. Сохранение в глобальную мапу
	modelSchemas[name] = schemaMap
}

func GetRegisteredSchemas() map[string]any {
	result := make(map[string]any, len(modelSchemas))
	for k, v := range modelSchemas {
		result[k] = v
	}

	return result
}

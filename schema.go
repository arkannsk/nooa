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
	fmt.Printf(" RegisterModel called for: %s\n", name)

	var zero T
	val := reflect.ValueOf(&zero)

	// 1. Поиск метода
	method := val.MethodByName("OaSchema")
	if !method.IsValid() {
		log.Printf("❌ FAIL: Method OaSchema not found on %T", zero)
		return
	}
	fmt.Println("   ✅ Method found")

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
	fmt.Printf("   ✅ OaSchema returned: %T\n", schemaObj)

	// 3. Маршалинг
	data, err := json.Marshal(schemaObj)
	if err != nil {
		log.Printf("❌ FAIL: Marshal error: %v", err)
		// Попробуем вывести тип ошибки подробнее
		fmt.Printf("   Schema object content: %+v\n", schemaObj)
		return
	}
	fmt.Println("   ✅ Marshaled successfully")

	// 4. Размаршалинг в map
	var schemaMap map[string]any
	if err := json.Unmarshal(data, &schemaMap); err != nil {
		log.Printf("❌ FAIL: Unmarshal error: %v", err)
		return
	}

	if len(schemaMap) == 0 {
		log.Printf("❌ FAIL: Resulting map is empty! Check if oa.Schema has json tags.")
		// Выведем сырой JSON для проверки
		fmt.Printf("   Raw JSON: %s\n", string(data))
		return
	}

	fmt.Printf("   ✅ Map created with %d keys\n", len(schemaMap))

	// 5. Сохранение в глобальную мапу
	modelSchemas[name] = schemaMap
	fmt.Printf("   🎉 SUCCESS: Registered '%s' into global registry\n", name)
	fmt.Printf("   Current registry size: %d\n", len(modelSchemas))
}

func GetRegisteredSchemas() map[string]any {
	fmt.Printf("🔍 GetRegisteredSchemas called. Global map size: %d\n", len(modelSchemas))

	result := make(map[string]any, len(modelSchemas))
	for k, v := range modelSchemas {
		result[k] = v
		fmt.Printf("   - Copying key: %s\n", k)
	}

	fmt.Printf("✅ Returning map with %d items\n", len(result))
	return result
}

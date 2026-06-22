# nooa проблемы (из vacuum-отчёта)

> Проблемы, связанные с библиотекой `nooa` — сборкой OpenAPI спецификации из схем elval-gen.

---

## 1. resolving-references — `$ref` на несуществующую схему

**Ошибка:** В `09_ignore` маршрут использует `$ref` на `09_ignore.WithIgnoredField`, но схема не зарегистрирована в `components/schemas`.

**Причина в nooa:** 
- `NewRoute` вызывает `RegisterModel(resSchemaName, resInstance)` 
- `registerModelInternal` проверяет `schemaProvider` интерфейс; если тип его не реализует (elval-gen не сгенерировал `OaSchema()`), регистрация молча пропускается
- `ResponseSchemaNames` всё равно получает имя `09_ignore.WithIgnoredField` 
- `buildResponseSchemaObject` генерирует `$ref` на несуществующую схему

**Фикс в `spec_builder.go`:** В `buildResponseSchemaObject` проверять, существует ли схема в `schemas` перед генерацией `$ref`. Если схемы нет — использовать fallback (`type: object`) или пропускать.

```go
func buildResponseSchemaObject(resp ResponseSpec, ct string, schemaName string, hasSchema bool, errorSchemas map[int]*errorSchema, schemas map[string]*oa.Schema) map[string]any {
    if hasSchema && schemaName != "" {
        if _, exists := schemas[schemaName]; exists {
            return map[string]any{"$ref": "#/components/schemas/" + schemaName}
        }
        // Схема не найдена — fallback
        log.Printf("nooa: schema %q not found, using fallback", schemaName)
        return map[string]any{"type": "object"}
    }
    // ... остальное без изменений
}
```

**Альтернативный фикс в `model.go`:** Если тип не реализует `schemaProvider`, автоматически генерировать базовую схему через рефлексию (обойти поля структуры).

**Файл:** `spec_builder.go` (строка 509), `model.go` (строка 40)

---

## 2. oas3-api-servers — trailing slash в server URL

**Предупреждения:** 13 нарушений (каждый пример имеет это предупреждение).

**Причина:** В `spec_builder.go` строка 42:
```go
"servers": []map[string]any{{"url": "/", "description": "Local server"}},
```

URL `/` имеет trailing slash. Согласно OpenAPI 3.0, server URL не должен заканчиваться слэшем.

**Фикс:** Заменить на пустую строку или использовать relative URL без слэша:
```go
"servers": []map[string]any{{"url": ".", "description": "Local server"}},
```
или
```go
"servers": []map[string]any{{"url": "", "description": "Local server"}},
```

**Файл:** `spec_builder.go`, строка 42

---

## 3. oas3-parameter-description — параметры без описания

**Предупреждения:** 15 нарушений, особенно в `08_http_params` (11).

**Причина:** `nooa` передаёт параметры из `OaParams()` моделей напрямую в OpenAPI spec, но не заполняет `description` для query/path параметров.

**Фикс:** 
- Добавить трансформер, который заполняет пустые `description` у параметров
- Или добавить валидацию в `buildOperationParameters` с warning-логированием

**Файл:** `spec_builder.go`, функция `buildOperationParameters` (строка 368)

---

## 4. description-duplication — дубликаты описаний

**Предупреждения:** 43 нарушения (info-уровень).

**Причина:** Когда response описывает схему через `$ref`, vacuum видит, что описание response совпадает с описанием поля внутри схемы. Например, `description` на уровне response и `description` свойства в `$ref`-схеме идентичны.

**Фикс:** 
- В `buildResponseDescription` использовать уникальные описания для response уровня, не копируя описания из полей схемы
- Это info-уровень — не критично, но улучшает качество спецификации

**Файл:** `spec_builder.go`, функция `buildResponseDescription` (строка 527)

---

## 5. Фоллбэк для типов без OaSchema

**Проблема:** Если elval-gen не генерирует `OaSchema()` для типа (например, из-за `@oa:ignore` на всех полях), `nooa` молча пропускает регистрацию, но продолжает использовать `$ref`.

**Фикс:** Добавить автоматическую генерацию схемы через рефлексию как fallback:

```go
// В registerModelInternal: если schemaProvider не реализован,
// попробовать создать базовую схему через рефлексию
func generateFallbackSchema(typ reflect.Type) *oa.Schema {
    schema := &oa.Schema{Type: "object", Properties: map[string]*oa.Schema{}}
    for i := range typ.NumField() {
        field := typ.Field(i)
        if !field.IsExported() {
            continue
        }
        // ... создать базовую схему для поля
    }
    return schema
}
```

**Файл:** `model.go`, функция `registerModelInternal`

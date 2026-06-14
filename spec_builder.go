package nooa

import (
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"

	oa "github.com/arkannsk/elval/pkg/openapi"
)

// Info содержит метаданные для OpenAPI спецификации.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// SpecTransformer — функция для модификации спецификации перед отдачей.
type SpecTransformer func(spec map[string]any) map[string]any

// normalizeSchema преобразует *oa.Schema в map[string]any с исправлениями:
//   - $ref изолируется (в OpenAPI $ref не может соседствовать с другими ключами)
//   - minimum/maximum на array преобразуются в minItems/maxItems
//   - рекурсивно нормализуются properties, items, oneOf, allOf, anyOf
//   - $ref значения ремапятся через refRemap (полные пути → короткие имена)
func normalizeSchema(schema *oa.Schema, refRemap map[string]string) map[string]any {
	if schema == nil {
		return nil
	}

	// Сериализуем через JSON чтобы получить все поля
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}

	raw := make(map[string]any)
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	// Если есть $ref — ремапим и возвращаем объект только с ним (OpenAPI правило)
	if ref, ok := raw["$ref"]; ok {
		if mapped, exists := refRemap[ref.(string)]; exists {
			return map[string]any{"$ref": mapped}
		}
		return map[string]any{"$ref": ref}
	}

	// Для array: minimum/maximum → minItems/maxItems
	if typ, ok := raw["type"]; ok && typ == "array" {
		if v, ok := raw["minimum"]; ok {
			raw["minItems"] = v
			delete(raw, "minimum")
		}
		if v, ok := raw["maximum"]; ok {
			raw["maxItems"] = v
			delete(raw, "maximum")
		}
	}

	// Рекурсивно нормализуем вложенные схемы
	if props, ok := raw["properties"].(map[string]any); ok {
		for key, val := range props {
			if propMap, ok := val.(map[string]any); ok {
				normalized := normalizeSchemaFromRaw(propMap, refRemap)
				if normalized != nil {
					props[key] = normalized
				}
			}
		}
	}

	if items, ok := raw["items"].(map[string]any); ok {
		raw["items"] = normalizeSchemaFromRaw(items, refRemap)
	}

	for _, key := range []string{"oneOf", "allOf", "anyOf"} {
		if schemasList, ok := raw[key].([]any); ok {
			for i, s := range schemasList {
				if sMap, ok := s.(map[string]any); ok {
					schemasList[i] = normalizeSchemaFromRaw(sMap, refRemap)
				}
			}
			raw[key] = schemasList
		}
	}

	return raw
}

// normalizeSchemaFromRaw — то же что normalizeSchema, но принимает уже распаршенную map.
// Используется для рекурсивной обработки вложенных схем.
func normalizeSchemaFromRaw(raw map[string]any, refRemap map[string]string) map[string]any {
	// Если есть $ref — ремапим и изолируем
	if _, ok := raw["$ref"]; ok {
		if ref, ok := raw["$ref"].(string); ok {
			if mapped, exists := refRemap[ref]; exists {
				return map[string]any{"$ref": mapped}
			}
			return map[string]any{"$ref": ref}
		}
		return map[string]any{"$ref": raw["$ref"]}
	}

	// Для array: minimum/maximum → minItems/maxItems
	if typ, ok := raw["type"]; ok && typ == "array" {
		if v, ok := raw["minimum"]; ok {
			raw["minItems"] = v
			delete(raw, "minimum")
		}
		if v, ok := raw["maximum"]; ok {
			raw["maxItems"] = v
			delete(raw, "maximum")
		}
	}

	// Рекурсивно нормализуем вложенные схемы
	if props, ok := raw["properties"].(map[string]any); ok {
		for key, val := range props {
			if propMap, ok := val.(map[string]any); ok {
				normalized := normalizeSchemaFromRaw(propMap, refRemap)
				if normalized != nil {
					props[key] = normalized
				}
			}
		}
	}

	if items, ok := raw["items"].(map[string]any); ok {
		raw["items"] = normalizeSchemaFromRaw(items, refRemap)
	}

	for _, key := range []string{"oneOf", "allOf", "anyOf"} {
		if schemasList, ok := raw[key].([]any); ok {
			for i, s := range schemasList {
				if sMap, ok := s.(map[string]any); ok {
					schemasList[i] = normalizeSchemaFromRaw(sMap, refRemap)
				}
			}
			raw[key] = schemasList
		}
	}

	return raw
}

// collectRefsFromSchema собирает все $ref значения из схемы (properties, items, oneOf/allOf/anyOf)
func collectRefsFromSchema(schema *oa.Schema, refs map[string]bool) {
	if schema == nil || refs == nil {
		return
	}

	if schema.Ref != "" {
		// Ref может быть как полным путём (github.com/...), так и уже содержать префикс $ref
		ref := schema.Ref
		if !strings.HasPrefix(ref, "#/") {
			ref = "#/components/schemas/" + ref
		}
		refs[ref] = true
	}

	for _, prop := range schema.Properties {
		collectRefsFromSchema(prop, refs)
	}
	if schema.Items != nil {
		collectRefsFromSchema(schema.Items, refs)
	}
	for _, s := range schema.OneOf {
		collectRefsFromSchema(s, refs)
	}
	for _, s := range schema.AllOf {
		collectRefsFromSchema(s, refs)
	}
	for _, s := range schema.AnyOf {
		collectRefsFromSchema(s, refs)
	}
}

// shortNameFromRef извлекает короткое имя из полного $ref.
// github.com/arkannsk/nooa/examples/models/03_nested.Address -> 03_nested.Address
func shortNameFromRef(ref string) string {
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

// generateRefRemap строит мапу для замены полных $ref на короткие имена.
// Проходит по всем схемам, собирает $ref из properties/items и сопоставляет их с ключами.
func generateRefRemap(schemas map[string]*oa.Schema) map[string]string {
	remap := make(map[string]string)

	// Собираем все $ref из свойств схем
	allRefs := make(map[string]bool)
	for _, schema := range schemas {
		collectRefsFromSchema(schema, allRefs)
	}

	// Для каждого $ref находим подходящий ключ
	for fullRef := range allRefs {
		shortName := shortNameFromRef(fullRef)
		if _, exists := schemas[shortName]; exists {
			remap[fullRef] = "#/components/schemas/" + shortName
		}
	}

	return remap
}

// buildSpecFromData собирает spec из переданных данных (без глобальных переменных)
func buildSpecFromData(info Info, routes []RouteSpec, schemas map[string]*oa.Schema, errorSchemas map[int]*errorSchema) map[string]any {
	// Строим remap для $ref (полные пути → короткие имена)
	refRemap := generateRefRemap(schemas)

	// Нормализуем схемы перед включением в спецификацию
	normalizedSchemas := make(map[string]any, len(schemas))
	for name, schema := range schemas {
		normalizedSchemas[name] = normalizeSchema(schema, refRemap)
	}

	// Собираем все уникальные теги из маршрутов
	allTagsMap := make(map[string]bool)
	tags := []map[string]any{}
	for _, r := range routes {
		for _, tag := range r.Tags {
			if tag != "" && !allTagsMap[tag] {
				allTagsMap[tag] = true
				tags = append(tags, map[string]any{
					"name":        tag,
					"description": "",
				})
			}
		}
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"servers": []map[string]any{{"url": "/", "description": "Current server"}},
		"paths":   map[string]any{},
		"components": map[string]any{
			"schemas": normalizedSchemas,
		},
		"tags": tags,
	}

	paths := spec["paths"].(map[string]any)
	pathParamRegex := regexp.MustCompile(`\{([^}]+)}`)

	for _, r := range routes {
		pathItem, ok := paths[r.Path].(map[string]any)
		if !ok {
			pathItem = map[string]any{}
			paths[r.Path] = pathItem
		}

		op := map[string]any{
			"operationId": r.OperationID,
			"summary":     r.Summary,
			"description": r.Description,
			"tags":        r.Tags,
			"deprecated":  r.Deprecated,
			"responses":   map[string]any{},
		}

		if len(r.Security) > 0 {
			secList := []map[string][]string{}
			for _, s := range r.Security {
				secList = append(secList, map[string][]string{s.Scheme: s.Scopes})
			}
			op["security"] = secList
		}

		if r.Extensions != nil {
			for k, v := range r.Extensions {
				op[k] = v
			}
		}

		params := []map[string]any{}
		for _, m := range pathParamRegex.FindAllStringSubmatch(r.Path, -1) {
			params = append(params, map[string]any{
				"name":        m[1],
				"in":          "path",
				"required":    true,
				"description": "Path parameter",
				"schema":      map[string]any{"type": "string"},
			})
		}
		if len(params) > 0 {
			op["parameters"] = params
		}

		// === requestBody с автоопределением multipart ===
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "DELETE" {
			contentTypes := r.RequestContentType

			// Если схема тела запроса содержит binary-поля — принудительно используем multipart
			if r.RequestBodySchemaName != "" {
				if schema, exists := schemas[r.RequestBodySchemaName]; exists {
					if hasBinaryField(schema) {
						contentTypes = []string{"multipart/form-data"}
					}
				}
			}

			content := map[string]any{}
			for _, ct := range contentTypes {
				schemaObj := map[string]any{"type": "object", "description": "Request body"}
				if r.RequestBodySchemaName != "" {
					schemaObj = map[string]any{"$ref": "#/components/schemas/" + r.RequestBodySchemaName}
				}
				content[ct] = map[string]any{"schema": schemaObj}
			}
			if len(content) > 0 {
				op["requestBody"] = map[string]any{"content": content}
			}
		}

		resps := op["responses"].(map[string]any)
		for _, resp := range r.Responses {
			code := strconv.Itoa(resp.Status)
			schemaName, hasSchema := r.ResponseSchemaNames[resp.Status]

			if resp.Status == 204 || resp.Status == 205 {
				resps[code] = map[string]any{"description": resp.Description}
				continue
			}

			content := map[string]any{}
			for _, ct := range resp.ContentTypes {
				var schemaObj map[string]any

				if hasSchema && schemaName != "" {
					schemaObj = map[string]any{"$ref": "#/components/schemas/" + schemaName}
				} else {
					// Если это ошибка и есть зарегистрированная схема — подтягиваем её
					if resp.IsError {
						if es := lookupErrorSchema(errorSchemas, resp.Status); es != nil && es.schemaName != "" {
							schemaObj = map[string]any{"$ref": "#/components/schemas/" + es.schemaName}
						} else {
							schemaObj = map[string]any{"type": "object"}
							if isBinaryContentType(ct) {
								schemaObj = map[string]any{"type": "string", "format": "binary"}
							}
						}
					} else {
						schemaObj = map[string]any{"type": "object"}
						if isBinaryContentType(ct) {
							schemaObj = map[string]any{"type": "string", "format": "binary"}
						}
					}
				}
				content[ct] = map[string]any{"schema": schemaObj}
			}

			// Описание: если это ошибка и есть зарегистрированная схема — используем её описание
			desc := resp.Description
			if resp.IsError {
				if es := lookupErrorSchema(errorSchemas, resp.Status); es != nil {
					desc = es.description
				}
			}

			if len(content) > 0 {
				resps[code] = map[string]any{
					"description": desc,
					"content":     content,
				}
			} else {
				resps[code] = map[string]any{"description": desc}
			}
		}

		pathItem[strings.ToLower(r.Method)] = op
	}

	return spec
}

func isBinaryContentType(ct string) bool {
	switch ct {
	case CTOctetStream, CTPNG, "image/jpeg", "image/gif", "application/pdf", CTCSV, CTPlainText:
		return true
	}
	return strings.Contains(ct, "octet") || strings.Contains(ct, "image/")
}

// lookupErrorSchema ищет зарегистрированную схему ошибки по статусу.
func lookupErrorSchema(errorSchemas map[int]*errorSchema, status int) *errorSchema {
	return errorSchemas[status]
}

func hasBinaryField(schema *oa.Schema) bool {
	if schema == nil {
		return false
	}
	// Проверяем свойства
	for _, prop := range schema.Properties {
		if prop.Format == "binary" {
			return true
		}
		if hasBinaryField(prop.Items) {
			return true
		}
		if slices.ContainsFunc(prop.OneOf, hasBinaryField) {
			return true
		}
		if slices.ContainsFunc(prop.AllOf, hasBinaryField) {
			return true
		}
		if slices.ContainsFunc(prop.AnyOf, hasBinaryField) {
			return true
		}
	}
	return false
}

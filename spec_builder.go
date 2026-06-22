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

// buildSpecFromData собирает spec из переданных данных (без глобальных переменных)
func buildSpecFromData(info Info, routes []RouteSpec, schemas map[string]*oa.Schema, errorSchemas map[int]*errorSchema, explicitTags map[string]string, securitySchemes []SecurityScheme, defaultSecurity []SecurityRequirement) map[string]any {
	refRemap := generateRefRemap(schemas)
	normalizedSchemas := normalizeAllSchemas(schemas, refRemap)
	tags := collectTags(routes, explicitTags)

	paths := map[string]any{}
	pathParamRegex := regexp.MustCompile(`\{([^}]+)}`)

	for _, r := range routes {
		pathItem := getOrCreatePathItem(paths, r.Path)
		op := buildOperation(r, refRemap, schemas, errorSchemas, pathParamRegex)
		pathItem[strings.ToLower(r.Method)] = op
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"servers": []map[string]any{{"url": ".", "description": "Local server"}},
		"paths":   paths,
		"components": map[string]any{
			"schemas": normalizedSchemas,
		},
		"tags": tags,
	}

	buildSpecSecuritySection(spec, securitySchemes, defaultSecurity, routes)

	return spec
}

// resolveRef remaps a $ref value through refRemap and returns an isolated object.
func resolveRef(refVal any, refRemap map[string]string) map[string]any {
	refStr, ok := refVal.(string)
	if !ok {
		return map[string]any{"$ref": refVal}
	}
	if mapped, exists := refRemap[refStr]; exists {
		return map[string]any{"$ref": mapped}
	}
	return map[string]any{"$ref": refStr}
}

// fixArrayConstraints replaces elval-gen array length keys with proper OpenAPI keys.
func fixArrayConstraints(raw map[string]any) {
	for oldKey, newKey := range map[string]string{
		"minimum":   "minItems",
		"maximum":   "maxItems",
		"minLength": "minItems",
		"maxLength": "maxItems",
	} {
		if v, ok := raw[oldKey]; ok {
			raw[newKey] = v
			delete(raw, oldKey)
		}
	}
}

// fixStringConstraints converts numeric constraints to string constraints when the schema
// is a string type. This fixes elval-gen bug where @evl:validate min:N on a rewritten
// string type generates "minimum" instead of "minLength".
func fixStringConstraints(raw map[string]any) {
	typ, _ := raw["type"].(string)
	if typ != "string" {
		return
	}
	if v, ok := raw["minimum"]; ok {
		raw["minLength"] = v
		delete(raw, "minimum")
	}
	if v, ok := raw["maximum"]; ok {
		raw["maxLength"] = v
		delete(raw, "maximum")
	}
}

// inferTypeFromConstraints infers the schema type from present constraint keys.
// This fixes elval-gen bug where @oa:rewrite.type doesn't inject "type" into the schema.
// OpenAPI spec requires that constraints like pattern/minLength apply only to string,
// and minimum/maximum apply only to number/integer.
func inferTypeFromConstraints(raw map[string]any) {
	// If type is already set, nothing to do
	if _, ok := raw["type"]; ok {
		return
	}
	// Infer from constraints
	if raw["pattern"] != nil || raw["minLength"] != nil || raw["maxLength"] != nil {
		raw["type"] = "string"
		return
	}
	if raw["minimum"] != nil || raw["maximum"] != nil ||
		raw["exclusiveMinimum"] != nil || raw["exclusiveMaximum"] != nil {
		raw["type"] = "number"
		return
	}
	// If we have minItems/maxItems, it's an array
	if raw["minItems"] != nil || raw["maxItems"] != nil || raw["items"] != nil {
		raw["type"] = "array"
		return
	}
	// If we have properties, it's an object
	if raw["properties"] != nil {
		raw["type"] = "object"
		return
	}
	// If we have additionalProperties, it's a map (object)
	if raw["additionalProperties"] != nil {
		raw["type"] = "object"
		return
	}
	// Infer from enum values
	if enumVal, ok := raw["enum"]; ok {
		if enumSlice, ok := enumVal.([]any); ok && len(enumSlice) > 0 {
			switch enumSlice[0].(type) {
			case string:
				raw["type"] = "string"
			case float64:
				raw["type"] = "number"
			case bool:
				raw["type"] = "boolean"
			}
			return
		}
	}
	// Fallback: empty schema (no type, no constraints) — default to object
	// This covers elval-gen cases like **T or *[]T where type is lost
	raw["type"] = "object"
}

// coerceNumericExample converts a string example to int/float when the schema type requires it.
func coerceNumericExample(raw map[string]any) {
	typ, _ := raw["type"].(string)
	if typ != "integer" && typ != "number" {
		return
	}
	ex, ok := raw["example"].(string)
	if !ok {
		return
	}
	if typ == "integer" {
		if parsed, err := strconv.ParseInt(ex, 10, 64); err == nil {
			raw["example"] = parsed
		}
	}
	if typ == "number" {
		if parsed, err := strconv.ParseFloat(ex, 64); err == nil {
			raw["example"] = parsed
		}
	}
}

// normalizeNestedSchemas recursively normalizes properties, items, and composition keywords.
func normalizeNestedSchemas(raw map[string]any, refRemap map[string]string) {
	if props, ok := raw["properties"].(map[string]any); ok {
		for key, val := range props {
			if propMap, ok := val.(map[string]any); ok {
				normalized := normalizeRawSchema(propMap, refRemap)
				if normalized != nil {
					props[key] = normalized
				}
			}
		}
	}

	if items, ok := raw["items"].(map[string]any); ok {
		raw["items"] = normalizeRawSchema(items, refRemap)
	}

	for _, key := range []string{"oneOf", "allOf", "anyOf"} {
		if schemasList, ok := raw[key].([]any); ok {
			for i, s := range schemasList {
				if sMap, ok := s.(map[string]any); ok {
					schemasList[i] = normalizeRawSchema(sMap, refRemap)
				}
			}
			raw[key] = schemasList
		}
	}
}

// normalizeRawSchema normalizes a raw schema map:
//   - isolates $ref (OpenAPI rule: $ref cannot coexist with other keys)
//   - fixes array constraints (minLength/maxLength → minItems/maxItems)
//   - coerces numeric examples (string → int/float)
//   - recursively normalizes nested schemas
func normalizeRawSchema(raw map[string]any, refRemap map[string]string) map[string]any {
	// $ref takes precedence — isolate and remap
	if ref, ok := raw["$ref"]; ok {
		return resolveRef(ref, refRemap)
	}

	// Инференс типа из ограничений (исправление бага elval-gen с @oa:rewrite.type)
	inferTypeFromConstraints(raw)

	typ, _ := raw["type"].(string)

	// Для string: minimum/maximum → minLength/maxLength (исправление бага elval-gen)
	fixStringConstraints(raw)

	// Для array: minimum/maximum/minLength/maxLength → minItems/maxItems
	if typ == "array" {
		fixArrayConstraints(raw)
	}

	// Приведение типа example для integer/number
	coerceNumericExample(raw)

	// Рекурсивно нормализуем вложенные схемы
	normalizeNestedSchemas(raw, refRemap)

	return raw
}

// normalizeSchema преобразует *oa.Schema в map[string]any с исправлениями:
//   - $ref изолируется (в OpenAPI $ref не может соседствовать с другими ключами)
//   - minimum/maximum на array преобразуются в minItems/maxItems
//   - рекурсивно нормализуются properties, items, oneOf, allOf, anyOf
//   - $ref значения ремапятся через refRemap (полные пути → короткие имена)
func normalizeSchema(schema *oa.Schema, refRemap map[string]string) map[string]any {
	if schema == nil {
		return nil
	}

	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}

	raw := make(map[string]any)
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	return normalizeRawSchema(raw, refRemap)
}

// collectRefsFromSchema собирает все $ref значения из схемы (properties, items, oneOf/allOf/anyOf)
func collectRefsFromSchema(schema *oa.Schema, refs map[string]bool) {
	if schema == nil || refs == nil {
		return
	}

	if schema.Ref != "" {
		ref := schema.Ref
		if !strings.HasPrefix(ref, "#/") {
			ref = "#/components/schemas/" + ref
		}
		refs[ref] = true
	}

	for _, child := range schema.Properties {
		collectRefsFromSchema(child, refs)
	}
	collectRefsFromSchema(schema.Items, refs)
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

func normalizeAllSchemas(schemas map[string]*oa.Schema, refRemap map[string]string) map[string]any {
	normalized := make(map[string]any, len(schemas))
	for name, schema := range schemas {
		normalized[name] = normalizeSchemaWithName(schema, refRemap, name)
	}
	return normalized
}

// normalizeSchemaWithName normalizes a schema and ensures it has a description.
// If the schema has no description, it generates one from the schema name (e.g. "03_nested.Address").
func normalizeSchemaWithName(schema *oa.Schema, refRemap map[string]string, schemaName string) map[string]any {
	normalized := normalizeSchema(schema, refRemap)
	if normalized == nil {
		return nil
	}
	if _, hasDesc := normalized["description"]; !hasDesc || normalized["description"] == "" {
		// Generate a human-readable description from the short name.
		// "03_nested.Address" -> "Address schema"
		parts := strings.Split(schemaName, ".")
		typeName := parts[len(parts)-1]
		normalized["description"] = typeName + " schema"
	}
	return normalized
}

func collectTags(routes []RouteSpec, explicitTags map[string]string) []map[string]any {
	seen := make(map[string]bool)
	var tags []map[string]any

	// Сначала добавляем явно зарегистрированные теги (из spec.AddTag)
	for name, desc := range explicitTags {
		if name != "" && !seen[name] {
			seen[name] = true
			tags = append(tags, map[string]any{"name": name, "description": desc})
		}
	}

	// Затем собираем теги из роутов (если тег уже есть — не дублируем)
	for _, r := range routes {
		for _, tag := range r.Tags {
			if tag != "" && !seen[tag] {
				seen[tag] = true
				// Если описание задано явно — используем его, иначе пустое
				desc := ""
				if d, ok := explicitTags[tag]; ok {
					desc = d
				}
				tags = append(tags, map[string]any{"name": tag, "description": desc})
			}
		}
	}
	return tags
}

func getOrCreatePathItem(paths map[string]any, path string) map[string]any {
	item, ok := paths[path].(map[string]any)
	if !ok {
		item = map[string]any{}
		paths[path] = item
	}
	return item
}

// buildOperation собирает operation object для одного маршрута.
func buildOperation(r RouteSpec, refRemap map[string]string, schemas map[string]*oa.Schema, errorSchemas map[int]*errorSchema, pathParamRegex *regexp.Regexp) map[string]any {
	op := map[string]any{
		"operationId": r.OperationID,
		"summary":     r.Summary,
		"description": r.Description,
		"tags":        r.Tags,
		"deprecated":  r.Deprecated,
	}

	buildOperationSecurity(op, r.Security)
	buildOperationExtensions(op, r.Extensions)
	buildOperationParameters(op, r, refRemap, pathParamRegex)
	buildOperationRequestBody(op, r, schemas)
	op["responses"] = buildResponses(r, errorSchemas, schemas)

	return op
}

func buildOperationSecurity(op map[string]any, secReqs []SecurityRequirement) {
	if len(secReqs) == 0 {
		return
	}
	secList := make([]any, 0, len(secReqs))
	for _, s := range secReqs {
		entry := map[string]any{}
		if len(s.Scopes) > 0 {
			entry[s.Scheme] = s.Scopes
		} else {
			entry[s.Scheme] = []string{}
		}
		secList = append(secList, entry)
	}
	op["security"] = secList
}

func buildOperationExtensions(op map[string]any, exts map[string]any) {
	for k, v := range exts {
		op[k] = v
	}
}

func buildOperationParameters(op map[string]any, r RouteSpec, refRemap map[string]string, pathParamRegex *regexp.Regexp) {
	var params []map[string]any

	if len(r.Parameters) > 0 {
		params = buildParamsFromSpec(r.Parameters, refRemap)
	} else {
		params = buildParamsFromPath(r.Path, pathParamRegex)
	}

	if len(params) > 0 {
		op["parameters"] = params
	}
}

func buildParamsFromSpec(parameters []*oa.Parameter, refRemap map[string]string) []map[string]any {
	seen := make(map[string]bool)
	var params []map[string]any

	for _, p := range parameters {
		key := p.Name + "/" + string(p.In)
		if seen[key] {
			continue
		}
		seen[key] = true

		paramMap := map[string]any{
			"name": p.Name,
			"in":   string(p.In),
		}
		if p.Description != "" {
			paramMap["description"] = p.Description
		} else {
			paramMap["description"] = p.Name + " parameter"
		}
		if p.Required {
			paramMap["required"] = true
		}
		if p.Schema != nil {
			paramMap["schema"] = normalizeSchema(p.Schema, refRemap)
		}
		if p.Example != nil {
			paramMap["example"] = p.Example
		}
		if p.Deprecated {
			paramMap["deprecated"] = true
		}
		if p.AllowEmptyValue {
			paramMap["allowEmptyValue"] = true
		}
		if p.Style != "" {
			paramMap["style"] = p.Style
		}
		if p.Explode != nil {
			paramMap["explode"] = *p.Explode
		}
		params = append(params, paramMap)
	}
	return params
}

func buildParamsFromPath(routePath string, pathParamRegex *regexp.Regexp) []map[string]any {
	var params []map[string]any
	for _, m := range pathParamRegex.FindAllStringSubmatch(routePath, -1) {
		params = append(params, map[string]any{
			"name":        m[1],
			"in":          "path",
			"required":    true,
			"description": "Path parameter",
			"schema":      map[string]any{"type": "string"},
		})
	}
	return params
}

func buildOperationRequestBody(op map[string]any, r RouteSpec, schemas map[string]*oa.Schema) {
	if r.Method == "GET" || r.Method == "HEAD" || r.Method == "DELETE" {
		return
	}

	contentTypes := r.RequestContentType
	if r.RequestBodySchemaName != "" {
		if schema, exists := schemas[r.RequestBodySchemaName]; exists {
			if hasBinaryField(schema) {
				contentTypes = []string{"multipart/form-data"}
			}
		}
	}

	content := map[string]any{}
	for _, ct := range contentTypes {
		var schemaObj map[string]any
		if r.RequestBodySchemaName != "" {
			if _, exists := schemas[r.RequestBodySchemaName]; exists {
				schemaObj = map[string]any{"$ref": "#/components/schemas/" + r.RequestBodySchemaName}
			} else {
				schemaObj = map[string]any{"type": "object"}
			}
		} else {
			schemaObj = map[string]any{"type": "object", "description": "Request body"}
		}
		content[ct] = map[string]any{"schema": schemaObj}
	}
	if len(content) > 0 {
		reqBody := map[string]any{"content": content}
		if r.Summary != "" {
			reqBody["description"] = r.Summary
		}
		reqBody["required"] = true
		op["requestBody"] = reqBody
	}
}

func buildResponses(r RouteSpec, errorSchemas map[int]*errorSchema, schemas map[string]*oa.Schema) map[string]any {
	resps := make(map[string]any)

	for _, resp := range r.Responses {
		code := strconv.Itoa(resp.Status)
		schemaName, hasSchema := r.ResponseSchemaNames[resp.Status]

		if resp.Status == 204 || resp.Status == 205 {
			resps[code] = map[string]any{"description": resp.Description}
			continue
		}

		content := map[string]any{}
		for _, ct := range resp.ContentTypes {
			schemaObj := buildResponseSchemaObject(resp, ct, schemaName, hasSchema, errorSchemas, schemas)
			content[ct] = map[string]any{"schema": schemaObj}
		}

		desc := buildResponseDescription(resp, errorSchemas)

		if len(content) > 0 {
			resps[code] = map[string]any{
				"description": desc,
				"content":     content,
			}
		} else {
			resps[code] = map[string]any{"description": desc}
		}
	}

	return resps
}

// buildResponseSchemaObject определяет schema object для response.
// Приоритет: явная схема > зарегистрированная схема ошибки > default (object или binary).
// Если схема указана, но не найдена в реестре — используется fallback {"type": "object"}.
func buildResponseSchemaObject(resp ResponseSpec, ct string, schemaName string, hasSchema bool, errorSchemas map[int]*errorSchema, schemas map[string]*oa.Schema) map[string]any {
	if hasSchema && schemaName != "" {
		if _, exists := schemas[schemaName]; exists {
			return map[string]any{"$ref": "#/components/schemas/" + schemaName}
		}
		// Схема не зарегистрирована (например, тип без OaSchema) — fallback
		return map[string]any{"type": "object"}
	}

	if resp.IsError {
		if es := errorSchemas[resp.Status]; es != nil && es.schemaName != "" {
			if _, exists := schemas[es.schemaName]; exists {
				return map[string]any{"$ref": "#/components/schemas/" + es.schemaName}
			}
		}
	}

	if isBinaryContentType(ct) {
		return map[string]any{"type": "string", "format": "binary"}
	}
	return map[string]any{"type": "object"}
}

// buildResponseDescription возвращает описание response, используя описание схемы ошибки если доступно.
func buildResponseDescription(resp ResponseSpec, errorSchemas map[int]*errorSchema) string {
	if resp.IsError {
		if es := errorSchemas[resp.Status]; es != nil {
			return es.description
		}
	}
	return resp.Description
}

func isBinaryContentType(ct string) bool {
	switch ct {
	case CTOctetStream, CTPNG, "image/jpeg", "image/gif", "application/pdf", CTCSV, CTPlainText:
		return true
	}
	return strings.Contains(ct, "octet") || strings.Contains(ct, "image/")
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

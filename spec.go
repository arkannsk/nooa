package nooa

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Info содержит метаданные для OpenAPI спецификации.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// SpecTransformer — функция для модификации спецификации перед отдачей.
type SpecTransformer func(spec map[string]any) map[string]any

// buildBaseSpec собирает базовую структуру OpenAPI из зарегистрированных роутов и схем.
func buildBaseSpec(info Info) map[string]any {
	// 1. Получаем зарегистрированные схемы (от elval)
	schemas := GetRegisteredSchemas()

	// --- ОТЛАДКА: Проверяем содержимое схем ---
	fmt.Printf("🔍 DEBUG: buildBaseSpec received %d schemas\n", len(schemas))
	for k := range schemas {
		fmt.Printf("   - Schema key: %s\n", k)
	}

	// Пробуем замаршалить схемы отдельно, чтобы увидеть их реальный вид
	if debugJSON, err := json.MarshalIndent(schemas, "", "  "); err == nil {
		fmt.Printf("📄 DEBUG: Schemas JSON content:\n%s\n", string(debugJSON))
	} else {
		fmt.Printf("❌ DEBUG: Failed to marshal schemas locally: %v\n", err)
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"servers": []map[string]string{{"url": "/", "description": "Current server"}},
		"paths":   map[string]any{},
		"components": map[string]any{
			"schemas": schemas, // Вставляем схемы сюда
		},
	}

	paths := spec["paths"].(map[string]any)
	pathParamRegex := regexp.MustCompile(`\{([^}]+)\}`)

	// 2. Собираем пути из RegisteredRoutes()
	for _, r := range RegisteredRoutes() {
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

		// Security
		if len(r.Security) > 0 {
			secList := []map[string][]string{}
			for _, s := range r.Security {
				secList = append(secList, map[string][]string{s.Scheme: s.Scopes})
			}
			op["security"] = secList
		}

		// Extensions (x-*)
		if r.Extensions != nil {
			for k, v := range r.Extensions {
				op[k] = v
			}
		}

		// Path Parameters
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

		// Request Body (если не GET/HEAD/DELETE)
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "DELETE" {
			content := map[string]any{}
			for _, ct := range r.RequestContentType {
				// Пытаемся найти схему по имени типа, если бы мы знали имя.
				// Пока ставим заглушку, но в идеале здесь должна быть $ref.
				// Так как мы используем RegisterModel, мы можем сослаться на схему, если знаем её имя.
				// Но пока оставим object, так как маппинг Req -> Schema Name не автоматический без反射.
				content[ct] = map[string]any{
					"schema": map[string]any{
						"type":        "object",
						"description": "Request schema (see components/schemas)",
					},
				}
			}
			op["requestBody"] = map[string]any{"content": content}
		}

		// Responses
		resps := op["responses"].(map[string]any)
		for _, resp := range r.Responses {
			code := strconv.Itoa(resp.Status)
			content := map[string]any{}
			for _, ct := range resp.ContentTypes {
				schema := map[string]any{"type": "object"}
				if isBinaryContentType(ct) {
					schema = map[string]any{"type": "string", "format": "binary"}
				}
				content[ct] = map[string]any{"schema": schema}
			}
			resps[code] = map[string]any{
				"description": resp.Description,
				"content":     content,
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

// GenerateSpecAtStartup генерирует финальный JSON спецификации.
func GenerateSpecAtStartup(info Info, transformers ...SpecTransformer) ([]byte, error) {
	spec := buildBaseSpec(info)

	// Применяем трансформеры (например, добавление security schemes)
	for _, t := range transformers {
		spec = t(spec)
	}

	// Финальная сериализация
	return json.MarshalIndent(spec, "", "  ")
}

// SpecMiddleware создает HTTP-мидлвару для отдачи спецификации по пути /openapi.json.
func SpecMiddleware(next http.Handler, info Info, transformers ...SpecTransformer) http.Handler {
	// Проверка схем перед генерацией
	schemas := GetRegisteredSchemas()
	fmt.Printf("🚀 SpecMiddleware: Found %d registered schemas before generation.\n", len(schemas))
	if len(schemas) == 0 {
		fmt.Println("️  WARNING: No schemas found! Make sure RegisterModel[] is called in init().")
	}

	specJSON, err := GenerateSpecAtStartup(info, transformers...)
	if err != nil {
		log.Printf("❌ Error generating spec: %v", err)
		specJSON = []byte(`{"openapi":"3.0.3","info":{"title":"Error","version":"0.0"},"paths":{},"components":{"schemas":{}}}`)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			_, _ = w.Write(specJSON)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SpecMiddlewareWithPath аналогичен SpecMiddleware, но с кастомным путем.
func SpecMiddlewareWithPath(next http.Handler, info Info, path string, transformers ...SpecTransformer) http.Handler {
	specJSON, err := GenerateSpecAtStartup(info, transformers...)
	if err != nil {
		log.Printf("Error generating spec: %v", err)
		specJSON = []byte(`{"openapi":"3.0.3","info":{"title":"Error","version":"0.0"},"paths":{}}`)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(specJSON)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// === Helpers for Transformers ===

// WithSecuritySchemes добавляет глобальные схемы безопасности.
func WithSecuritySchemes(schemes map[string]any) SpecTransformer {
	return func(spec map[string]any) map[string]any {
		comp, ok := spec["components"].(map[string]any)
		if !ok {
			comp = make(map[string]any)
			spec["components"] = comp
		}
		comp["securitySchemes"] = schemes
		return spec
	}
}

// FilterByTag удаляет пути, содержащие указанный тег.
func FilterByTag(excludeTag string) SpecTransformer {
	return func(spec map[string]any) map[string]any {
		paths, ok := spec["paths"].(map[string]any)
		if !ok {
			return spec
		}
		for path, pathItem := range paths {
			itemMap, ok := pathItem.(map[string]any)
			if !ok {
				continue
			}
			needDelete := false
			for method, op := range itemMap {
				if method == "parameters" {
					continue
				}
				opMap, ok := op.(map[string]any)
				if !ok {
					continue
				}
				tags, _ := opMap["tags"].([]any)
				for _, t := range tags {
					if tStr, ok := t.(string); ok && tStr == excludeTag {
						needDelete = true
						break
					}
				}
				if needDelete {
					break
				}
			}
			if needDelete {
				delete(paths, path)
			}
		}
		return spec
	}
}

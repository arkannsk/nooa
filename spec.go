// Package nooa — добавь этот файл в корень модуля
// spec.go: генерация базовой спецификации на старте сервера

package nooa

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Info — метаданные для OpenAPI spec.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// GenerateSpecAtStartup собирает скелет OpenAPI 3.0 из RegisteredRoutes().
// Не требует AST/reflection — работает полностью в рантайме.
// ⚠️ Схемы тел запросов/ответов будут заглушками (тип стёрт при компиляции).
func GenerateSpecAtStartup(info Info) ([]byte, error) {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"servers": []map[string]string{{"url": "/", "description": "Current server"}},
		"paths":   map[string]any{},
		"components": map[string]any{
			"schemas": map[string]any{}, // заглушки, заполняются AST-генератором
		},
	}
	paths := spec["paths"].(map[string]any)

	pathParamRegex := regexp.MustCompile(`\{([^}]+)\}`)

	for _, r := range RegisteredRoutes() {
		// Поддержка нескольких методов на один path
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
			sec := []map[string][]string{}
			for _, s := range r.Security {
				sec = append(sec, map[string][]string{s.Scheme: s.Scopes})
			}
			op["security"] = sec
		}

		// Path параметры (без типов — только имена)
		params := []map[string]any{}
		for _, m := range pathParamRegex.FindAllStringSubmatch(r.Path, -1) {
			params = append(params, map[string]any{
				"name":        m[1],
				"in":          "path",
				"required":    true,
				"description": "Path parameter",
				"schema":      map[string]any{"type": "string"}, // заглушка
			})
		}
		if len(params) > 0 {
			op["parameters"] = params
		}

		// Request body (если есть)
		// Тип тела неизвестен в рантайме → заглушка
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "DELETE" {
			content := map[string]any{}
			for _, ct := range r.RequestContentType {
				content[ct] = map[string]any{
					"schema": map[string]any{
						"type":        "object",
						"description": "Request schema available via go:generate",
					},
				}
			}
			op["requestBody"] = map[string]any{
				"content": content,
			}
		}

		// Responses
		resps := op["responses"].(map[string]any)
		for _, resp := range r.Responses {
			code := strconv.Itoa(resp.Status)
			content := map[string]any{}
			for _, ct := range resp.ContentTypes {
				// Автодетект бинарных типов по расширению/CT
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

	return json.MarshalIndent(spec, "", "  ")
}

func isBinaryContentType(ct string) bool {
	switch ct {
	case CTOctetStream, CTPNG, "image/jpeg", "image/gif", "application/pdf":
		return true
	case CTPlainText:
		return true // текст, но не JSON
	}
	return strings.Contains(ct, "octet") || strings.Contains(ct, "image/")
}

// SpecMiddleware оборачивает handler и отдаёт /openapi.json
func SpecMiddleware(next http.Handler, info Info) http.Handler {
	specJSON, err := GenerateSpecAtStartup(info)
	if err != nil {
		// Fallback: пустая спецификация
		specJSON = []byte(`{"openapi":"3.0.3","info":{},"paths":{}}`)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(specJSON)
			return
		}
		// Опционально: Swagger UI на /
		// if r.URL.Path == "/" || r.URL.Path == "/docs" { ... }
		next.ServeHTTP(w, r)
	})
}

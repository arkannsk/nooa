package nooa

import (
	"regexp"
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
func buildSpecFromData(info Info, routes []RouteSpec, schemas map[string]*oa.Schema) map[string]any {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info":    info,
		"servers": []map[string]string{{"url": "/", "description": "Current server"}},
		"paths":   map[string]any{},
		"components": map[string]any{
			"schemas": schemas,
		},
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

		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "DELETE" {
			content := map[string]any{}
			for _, ct := range r.RequestContentType {
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
					schemaObj = map[string]any{"type": "object"}
					if isBinaryContentType(ct) {
						schemaObj = map[string]any{"type": "string", "format": "binary"}
					}
				}
				content[ct] = map[string]any{"schema": schemaObj}
			}

			if len(content) > 0 {
				resps[code] = map[string]any{
					"description": resp.Description,
					"content":     content,
				}
			} else {
				resps[code] = map[string]any{"description": resp.Description}
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

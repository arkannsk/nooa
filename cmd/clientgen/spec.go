package main

import (
	"fmt"
	"strings"
)

// str extracts a string value from a map, returning empty string if missing or nil.
func str(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// boolVal extracts a bool value from a map, returning false if missing or nil.
func boolVal(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}

// Spec represents a parsed OpenAPI 3.x document.
type Spec struct {
	Info     Info
	Paths    map[string]*PathItem
	Schemas  map[string]*Schema
	Tags     []Tag
}

// Info holds spec metadata.
type Info struct {
	Title   string
	Version string
}

// Tag represents an OpenAPI tag.
type Tag struct {
	Name        string
	Description string
}

// PathItem represents a single path in the spec.
type PathItem struct {
	Path      string
	Operations map[string]*Operation // "get", "post", etc.
}

// Operation represents a single HTTP operation.
type Operation struct {
	Method      string
	ID          string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   map[string]*Response
}

// Parameter represents an operation parameter.
type Parameter struct {
	Name        string
	In          string // "path", "query", "header", "cookie"
	Required    bool
	Description string
	Schema      *Schema
}

// RequestBody represents an operation request body.
type RequestBody struct {
	Description string
	Required    bool
	Content     map[string]*MediaType // content-type -> media type
}

// Response represents an operation response.
type Response struct {
	Description string
	Content     map[string]*MediaType
}

// MediaType represents a content type with a schema.
type MediaType struct {
	Schema *Schema
}

// Schema represents an OpenAPI schema.
type Schema struct {
	Name       string
	Type       string            // "object", "array", "string", "integer", "number", "boolean"
	Properties map[string]*Schema
	Items      *Schema           // for arrays
	Ref        string            // "$ref" target
	Format     string            // "int64", "float", "date-time", etc.
	Required   []string
	Description string
}

// parseSpec parses an OpenAPI 3.x spec from a map.
func parseSpec(raw map[string]any) (*Spec, error) {
	s := &Spec{
		Paths:   make(map[string]*PathItem),
		Schemas: make(map[string]*Schema),
	}

	// Parse info
	if info, ok := raw["info"].(map[string]any); ok {
		s.Info.Title, _ = info["title"].(string)
		s.Info.Version, _ = info["version"].(string)
	}

	// Parse tags
	if tagsRaw, ok := raw["tags"].([]any); ok {
		for _, t := range tagsRaw {
			if tm, ok := t.(map[string]any); ok {
				s.Tags = append(s.Tags, Tag{
					Name:        str(tm, "name"),
					Description: str(tm, "description"),
				})
			}
		}
	}

	// Parse paths
	if pathsRaw, ok := raw["paths"].(map[string]any); ok {
		for pathStr, pathVal := range pathsRaw {
			pm, ok := pathVal.(map[string]any)
			if !ok {
				continue
			}
			pi := &PathItem{Path: pathStr, Operations: make(map[string]*Operation)}
			for method, opVal := range pm {
				if !strings.HasPrefix(strings.ToLower(method), "get") &&
					!strings.HasPrefix(strings.ToLower(method), "post") &&
					!strings.HasPrefix(strings.ToLower(method), "put") &&
					!strings.HasPrefix(strings.ToLower(method), "delete") &&
					!strings.HasPrefix(strings.ToLower(method), "patch") &&
					!strings.HasPrefix(strings.ToLower(method), "head") &&
					!strings.HasPrefix(strings.ToLower(method), "options") {
					continue
				}
				op, err := parseOperation(opVal, method)
				if err != nil {
					return nil, fmt.Errorf("path %s method %s: %w", pathStr, method, err)
				}
				pi.Operations[method] = op
			}
			s.Paths[pathStr] = pi
		}
	}

	// Parse schemas
	if comp, ok := raw["components"].(map[string]any); ok {
		if schemasRaw, ok := comp["schemas"].(map[string]any); ok {
			for name, schemaVal := range schemasRaw {
				sm, ok := schemaVal.(map[string]any)
				if !ok {
					continue
				}
								schema, err := parseSchema(sm, name)
				if err != nil {
					return nil, fmt.Errorf("schema %s: %w", name, err)
				}
				s.Schemas[name] = schema
			}
		}
	}

	return s, nil
}

func parseOperation(raw any, method string) (*Operation, error) {
	opMap, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid operation")
	}

	op := &Operation{
		Method:    strings.ToUpper(method),
		ID:        str(opMap, "operationId"),
		Summary:   str(opMap, "summary"),
		Responses: make(map[string]*Response),
	}

	if desc, ok := opMap["description"].(string); ok {
		op.Description = desc
	}
	if tags, ok := opMap["tags"].([]any); ok {
		for _, t := range tags {
			if ts, ok := t.(string); ok {
				op.Tags = append(op.Tags, ts)
			}
		}
	}

	// Parameters
	if params, ok := opMap["parameters"].([]any); ok {
		for _, p := range params {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			param := Parameter{
				Name:        str(pm, "name"),
				In:          str(pm, "in"),
				Required:    boolVal(pm, "required"),
				Description: str(pm, "description"),
			}
			if schemaRaw, ok := pm["schema"].(map[string]any); ok {
				param.Schema, _ = parseSchema(schemaRaw, "")
			}
			op.Parameters = append(op.Parameters, param)
		}
	}

	// Request body
	if rbRaw, ok := opMap["requestBody"].(map[string]any); ok {
		rb := &RequestBody{
			Required:    boolVal(rbRaw, "required"),
			Description: str(rbRaw, "description"),
			Content:     make(map[string]*MediaType),
		}
		if content, ok := rbRaw["content"].(map[string]any); ok {
			for ct, ctVal := range content {
				ctm, ok := ctVal.(map[string]any)
				if !ok {
					continue
				}
				mt := &MediaType{}
				if schemaRaw, ok := ctm["schema"].(map[string]any); ok {
					mt.Schema, _ = parseSchema(schemaRaw, "")
				}
				rb.Content[ct] = mt
			}
		}
		op.RequestBody = rb
	}

	// Responses
	if resps, ok := opMap["responses"].(map[string]any); ok {
		for code, respVal := range resps {
			rm, ok := respVal.(map[string]any)
			if !ok {
				continue
			}
			resp := &Response{
				Description: str(rm, "description"),
				Content:     make(map[string]*MediaType),
			}
			if content, ok := rm["content"].(map[string]any); ok {
				for ct, ctVal := range content {
					ctm, ok := ctVal.(map[string]any)
					if !ok {
						continue
					}
					mt := &MediaType{}
					if schemaRaw, ok := ctm["schema"].(map[string]any); ok {
						mt.Schema, _ = parseSchema(schemaRaw, "")
					}
					resp.Content[ct] = mt
				}
			}
			op.Responses[code] = resp
		}
	}

	return op, nil
}

func parseSchema(raw map[string]any, name string) (*Schema, error) {
	s := &Schema{Name: name}

	if ref, ok := raw["$ref"].(string); ok {
		s.Ref = ref
		// Extract short name from "#/components/schemas/Name"
		s.Name = strings.TrimPrefix(ref, "#/components/schemas/")
		return s, nil
	}

	s.Type, _ = raw["type"].(string)
	s.Format, _ = raw["format"].(string)
	s.Description, _ = raw["description"].(string)

	if props, ok := raw["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*Schema)
		for pname, pval := range props {
			pm, ok := pval.(map[string]any)
			if !ok {
				continue
			}
			ps, err := parseSchema(pm, pname)
			if err != nil {
				return nil, err
			}
			s.Properties[pname] = ps
		}
	}

	if req, ok := raw["required"].([]any); ok {
		for _, r := range req {
			if rs, ok := r.(string); ok {
				s.Required = append(s.Required, rs)
			}
		}
	}

	if items, ok := raw["items"].(map[string]any); ok {
		s.Items, _ = parseSchema(items, "")
	}

	return s, nil
}

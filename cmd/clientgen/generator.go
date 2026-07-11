package main

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

//go:embed templates/*.tpl
var templatesFS embed.FS

// Generator produces Go client code from parsed package info.
type Generator struct {
	info           *PackageInfo
	cfg            *Config
	packageAliases map[string]string // import path -> alias
	title          string
	version        string
}

// NewGenerator creates a new Generator.
func NewGenerator(info *PackageInfo, cfg *Config) *Generator {
	return &Generator{
		info:    info,
		cfg:     cfg,
		title:   info.Title,
		version: info.Version,
	}
}

// Generate renders the client Go code.
func (g *Generator) Generate() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoName":    toGoName,
		"toGoTypeName": toGoTypeName,
		"camelCase":   camelCase,
		"trimSlash":   strings.TrimRight,
		"retNil":      func(b bool) string { if b { return "nil" }; return "" },
		"retPrefix":   func(b bool) string { if b { return "nil, " }; return "" },
	}

	tmpl, err := template.New("client").Funcs(funcMap).ParseFS(templatesFS, "templates/*.tpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	data := g.buildTemplateData()

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "client.go.tpl", data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// TemplateData holds all data passed to the template.
type TemplateData struct {
	Package             string
	Title               string
	Version             string
	Imports             []string
	PackageAliases      map[string]string // import path -> alias
	Operations          []*OperationData
	HasQueryParams      bool
	HasRequestBody      bool
	HasStreamingResponse bool
}

// OperationData holds parsed data for a single operation.
type OperationData struct {
	Path           string
	Method         string
	ID             string
	MethodName     string
	Summary        string
	Description    string
	Tags           []string
	PathParams     []ParameterData
	QueryParams    []ParameterData
	HeaderParams   []ParameterData
	RequestBody    *RequestBodyData
	ResponseBody   *ResponseBodyData
	SuccessCodes   []string
	HasRequestBody bool
	// UsesModelAsInput — when true, the method accepts the model directly
	// instead of a generated *Request struct.
	UsesModelAsInput bool
	// ModelInputGoType — the model type with package alias prefix (e.g. "httpparams.QueryParams").
	ModelInputGoType string
}

// ParameterData holds data for a single parameter.
type ParameterData struct {
	Name        string // parameter name for HTTP (e.g. "X-API-Key", "query")
	GoFieldName string // original Go struct field name (e.g. "APIKey", "Query")
	In          string
	Required    bool
	Description string
	GoType      string
	ImportPath  string
}

// RequestBodyData holds data for a request body.
type RequestBodyData struct {
	Required    bool
	Description string
	SchemaName  string
	GoType      string
	ImportPath  string
}

// ResponseBodyData holds data for a response body container.
type ResponseBodyData struct {
	Fields map[string]*ResponseField
}

// ResponseField holds info for one status code in the response container.
type ResponseField struct {
	StatusCode   int
	SchemaName   string
	GoType       string
	ImportPath   string
	ContentTypes []string
}

func (g *Generator) buildTemplateData() *TemplateData {
	data := &TemplateData{
		Package: g.cfg.Package,
		Title:   g.info.Title,
		Version: g.info.Version,
	}

	// Sort routes by path for deterministic output
	sortedRoutes := make([]RouteInfo, len(g.info.Routes))
	copy(sortedRoutes, g.info.Routes)
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].Path < sortedRoutes[j].Path
	})

	// First pass: collect used imports
	usedImports := make(map[string]bool)
	for _, route := range sortedRoutes {
		if route.ReqImport != "" {
			usedImports[route.ReqImport] = true
		}
		if route.ResImport != "" {
			usedImports[route.ResImport] = true
		}
		// MultiRespEntries may have their own imports
		for _, entry := range route.MultiRespEntries {
			if entry.ImportPath != "" {
				usedImports[entry.ImportPath] = true
			}
		}
	}

	// Build package aliases for used imports
	g.packageAliases = make(map[string]string)
	importList := make([]string, 0, len(usedImports))
	for impPath := range usedImports {
		importList = append(importList, impPath)
		parts := strings.Split(impPath, "/")
		last := parts[len(parts)-1]
		alias := sanitizePackageName(last)
		g.packageAliases[impPath] = alias
	}
	sort.Strings(importList)
	data.Imports = importList
	data.PackageAliases = g.packageAliases

	// Second pass: build operations with aliases available
	var ops []*OperationData
	for _, route := range sortedRoutes {
		od := g.buildOperationData(route)
		if od != nil {
			ops = append(ops, od)
		}
	}

	// Check flags
	for _, op := range ops {
		if len(op.QueryParams) > 0 {
			data.HasQueryParams = true
		}
		if op.RequestBody != nil {
			data.HasRequestBody = true
		}
	}
	data.Operations = ops

	return data
}

func (g *Generator) buildOperationData(route RouteInfo) *OperationData {
	od := &OperationData{
		Path:        route.Path,
		Method:      route.Method,
		ID:          route.OperationID,
		MethodName:  sanitizeMethodName(route.OperationID),
		Summary:     route.Summary,
		Tags:        route.Tags,
	}
	if od.MethodName == "" {
		od.MethodName = camelCase(route.Method)
	}

	// Extract params from model annotations (if available) or from path template
	od.PathParams = g.extractPathParams(route)
	od.QueryParams = g.extractQueryParams(route)
	od.HeaderParams = g.extractHeaderParams(route)

	// Determine if we can use the model directly as input.
	// When the request model exists, always accept it directly —
	// no need to generate a duplicate *Request struct.
	hasAnyParams := len(od.PathParams) > 0 || len(od.QueryParams) > 0 || len(od.HeaderParams) > 0
	if route.ReqImport != "" {
		od.UsesModelAsInput = true
		od.ModelInputGoType = g.typeRef(route.ReqType, route.ReqImport)
		// Only set HasRequestBody if the model actually has body fields
		if route.HasReqBody && route.HasBodyFields {
			od.HasRequestBody = true
			od.RequestBody = &RequestBodyData{
				Required:   true,
				SchemaName: route.ReqType,
				GoType:     g.typeRef(route.ReqType, route.ReqImport),
				ImportPath: route.ReqImport,
			}
		}
	} else if hasAnyParams {
		// Params exist but no request model — generate *Request struct.
		od.UsesModelAsInput = false
	}

	// Response body — collect all status codes
	if route.HasRespBody {
		fields := make(map[string]*ResponseField)
		successCodes := make([]string, 0)

		// NewRouteMultiResp: each entry has its own schema and import
		if len(route.MultiRespEntries) > 0 {
			for _, entry := range route.MultiRespEntries {
				key := strconv.Itoa(entry.Status)
				if _, exists := fields[key]; exists {
					continue
				}
				ct := entry.ContentTypes
				if len(ct) == 0 {
					ct = []string{"application/json"}
				}
				fields[key] = &ResponseField{
					StatusCode:   entry.Status,
					SchemaName:   entry.SchemaName,
					GoType:       g.typeRef(entry.SchemaName, entry.ImportPath),
					ImportPath:   entry.ImportPath,
					ContentTypes: ct,
				}
				successCodes = append(successCodes, key)
			}
		} else if len(route.ResponseStatuses) > 0 {
			// NewRoute with explicit OnSuccess/OnNoContent/Response
			for _, status := range route.ResponseStatuses {
				key := strconv.Itoa(status)
				if _, exists := fields[key]; exists {
					continue
				}

				schemaName := route.ResType
				if route.ResponseSchemas != nil {
					if sn, ok := route.ResponseSchemas[status]; ok {
						schemaName = sn
					}
				}
				ct := []string{"application/json"}
				if route.ResponseContentTypes != nil {
					if cts, ok := route.ResponseContentTypes[status]; ok && len(cts) > 0 {
						ct = cts
					}
				}

				fields[key] = &ResponseField{
					StatusCode:   status,
					SchemaName:   schemaName,
					GoType:       g.typeRef(schemaName, route.ResImport),
					ImportPath:   route.ResImport,
					ContentTypes: ct,
				}
				successCodes = append(successCodes, key)
			}
		} else if route.ResImport != "" {
			// No explicit statuses — default to 200
			fields["200"] = &ResponseField{
				StatusCode:   200,
				SchemaName:   route.ResType,
				GoType:       g.typeRef(route.ResType, route.ResImport),
				ImportPath:   route.ResImport,
				ContentTypes: []string{"application/json"},
			}
			successCodes = append(successCodes, "200")
		}

		if len(fields) > 0 {
			od.ResponseBody = &ResponseBodyData{Fields: fields}
			od.SuccessCodes = successCodes
		}
	}

	return od
}

// extractPathParams returns path params from model annotations if available,
// otherwise falls back to parsing the path template.
func (g *Generator) extractPathParams(route RouteInfo) []ParameterData {
	if len(route.PathParams) > 0 {
		var params []ParameterData
		for _, p := range route.PathParams {
			params = append(params, ParameterData{
				Name:        p.ParamName,
				GoFieldName: p.Name,
				In:          "path",
				Required:    true,
				Description: p.Description,
				GoType:      p.GoType,
			})
		}
		return params
	}
	// Fallback: parse from path template
	var params []ParameterData
	segments := strings.Split(route.Path, "/")
	for _, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := seg[1 : len(seg)-1]
			params = append(params, ParameterData{
				Name:        name,
				GoFieldName: toGoTypeName(name),
				In:          "path",
				Required:    true,
				GoType:      "string",
			})
		}
	}
	return params
}

// extractQueryParams returns query params from model annotations.
func (g *Generator) extractQueryParams(route RouteInfo) []ParameterData {
	var params []ParameterData
	for _, p := range route.QueryParams {
		params = append(params, ParameterData{
			Name:        p.ParamName,
			GoFieldName: p.Name,
			In:          "query",
			Required:    false,
			Description: p.Description,
			GoType:      p.GoType,
		})
	}
	return params
}

// extractHeaderParams returns header params from model annotations.
func (g *Generator) extractHeaderParams(route RouteInfo) []ParameterData {
	var params []ParameterData
	for _, p := range route.HeaderParams {
		params = append(params, ParameterData{
			Name:        p.ParamName,
			GoFieldName: p.Name,
			In:          "header",
			Required:    false,
			Description: p.Description,
			GoType:      p.GoType,
		})
	}
	return params
}
// typeRef returns the type name prefixed with package alias if applicable.
func (g *Generator) typeRef(typeName string, importPath string) string {
	if importPath == "" {
		return typeName
	}
	alias, ok := g.packageAliases[importPath]
	if !ok {
		return typeName
	}
	// typeName is like "alias.TypeName" or just "TypeName"
	parts := strings.Split(typeName, ".")
	last := parts[len(parts)-1]
	return alias + "." + last
}

// toGoName converts a name to a valid Go identifier (exported).
func toGoName(name string) string {
	return toGoTypeName(name)
}

// toGoTypeName converts a name to a Go type name.
func toGoTypeName(name string) string {
	if name == "" {
		return "Object"
	}
	// Handle numeric status codes: "200" -> "Ok", "201" -> "Created", "404" -> "NotFound"
	statusNames := map[string]string{
		"200": "Ok",
		"201": "Created",
		"202": "Accepted",
		"204": "NoContent",
		"400": "BadRequest",
		"401": "Unauthorized",
		"403": "Forbidden",
		"404": "NotFound",
		"500": "InternalServerError",
	}
	if n, ok := statusNames[name]; ok {
		return n
	}
	parts := strings.Split(name, ".")
	last := parts[len(parts)-1]
	return camelCase(last)
}

// sanitizeMethodName removes characters invalid for Go identifiers.
func sanitizeMethodName(s string) string {
	r := strings.NewReplacer(
		":", "",
		"/", "",
		"-", "",
		"_", "",
	)
	name := r.Replace(s)
	if name == "" {
		return "Operation"
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		name = strings.ToUpper(string(name[0])) + name[1:]
	}
	return name
}

// sanitizePackageName ensures the package name is a valid Go identifier.
func sanitizePackageName(s string) string {
	r := strings.NewReplacer(
		" ", "",
		"-", "",
		"_", "",
		":", "",
		"/", "",
		"&", "",
		"|", "",
		"*", "",
		"?", "",
		"!", "",
		"#", "",
		"$", "",
		"%", "",
		"@", "",
		"~", "",
	)
	name := r.Replace(strings.ToLower(s))
	name = strings.TrimLeft(name, "0123456789")
	if name == "" {
		name = "client"
	}
	return name
}

// camelCase converts a string to CamelCase.
func camelCase(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

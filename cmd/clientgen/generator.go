package main

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
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
	Package        string
	Title          string
	Version        string
	Imports        []string
	PackageAliases map[string]string // import path -> alias
	Operations     []*OperationData
	HasQueryParams bool
	HasRequestBody bool
}

// OperationData holds parsed data for a single operation.
type OperationData struct {
	Path         string
	Method       string
	ID           string
	MethodName   string
	Summary      string
	Description  string
	Tags         []string
	PathParams   []ParameterData
	QueryParams  []ParameterData
	HeaderParams []ParameterData
	RequestBody  *RequestBodyData
	ResponseBody *ResponseBodyData
	SuccessCodes []string
}

// ParameterData holds data for a single parameter.
type ParameterData struct {
	Name        string
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
	StatusCode int
	SchemaName string
	GoType     string
	ImportPath string
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
		PathParams:  g.extractPathParams(route.Path),
	}
	if od.MethodName == "" {
		od.MethodName = camelCase(route.Method)
	}

	// Request body for POST/PUT/PATCH
	if route.HasReqBody && route.ReqImport != "" {
		od.RequestBody = &RequestBodyData{
			Required:   true,
			SchemaName: route.ReqType,
			GoType:     g.typeRef(route.ReqType, route.ReqImport),
			ImportPath: route.ReqImport,
		}
	}

	// Response body
	if route.HasRespBody && route.ResImport != "" {
		od.ResponseBody = &ResponseBodyData{
			Fields: map[string]*ResponseField{
				"200": {
					StatusCode: 200,
					SchemaName: route.ResType,
					GoType:     g.typeRef(route.ResType, route.ResImport),
					ImportPath: route.ResImport,
				},
			},
		}
		od.SuccessCodes = []string{"200"}
	}

	return od
}

// extractPathParams parses path template like /users/{id} and returns path params.
func (g *Generator) extractPathParams(path string) []ParameterData {
	var params []ParameterData
	segments := strings.Split(path, "/")
	for _, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := seg[1 : len(seg)-1]
			params = append(params, ParameterData{
				Name:     name,
				In:       "path",
				Required: true,
				GoType:   "string",
			})
		}
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

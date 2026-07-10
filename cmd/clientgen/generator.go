package main

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

//go:embed templates/*.tpl
var templatesFS embed.FS

// Generator produces Go client code from a parsed OpenAPI spec.
type Generator struct {
	spec           *Spec
	cfg            *Config
	title          string
	version        string
	packageAliases map[string]string // import path -> alias
}

// NewGenerator creates a new Generator.
func NewGenerator(raw map[string]any, cfg *Config, title, version string) *Generator {
	s, _ := parseSpec(raw)
	return &Generator{
		spec:    s,
		cfg:     cfg,
		title:   title,
		version: version,
	}
}

// Generate renders the client Go code.
func (g *Generator) Generate() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":     g.toGoType,
		"toGoName":     toGoName,
		"toGoTypeName": toGoTypeName,
		"toImportPath": g.toImportPath,
		"camelCase":    camelCase,
		"trimSlash":    strings.TrimRight,
		"hasJSON":      g.hasJSONResponse,
		"jsonSchema":   g.jsonSchema,
		"requestSchema": g.requestSchema,
		"pathParams":   g.pathParams,
		"retNil":       func(b bool) string { if b { return "nil" }; return "" },
		"queryParams":  g.queryParams,
		"headerParams": g.headerParams,
		"responseTypeName": g.responseTypeName,
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
	BaseImport     string
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
	MethodName   string // Go-exported method name
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
	// Fields maps status code -> schema info
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
		Package:    g.cfg.Package,
		Title:      g.title,
		Version:    g.version,
		BaseImport: g.cfg.BaseImport,
	}

	// First pass: collect all imports
	imports := make(map[string]bool)
	methodOrder := []string{"get", "post", "put", "patch", "delete", "head", "options"}
	pathKeys := make([]string, 0, len(g.spec.Paths))
	for p := range g.spec.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)
	for _, pathKey := range pathKeys {
		pi := g.spec.Paths[pathKey]
		for _, method := range methodOrder {
			op, ok := pi.Operations[method]
			if !ok {
				continue
			}
			g.collectImports(op, imports)
		}
	}

	// Build package aliases
	g.packageAliases = make(map[string]string)
	importList := make([]string, 0, len(imports))
	for imp := range imports {
		importList = append(importList, imp)
		parts := strings.Split(imp, "/")
		last := parts[len(parts)-1]
		alias := sanitizePackageName(last)
		g.packageAliases[imp] = alias
	}
	sort.Strings(importList)
	data.Imports = importList
	data.PackageAliases = g.packageAliases

	// Second pass: build operations with aliases available
	var ops []*OperationData
	for _, pathKey := range pathKeys {
		pi := g.spec.Paths[pathKey]
		for _, method := range methodOrder {
			op, ok := pi.Operations[method]
			if !ok {
				continue
			}
			od := g.buildOperationData(op, pathKey, imports)
			if od != nil {
				ops = append(ops, od)
			}
		}
	}

	// Check if any operation has query params or request body
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

// collectImports gathers all import paths from an operation's schemas.
func (g *Generator) collectImports(op *Operation, imports map[string]bool) {
	// Request body
	if op.RequestBody != nil {
		for _, mt := range op.RequestBody.Content {
			if mt.Schema != nil {
				if imp := g.toImportPath(mt.Schema); imp != "" {
					imports[imp] = true
				}
			}
		}
	}
	// Responses — only collect imports for success codes (2xx)
	for code, resp := range op.Responses {
		var n int
		fmt.Sscanf(code, "%d", &n)
		if n >= 400 {
			continue
		}
		for _, mt := range resp.Content {
			if mt.Schema != nil {
				if imp := g.toImportPath(mt.Schema); imp != "" {
					imports[imp] = true
				}
			}
		}
	}
}

func (g *Generator) buildOperationData(op *Operation, path string, imports map[string]bool) *OperationData {
	od := &OperationData{
		Path:         path,
		Method:       op.Method,
		ID:           op.ID,
		MethodName:   sanitizeMethodName(op.ID),
		Summary:      op.Summary,
		Description:  op.Description,
		Tags:         op.Tags,
		PathParams:   g.buildParams(op.Parameters, "path"),
		QueryParams:  g.buildParams(op.Parameters, "query"),
		HeaderParams: g.buildParams(op.Parameters, "header"),
	}
	if od.MethodName == "" {
		od.MethodName = camelCase(op.Method)
	}

	// Request body
	if op.RequestBody != nil {
		rb := g.buildRequestBody(op.RequestBody, imports)
		if rb != nil {
			od.RequestBody = rb
		}
	}

	// Response body — collect JSON responses from success codes
	successCodes := g.successCodes(op)
	rbd := g.buildResponseBody(op, successCodes, imports)
	if rbd != nil {
		od.ResponseBody = rbd
		od.SuccessCodes = successCodes
	}

	return od
}

func (g *Generator) buildParams(params []Parameter, paramType string) []ParameterData {
	var result []ParameterData
	for _, p := range params {
		if p.In != paramType {
			continue
		}
		gd := ParameterData{
			Name:        p.Name,
			In:          p.In,
			Required:    p.Required,
			Description: p.Description,
		}
		if p.Schema != nil {
			gd.GoType = g.schemaToGoType(p.Schema)
			if imp := g.toImportPath(p.Schema); imp != "" {
				gd.ImportPath = imp
			}
		} else {
			gd.GoType = "string"
		}
		result = append(result, gd)
	}
	return result
}

func (g *Generator) buildRequestBody(rb *RequestBody, imports map[string]bool) *RequestBodyData {
	// Find JSON content type
	for ct, mt := range rb.Content {
		if !strings.Contains(ct, "json") {
			continue
		}
		if mt.Schema == nil {
			continue
		}
		schemaName := mt.Schema.Name
		if schemaName == "" && mt.Schema.Ref != "" {
			schemaName = strings.TrimPrefix(mt.Schema.Ref, "#/components/schemas/")
		}
		if schemaName == "" {
			continue
		}
		importPath := g.toImportPath(mt.Schema)
		if importPath != "" {
			imports[importPath] = true
		}
		return &RequestBodyData{
			Required:    rb.Required,
			Description: rb.Description,
			SchemaName:  schemaName,
			GoType:      g.schemaToGoType(mt.Schema),
			ImportPath:  importPath,
		}
	}
	return nil
}

func (g *Generator) buildResponseBody(op *Operation, successCodes []string, imports map[string]bool) *ResponseBodyData {
	if len(successCodes) == 0 {
		return nil
	}

	fields := make(map[string]*ResponseField)

	for _, code := range successCodes {
		resp, ok := op.Responses[code]
		if !ok {
			continue
		}
		// Find JSON content type
		for ct, mt := range resp.Content {
			if !strings.Contains(ct, "json") || mt.Schema == nil {
				continue
			}
			schemaName := mt.Schema.Name
			if schemaName == "" && mt.Schema.Ref != "" {
				schemaName = strings.TrimPrefix(mt.Schema.Ref, "#/components/schemas/")
			}
			if schemaName == "" {
				continue
			}
			importPath := g.toImportPath(mt.Schema)
			if importPath != "" {
				imports[importPath] = true
			}
			statusInt := 0
			fmt.Sscanf(code, "%d", &statusInt)
			fields[code] = &ResponseField{
				StatusCode: statusInt,
				SchemaName: schemaName,
				GoType:     g.schemaToGoType(mt.Schema),
				ImportPath: importPath,
			}
			break
		}
	}

	if len(fields) == 0 {
		return nil
	}

	return &ResponseBodyData{Fields: fields}
}

func (g *Generator) successCodes(op *Operation) []string {
	var codes []string
	for code := range op.Responses {
		var n int
		fmt.Sscanf(code, "%d", &n)
		if n >= 200 && n < 300 {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	return codes
}

// hasJSONResponse checks if an operation has a JSON response.
func (g *Generator) hasJSONResponse(op *Operation, code string) bool {
	resp, ok := op.Responses[code]
	if !ok {
		return false
	}
	for ct := range resp.Content {
		if strings.Contains(ct, "json") {
			return true
		}
	}
	return false
}

// jsonSchema returns the JSON schema for a given response code.
func (g *Generator) jsonSchema(op *Operation, code string) *Schema {
	resp, ok := op.Responses[code]
	if !ok {
		return nil
	}
	for ct, mt := range resp.Content {
		if strings.Contains(ct, "json") && mt.Schema != nil {
			return mt.Schema
		}
	}
	return nil
}

// requestSchema returns the JSON request body schema.
func (g *Generator) requestSchema(op *Operation) *Schema {
	if op.RequestBody == nil {
		return nil
	}
	for ct, mt := range op.RequestBody.Content {
		if strings.Contains(ct, "json") && mt.Schema != nil {
			return mt.Schema
		}
	}
	return nil
}

// pathParams returns path parameters.
func (g *Generator) pathParams(op *Operation) []Parameter {
	return g.filterParams(op.Parameters, "path")
}

// queryParams returns query parameters.
func (g *Generator) queryParams(op *Operation) []Parameter {
	return g.filterParams(op.Parameters, "query")
}

// headerParams returns header parameters.
func (g *Generator) headerParams(op *Operation) []Parameter {
	return g.filterParams(op.Parameters, "header")
}

func (g *Generator) filterParams(params []Parameter, paramType string) []Parameter {
	var result []Parameter
	for _, p := range params {
		if p.In == paramType {
			result = append(result, p)
		}
	}
	return result
}

// responseTypeName generates a Go type name for a response container.
func (g *Generator) responseTypeName(op *Operation) string {
	base := op.ID
	if base == "" {
		base = camelCase(op.Method) + "Root"
	}
	return base + "Response"
}

// toGoType converts an OpenAPI type to a Go type.
func (g *Generator) toGoType(s *Schema) string {
	return g.schemaToGoType(s)
}

// toGoName converts a name to a valid Go identifier (exported).
func toGoName(name string) string {
	return toGoTypeName(name)
}

// toGoTypeName converts a schema name to a Go type name.
func toGoTypeName(name string) string {
	if name == "" {
		return "Object"
	}
	// Handle names like "01_basic_types.User"
	parts := strings.Split(name, ".")
	last := parts[len(parts)-1]
	return camelCase(last)
}

// prefixWithTypeAlias returns the type name prefixed with the package alias
// if the schema name contains a package prefix (e.g. "01_basic_types.User").
func (g *Generator) prefixWithTypeAlias(name string) string {
	if name == "" {
		return "Object"
	}
	parts := strings.Split(name, ".")
	typeName := camelCase(parts[len(parts)-1])
	if len(parts) < 2 {
		return typeName
	}
	// Find the import path for this package prefix
	pkgPrefix := parts[0]
	for impPath, alias := range g.packageAliases {
		if strings.HasSuffix(impPath, "/"+pkgPrefix) {
			return alias + "." + typeName
		}
	}
	return typeName
}

// toImportPath returns the Go import path for a schema.
// First checks explicit model map, then falls back to base import.
func (g *Generator) toImportPath(s *Schema) string {
	if s == nil || s.Name == "" {
		return ""
	}
	// Schema names are like "01_basic_types.User"
	parts := strings.Split(s.Name, ".")
	if len(parts) < 2 {
		return ""
	}
	pkg := parts[len(parts)-2]
	// Check explicit model map first
	for _, mapping := range g.cfg.ModelMap {
		kv := strings.SplitN(mapping, "=", 2)
		if len(kv) == 2 && kv[0] == pkg {
			return kv[1]
		}
	}
	// Fall back to base import
	base := g.cfg.BaseImport
	if g.cfg.ModelsDir != "" {
		base = filepath.Join(base, g.cfg.ModelsDir)
	}
	return filepath.Join(base, pkg)
}

// schemaToGoType converts an OpenAPI schema to a Go type string.
func (g *Generator) schemaToGoType(s *Schema) string {
	if s == nil {
		return "any"
	}

	// $ref — resolve to actual type name
	if s.Ref != "" {
		name := strings.TrimPrefix(s.Ref, "#/components/schemas/")
		if name == "" {
			name = s.Name
		}
		return g.prefixWithTypeAlias(name)
	}

	switch s.Type {
	case "object":
		if s.Name != "" {
			return g.prefixWithTypeAlias(s.Name)
		}
		return "map[string]any"
	case "array":
		if s.Items != nil {
			elem := g.schemaToGoType(s.Items)
			return "[]" + elem
		}
		return "[]any"
	case "string":
		return "string"
	case "integer":
		if s.Format == "int64" {
			return "int64"
		}
		if s.Format == "int32" {
			return "int32"
		}
		return "int"
	case "number":
		if s.Format == "float" {
			return "float32"
		}
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "any"
	}
}

// sanitizeMethodName removes characters invalid for Go identifiers
// and ensures the name is exported (starts with uppercase).
func sanitizeMethodName(s string) string {
	// Remove : and other invalid chars
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
	// Ensure exported
	if name[0] >= 'a' && name[0] <= 'z' {
		name = strings.ToUpper(string(name[0])) + name[1:]
	}
	return name
}

// sanitizePackageName ensures the package name is a valid Go identifier.
// It removes digits from the start, replaces invalid chars, and lowercases.
func sanitizePackageName(s string) string {
	// Remove invalid characters
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
	// Remove leading digits
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
	// Split on both _ and -
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

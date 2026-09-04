package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FieldParameterInfo describes a struct field annotated with @oa:in.
type FieldParameterInfo struct {
	Name        string // Go field name
	In          string // "query", "path", "header"
	ParamName   string // the actual parameter name (e.g. "X-API-Key", "userId")
	Required    bool   // path params are always required
	GoType      string
	Description string
}

// parseModelFile finds the Go file for the given import path and typeName,
// then extracts @oa:in annotations from struct fields.
// It returns query, header, path params and a boolean indicating whether
// the struct has body fields (fields without @oa:in).
func parseModelFile(importPath, typeName string, pkgDir string) ([]FieldParameterInfo, []FieldParameterInfo, []FieldParameterInfo, bool) {
	modelFile := findModelFile(pkgDir)
	if modelFile == "" {
		return nil, nil, nil, false
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, modelFile, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, false
	}

	var queryParams, headerParams, pathParams []FieldParameterInfo
	hasBody := false

	reOAIn := regexp.MustCompile(`@oa:in\s+(\w+)(?:\s+(\S+))?`)
	reOADesc := regexp.MustCompile(`@oa:description\s+"([^"]*)"`)

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				continue
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue // embedded field
				}

				fieldName := field.Names[0].Name
				fieldType := exprToName(field.Type)

				// Check comments for @oa:in — field Doc comments (before the field)
				var in string
				var paramName string
				var desc string
				if field.Doc != nil {
					for _, c := range field.Doc.List {
						if m := reOAIn.FindStringSubmatch(c.Text); len(m) >= 2 {
							in = m[1]
							paramName = m[2]
						}
						if m := reOADesc.FindStringSubmatch(c.Text); len(m) >= 2 {
							desc = m[1]
						}
					}
				}

				if in == "" {
					hasBody = true
					continue
				}

				if paramName == "" {
					paramName = strings.ToLower(fieldName)
				}

				fpi := FieldParameterInfo{
					Name:        fieldName,
					In:          in,
					ParamName:   paramName,
					GoType:      fieldType,
					Description: desc,
				}

				switch in {
				case "query":
					queryParams = append(queryParams, fpi)
				case "header":
					headerParams = append(headerParams, fpi)
				case "path":
					fpi.Required = true
					pathParams = append(pathParams, fpi)
				}
			}
		}
	}

	return queryParams, headerParams, pathParams, hasBody
}

// findModelFile tries to locate the Go source file for a given import path.
func findModelFile(pkgDir string) string {
	absPkgDir, _ := filepath.Abs(pkgDir)
	root := findProjectRoot(absPkgDir)
	if root == "" {
		return ""
	}

	// Extract the example name from pkgDir (last segment after elval-integration/)
	parts := strings.Split(absPkgDir, string(filepath.Separator))
	var exampleName string
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "elval-integration" && i+1 < len(parts) {
			exampleName = parts[i+1]
			break
		}
	}
	if exampleName == "" {
		exampleName = parts[len(parts)-1]
	}

	modelsDir := filepath.Join(root, "examples", "models", exampleName)
	if entries, err := os.ReadDir(modelsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") && !strings.HasSuffix(e.Name(), ".gen.go") {
				return filepath.Join(modelsDir, e.Name())
			}
		}
	}

	return ""
}

// exprToName converts an AST expression to a Go type name string.
func exprToName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToName(e.X)
	case *ast.ArrayType:
		return "[]" + exprToName(e.Elt)
	case *ast.MapType:
		return "map[" + exprToName(e.Key) + "]" + exprToName(e.Value)
	case *ast.SelectorExpr:
		return exprToName(e.X) + "." + e.Sel.Name
	case *ast.IndexExpr:
		return exprToName(e.X) + "[" + exprToName(e.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, len(e.Indices))
		for i, idx := range e.Indices {
			parts[i] = exprToName(idx)
		}
		return exprToName(e.X) + "[" + strings.Join(parts, ", ") + "]"
	default:
		return "any"
	}
}

// findProjectRoot walks up from dir looking for go.mod.
func findProjectRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

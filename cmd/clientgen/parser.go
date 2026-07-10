package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PackageInfo holds everything extracted from a Go package's main.go.
type PackageInfo struct {
	Title   string
	Version string
	Routes  []RouteInfo
	Imports map[string]string // alias -> import path (non-stdlib, non-nooa)
}

// RouteInfo holds extracted info for one NewRoute call.
type RouteInfo struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Tags        []string
	Security    []string
	ReqType     string // alias.TypeName
	ResType     string // alias.TypeName
	ReqImport   string
	ResImport   string
	HasReqBody  bool
	HasRespBody bool
}

// parsePackage parses main.go in the given directory and extracts route/spec info.
func ParsePackage(pkgPath string) (*PackageInfo, error) {
	mainFile := findMainFile(pkgPath)
	if mainFile == "" {
		return nil, fmt.Errorf("no main.go found in %s", pkgPath)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, mainFile, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", mainFile, err)
	}

	info := &PackageInfo{
		Imports: make(map[string]string),
	}

	// Extract imports
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if isStdlib(path) || path == "github.com/arkannsk/nooa" {
			continue
		}
		alias := imp.Name.String()
		if alias == "" || alias == "_" {
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		info.Imports[alias] = path
	}

	// Extract spec info
	extractSpecInfo(node, info)

	// Extract routes with chained metadata
	extractRoutes(node, info)

	return info, nil
}

func findMainFile(dir string) string {
	path := filepath.Join(dir, "main.go")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*", "main.go"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func isStdlib(path string) bool {
	return !strings.Contains(path, "/")
}

// extractSpecInfo finds nooa.NewSpec(nooa.Info{...}).
func extractSpecInfo(file *ast.File, info *PackageInfo) {
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewSpec" {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "nooa" {
			return true
		}
		cl, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			val, ok := kv.Value.(*ast.BasicLit)
			if !ok {
				continue
			}
			str := strings.Trim(val.Value, `"`)
			switch key.Name {
			case "Title":
				info.Title = str
			case "Version":
				info.Version = str
			}
		}
		return true
	})
}

// extractRoutes finds all NewRoute[Req, Res] calls and their chained metadata.
func extractRoutes(file *ast.File, info *PackageInfo) {
	// Collect all NewRoute calls with their positions
	type routeCandidate struct {
		route RouteInfo
		pos   token.Pos
	}
	var candidates []routeCandidate

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Match nooa.NewRoute[Req, Res](...)
		idx, ok := call.Fun.(*ast.IndexListExpr)
		if !ok {
			return true
		}
		sel, ok := idx.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewRoute" {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "nooa" {
			return true
		}
		if len(idx.Indices) < 2 || len(call.Args) < 2 {
			return true
		}

		methodLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		pathLit, ok := call.Args[1].(*ast.BasicLit)
		if !ok {
			return true
		}

		method := strings.ToUpper(strings.Trim(methodLit.Value, `"`))
		path := strings.Trim(pathLit.Value, `"`)

		reqType, reqImport := resolveType(idx.Indices[0], info)
		resType, resImport := resolveType(idx.Indices[1], info)

		candidates = append(candidates, routeCandidate{
			route: RouteInfo{
				Method:      method,
				Path:        path,
				OperationID: defaultOperationID(method, path),
				ReqType:     reqType,
				ResType:     resType,
				ReqImport:   reqImport,
				ResImport:   resImport,
				HasReqBody:  method != "GET" && method != "HEAD" && method != "DELETE",
				HasRespBody: true,
			},
			pos: call.Lparen,
		})
		return true
	})

	// Second pass: find chained metadata calls and match to routes by position.
	// A chained call like NewRoute[...](...).Summary("...").RegisterSpecAndMux(...)
	// is a *ast.CallExpr where Fun is *ast.SelectorExpr, and X chains back.
	// We trace the X chain to find the original NewRoute call position.

	type chainMeta struct {
		Summary   string
		Tags      []string
		Security  []string
		OpID      string
	}
	metaByPos := make(map[token.Pos]*chainMeta)

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Trace back through X chain to find the original NewRoute call
		origPos := traceToNewRoute(sel.X)
		if origPos == 0 {
			return true
		}

		cm, ok := metaByPos[origPos]
		if !ok {
			cm = &chainMeta{}
			metaByPos[origPos] = cm
		}

		// Extract argument string(s)
		if len(call.Args) > 0 {
			switch sel.Sel.Name {
			case "Summary":
				if lit, ok := call.Args[0].(*ast.BasicLit); ok {
					cm.Summary = strings.Trim(lit.Value, `"`)
				}
			case "Tags":
				for _, arg := range call.Args {
					if lit, ok := arg.(*ast.BasicLit); ok {
						cm.Tags = append(cm.Tags, strings.Trim(lit.Value, `"`))
					}
				}
			case "Secure":
				if lit, ok := call.Args[0].(*ast.BasicLit); ok {
					cm.Security = append(cm.Security, strings.Trim(lit.Value, `"`))
				}
			case "OperationID":
				if lit, ok := call.Args[0].(*ast.BasicLit); ok {
					cm.OpID = strings.Trim(lit.Value, `"`)
				}
			}
		}
		return true
	})

	// Apply metadata to routes
	for _, c := range candidates {
		if cm, ok := metaByPos[c.pos]; ok {
			if cm.Summary != "" {
				c.route.Summary = cm.Summary
			}
			if len(cm.Tags) > 0 {
				c.route.Tags = cm.Tags
			}
			if len(cm.Security) > 0 {
				c.route.Security = cm.Security
			}
			if cm.OpID != "" {
				c.route.OperationID = cm.OpID
			}
		}
		info.Routes = append(info.Routes, c.route)
	}
}

// traceToNewRoute follows the X chain of selector expressions back to the
// original nooa.NewRoute[...] call and returns its Lparen position.
// Returns 0 if no NewRoute call is found.
func traceToNewRoute(expr ast.Node) token.Pos {
	switch n := expr.(type) {
	case *ast.CallExpr:
		// Check if this IS the NewRoute call
		idx, ok := n.Fun.(*ast.IndexListExpr)
		if !ok {
			return 0
		}
		sel, ok := idx.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewRoute" {
			return 0
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != "nooa" {
			return 0
		}
		return n.Lparen
	case *ast.SelectorExpr:
		return traceToNewRoute(n.X)
	default:
		return 0
	}
}

// resolveType extracts the type name and import path from a type expression.
func resolveType(expr ast.Expr, info *PackageInfo) (string, string) {
	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type — check if it's an import alias
		if impPath, ok := info.Imports[t.Name]; ok {
			return t.Name, impPath
		}
		return t.Name, ""
	case *ast.SelectorExpr:
		alias, ok := t.X.(*ast.Ident)
		if !ok {
			return t.Sel.Name, ""
		}
		impPath, ok := info.Imports[alias.Name]
		if !ok {
			return t.Sel.Name, ""
		}
		return alias.Name + "." + t.Sel.Name, impPath
	default:
		return "", ""
	}
}

func defaultOperationID(method, path string) string {
	method = strings.ToUpper(method)
	path = strings.Trim(path, "/ ")
	if path == "" {
		return method + "Root"
	}
	var sb strings.Builder
	sb.WriteString(method)
	for _, seg := range strings.Split(path, "/") {
		seg = strings.Trim(seg, "{} ")
		if seg == "" {
			continue
		}
		sb.WriteString(strings.ToUpper(seg[:1]))
		if len(seg) > 1 {
			sb.WriteString(seg[1:])
		}
	}
	return sb.String()
}

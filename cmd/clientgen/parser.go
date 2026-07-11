package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PackageInfo holds everything extracted from a Go package's main.go.
type PackageInfo struct {
	Title   string
	Version string
	Routes  []RouteInfo
	Imports map[string]string // alias -> import path (non-stdlib, non-nooa)
}

// RouteInfo holds extracted info for one NewRoute or NewRouteMultiResp call.
type RouteInfo struct {
	Method              string
	Path                string
	OperationID         string
	Summary             string
	Tags                []string
	Security            []string
	ReqType             string // alias.TypeName
	ResType             string // alias.TypeName (for NewRoute) or empty (for MultiResp)
	ReqImport           string
	ResImport           string
	HasReqBody          bool
	HasRespBody         bool
	ResponseStatuses    []int  // status codes from OnSuccess/OnNoContent/Response
	ResponseContentTypes map[int][]string
	ResponseSchemas     map[int]string // status -> schema name (from .Response())
	// MultiRespEntries holds per-status schema info for NewRouteMultiResp
	MultiRespEntries    []MultiRespEntry
}

// MultiRespEntry holds one ResponseEntry from NewRouteMultiResp.
type MultiRespEntry struct {
	Status       int
	SchemaName   string
	ImportPath   string
	ContentTypes []string
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

// extractRoutes finds all NewRoute[Req, Res] and NewRouteMultiResp[Req] calls
// and their chained metadata.
func extractRoutes(file *ast.File, info *PackageInfo) {
	// Collect all route calls with their positions
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

		// Need at least 2 args (method, path) for NewRoute
		if len(call.Args) < 2 {
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

		// Match nooa.NewRoute[Req, Res](...)
		if idx, ok := call.Fun.(*ast.IndexListExpr); ok {
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
		}

		// Match nooa.NewRouteMultiResp[Req](..., ResponseEntry{...}, ...)
		if idx, ok := call.Fun.(*ast.IndexListExpr); ok {
			sel, ok := idx.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "NewRouteMultiResp" {
				return true
			}
			id, ok := sel.X.(*ast.Ident)
			if !ok || id.Name != "nooa" {
				return true
			}
			if len(idx.Indices) < 1 || len(call.Args) < 3 {
				return true
			}

			reqType, reqImport := resolveType(idx.Indices[0], info)

			// Parse ResponseEntry composite literals from args[2:]
			var entries []MultiRespEntry
			var respStatuses []int
			respSchemas := make(map[int]string)
			respCTs := make(map[int][]string)

			for _, arg := range call.Args[2:] {
				cl, ok := arg.(*ast.CompositeLit)
				if !ok {
					continue
				}
				var entry MultiRespEntry
				for _, elt := range cl.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok {
						continue
					}
					switch key.Name {
					case "Status":
						if lit, ok := kv.Value.(*ast.BasicLit); ok {
							if s, err := strconv.Atoi(lit.Value); err == nil {
								entry.Status = s
							}
						}
					case "Instance":
						// new(SomeType) — extract type name
						if newCall, ok := kv.Value.(*ast.CallExpr); ok {
							if ident, ok := newCall.Fun.(*ast.Ident); ok {
								entry.SchemaName = ident.Name
								entry.ImportPath = resolveImportForType(ident.Name, info)
							}
							// SelectorExpr: alias.TypeName
							if sel, ok := newCall.Fun.(*ast.SelectorExpr); ok {
								alias := sel.X.(*ast.Ident)
								if impPath, ok2 := info.Imports[alias.Name]; ok2 {
									entry.SchemaName = alias.Name + "." + sel.Sel.Name
									entry.ImportPath = impPath
								}
							}
						}
					case "Desc":
						// skip
					case "ContentTypes":
						if list, ok := kv.Value.(*ast.CompositeLit); ok {
							for _, e := range list.Elts {
								if lit, ok := e.(*ast.BasicLit); ok {
									entry.ContentTypes = append(entry.ContentTypes, strings.Trim(lit.Value, `"`))
								}
							}
						}
					}
				}
				if entry.Status > 0 {
					entries = append(entries, entry)
					respStatuses = append(respStatuses, entry.Status)
					respSchemas[entry.Status] = entry.SchemaName
					if len(entry.ContentTypes) > 0 {
						respCTs[entry.Status] = entry.ContentTypes
					}
				}
			}

			candidates = append(candidates, routeCandidate{
				route: RouteInfo{
					Method:              method,
					Path:                path,
					OperationID:         defaultOperationID(method, path),
					ReqType:             reqType,
					ReqImport:           reqImport,
					HasReqBody:          method != "GET" && method != "HEAD" && method != "DELETE",
					HasRespBody:         len(entries) > 0,
					ResponseStatuses:    respStatuses,
					ResponseSchemas:     respSchemas,
					ResponseContentTypes: respCTs,
					MultiRespEntries:    entries,
				},
				pos: call.Lparen,
			})
			return true
		}
		return true
	})

	// Second pass: find chained metadata calls and match to routes by position.
	// A chained call like NewRoute[...](...).Summary("...").RegisterSpecAndMux(...)
	// is a *ast.CallExpr where Fun is *ast.SelectorExpr, and X chains back.
	// We trace the X chain to find the original NewRoute call position.

	type chainMeta struct {
		Summary           string
		Tags              []string
		Security          []string
		OpID              string
		ResponseStatuses  []int
		ResponseContentTypes map[int][]string
		ResponseSchemas   map[int]string
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
			case "OnSuccess", "OnNoContent":
				// OnSuccess(status, desc, ct...)
				if len(call.Args) >= 1 {
					if lit, ok := call.Args[0].(*ast.BasicLit); ok {
						status, err := strconv.Atoi(strings.Trim(lit.Value, `"`))
						if err == nil {
							cm.ResponseStatuses = append(cm.ResponseStatuses, status)
							if cm.ResponseContentTypes == nil {
								cm.ResponseContentTypes = make(map[int][]string)
							}
							// Collect ContentTypes from optional args (after status and desc)
							for _, arg := range call.Args[2:] {
								if lit, ok := arg.(*ast.BasicLit); ok {
									cm.ResponseContentTypes[status] = append(cm.ResponseContentTypes[status], strings.Trim(lit.Value, `"`))
								}
							}
						}
					}
				}
			case "Response":
				// Response(status, schemaName, desc, ct...)
				if len(call.Args) >= 2 {
					if lit, ok := call.Args[0].(*ast.BasicLit); ok {
						status, err := strconv.Atoi(strings.Trim(lit.Value, `"`))
						if err == nil {
							cm.ResponseStatuses = append(cm.ResponseStatuses, status)
							if cm.ResponseContentTypes == nil {
								cm.ResponseContentTypes = make(map[int][]string)
							}
							// schemaName is arg[1]
							if lit2, ok := call.Args[1].(*ast.BasicLit); ok {
								if cm.ResponseSchemas == nil {
									cm.ResponseSchemas = make(map[int]string)
								}
								cm.ResponseSchemas[status] = strings.Trim(lit2.Value, `"`)
							}
							// ContentTypes from arg[3:]
							for _, arg := range call.Args[3:] {
								if lit, ok := arg.(*ast.BasicLit); ok {
									cm.ResponseContentTypes[status] = append(cm.ResponseContentTypes[status], strings.Trim(lit.Value, `"`))
								}
							}
						}
					}
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
			if len(cm.ResponseStatuses) > 0 {
				c.route.ResponseStatuses = cm.ResponseStatuses
				c.route.ResponseContentTypes = cm.ResponseContentTypes
				c.route.ResponseSchemas = cm.ResponseSchemas
			}
		}
		info.Routes = append(info.Routes, c.route)
	}
}

// traceToNewRoute follows the X chain of selector expressions back to the
// original nooa.NewRoute[...] or nooa.NewRouteMultiResp[...] call and returns
// its Lparen position. Returns 0 if no route call is found.
func traceToNewRoute(expr ast.Node) token.Pos {
	switch n := expr.(type) {
	case *ast.CallExpr:
		// Check if this IS a route call (NewRoute or NewRouteMultiResp)
		if idx, ok := n.Fun.(*ast.IndexListExpr); ok {
			if sel, ok := idx.X.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "nooa" {
					if sel.Sel.Name == "NewRoute" || sel.Sel.Name == "NewRouteMultiResp" {
						return n.Lparen
					}
				}
			}
		}
		// If it's a chained call like NewRoute[...](...).Summary("..."),
		// trace through the selector's X
		if sel2, ok := n.Fun.(*ast.SelectorExpr); ok {
			return traceToNewRoute(sel2.X)
		}
		return 0
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

// resolveImportForType finds the import path for a given type name by
// scanning all known imports. Used for parsing Instance fields in ResponseEntry.
func resolveImportForType(typeName string, info *PackageInfo) string {
	for _, impPath := range info.Imports {
		parts := strings.Split(impPath, "/")
		pkgName := parts[len(parts)-1]
		// If the type name matches the package name (it's an alias), return the path
		if pkgName == typeName {
			return impPath
		}
	}
	return ""
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

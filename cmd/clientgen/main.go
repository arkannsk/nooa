package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds CLI flags for the generator.
type Config struct {
	SpecFile    string // path to OpenAPI JSON file
	Output      string // output file path
	Package     string // Go package name for the generated client
	BaseImport  string // base import path for models (e.g. "github.com/myproject/pkg/models")
	ModelsDir   string // subdirectory appended to base import (e.g. "models" for models/pkg)
	PkgOverride string // override package name (if not specified, derived from title)
	ModelMap    []string // explicit pkg=import_path mappings
}

func main() {
	cfg := &Config{}

	flag.StringVar(&cfg.SpecFile, "spec", "", "path to OpenAPI 3.x JSON spec file")
	flag.StringVar(&cfg.Output, "out", "client.go", "output Go file path")
	flag.StringVar(&cfg.Package, "package", "client", "Go package name (default: derived from title)")
	flag.StringVar(&cfg.BaseImport, "base", "", "base import path for model packages")
	flag.StringVar(&cfg.ModelsDir, "models-dir", "", "subdirectory appended to base import for model paths")
	flag.StringVar(&cfg.PkgOverride, "pkg-name", "", "override Go package name")
	var modelMapStr string
	flag.StringVar(&modelMapStr, "model-map", "", "explicit pkg=path mappings (comma-separated, e.g. 'models=github.com/proj/models,types=github.com/proj/types')")
	flag.Parse()

	// Parse model-map into slice
	if modelMapStr != "" {
		for _, m := range strings.Split(modelMapStr, ",") {
			cfg.ModelMap = append(cfg.ModelMap, strings.TrimSpace(m))
		}
	}

	if cfg.SpecFile == "" {
		fmt.Fprintln(os.Stderr, "error: -spec is required")
		flag.Usage()
		os.Exit(1)
	}
	if cfg.BaseImport == "" {
		fmt.Fprintln(os.Stderr, "error: -base is required (base import path for models)")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(cfg.SpecFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading spec: %v\n", err)
		os.Exit(1)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing spec: %v\n", err)
		os.Exit(1)
	}

	ver, _ := spec["openapi"].(string)
	if ver == "" {
		fmt.Fprintln(os.Stderr, "error: not a valid OpenAPI 3.x spec")
		os.Exit(1)
	}

	info, _ := spec["info"].(map[string]any)
	title, _ := info["title"].(string)
	version, _ := info["version"].(string)

	// Apply package name override or sanitize from title
	if cfg.PkgOverride != "" {
		cfg.Package = cfg.PkgOverride
	} else if cfg.Package == "client" && title != "" {
		// Auto-derive package name from title if no override
		cfg.Package = sanitizePackageName(title)
	}

	g := NewGenerator(spec, cfg, title, version)

	out, err := g.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generation error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(cfg.Output, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("generated %s (%s v%s)\n", cfg.Output, title, version)
}

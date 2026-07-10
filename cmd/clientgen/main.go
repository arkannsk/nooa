package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds CLI flags for the generator.
type Config struct {
	PkgPath string // path to the Go package containing main.go (e.g. ./examples/01_basic_types)
	Output  string // output file path
	Package string // Go package name for the generated client
}

func main() {
	cfg := &Config{}

	flag.StringVar(&cfg.PkgPath, "pkg", ".", "path to the Go package to generate client for")
	flag.StringVar(&cfg.Output, "out", "client.go", "output Go file path")
	flag.StringVar(&cfg.Package, "package", "client", "Go package name (default: derived from title)")
	flag.Parse()

	if cfg.PkgPath == "" {
		fmt.Fprintln(os.Stderr, "error: -pkg is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the package AST and extract routes, imports, spec info
	info, err := ParsePackage(cfg.PkgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	// Derive package name from title if not overridden
	if cfg.Package == "client" && info.Title != "" {
		cfg.Package = sanitizePackageName(info.Title)
	}

	g := NewGenerator(info, cfg)

	out, err := g.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generation error: %v\n", err)
		os.Exit(1)
	}

	// Ensure output directory exists
	if dir := filepath.Dir(cfg.Output); dir != "." && dir != "/" {
		os.MkdirAll(dir, 0755)
	}

	if err := os.WriteFile(cfg.Output, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("generated %s (%s v%s)\n", cfg.Output, info.Title, info.Version)
}

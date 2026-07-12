package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Config holds CLI flags for the generator.
type Config struct {
	PkgPath string // path to the Go package containing main.go (e.g. ./examples/01_basic_types)
	Output  string // output file path or directory (if ends with /, treated as directory)
	Package string // Go package name for the generated client
}

func main() {
	cfg := &Config{}

	flag.StringVar(&cfg.PkgPath, "pkg", ".", "path to the Go package to generate client for")
	flag.StringVar(&cfg.Output, "out", "client.go", "output file path or directory")
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

	files, err := g.GenerateFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generation error: %v\n", err)
		os.Exit(1)
	}

	// Determine output directory
	outDir := cfg.Output
	if !filepath.IsAbs(outDir) && !stringsHasSuffix(outDir, "/") {
		// If output doesn't end with /, treat it as a directory if multiple files
		if len(files) > 1 {
			outDir = filepath.Dir(cfg.Output)
			if outDir == "." {
				outDir = cfg.Output
				// Actually if it's like "client.go" and multiple files, use dir
				outDir = filepath.Dir(cfg.Output)
				if outDir == "." {
					outDir = "."
				}
			}
		}
	}

	// If output ends with /, it's a directory
	if stringsHasSuffix(cfg.Output, "/") {
		outDir = cfg.Output
	}

	// Sort filenames for deterministic output
	filenames := make([]string, 0, len(files))
	for f := range files {
		filenames = append(filenames, f)
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		content := files[filename]
		path := filepath.Join(outDir, filename)

		if dir := filepath.Dir(path); dir != "." && dir != "/" {
			os.MkdirAll(dir, 0755)
		}

		if err := os.WriteFile(path, content, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("generated %s\n", path)
	}
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

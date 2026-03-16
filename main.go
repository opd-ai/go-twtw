// go-twtw – Go The Way Things Work
//
// A deterministic Go package visualization tool that represents Go constructs
// as illustrated physical machines in the style of David Macaulay's
// The Way Things Work and Sierra's The Incredible Machine.
//
// Usage:
//
//	go-twtw [flags] <package-dir>
//
// Flags:
//
//	-format string    Output format: text or json (default "text")
//	-output string    Directory to write SVG files into (default ".")
//	-no-svg           Skip SVG generation
//	-recursive        Walk sub-packages recursively
//
// Examples:
//
//	go-twtw ./mypackage
//	go-twtw -format json -output ./diagrams ./mypackage
//	go-twtw -recursive ./myproject
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/opd-ai/go-twtw/analyzer"
	"github.com/opd-ai/go-twtw/report"
	"github.com/opd-ai/go-twtw/svg"
)

func main() {
	var (
		format    = flag.String("format", "text", "Output format: text or json")
		outputDir = flag.String("output", ".", "Directory to write SVG files into")
		noSVG     = flag.Bool("no-svg", false, "Skip SVG generation")
		recursive = flag.Bool("recursive", false, "Analyze sub-packages recursively")
	)

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "go-twtw – Go: The Way Things Work")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Visualizes Go packages as physical machine diagrams.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  go-twtw [flags] <package-dir>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  go-twtw ./mypackage")
		fmt.Fprintln(os.Stderr, "  go-twtw -format json -output ./diagrams ./mypackage")
		fmt.Fprintln(os.Stderr, "  go-twtw -recursive ./myproject")
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	dir := flag.Arg(0)

	// ── Analysis ────────────────────────────────────────────────────────────
	var (
		pkgs     []*analyzer.PackageInfo
		analyErr error
	)
	if *recursive {
		pkgs, analyErr = analyzer.AnalyzeRecursive(dir)
	} else {
		var pkg *analyzer.PackageInfo
		pkg, analyErr = analyzer.Analyze(dir)
		if analyErr == nil {
			pkgs = []*analyzer.PackageInfo{pkg}
		}
	}
	if analyErr != nil {
		log.Fatalf("go-twtw: analysis error: %v", analyErr)
	}
	if len(pkgs) == 0 {
		log.Fatal("go-twtw: no Go packages found in the specified directory")
	}

	// ── Report ──────────────────────────────────────────────────────────────
	r := report.Generate(pkgs)

	switch *format {
	case "json":
		if err := report.RenderJSON(r, os.Stdout); err != nil {
			log.Fatalf("go-twtw: JSON render error: %v", err)
		}
	default:
		if err := report.RenderText(r, os.Stdout); err != nil {
			log.Fatalf("go-twtw: text render error: %v", err)
		}
	}

	// ── SVG generation ──────────────────────────────────────────────────────
	if !*noSVG {
		if err := os.MkdirAll(*outputDir, 0o755); err != nil {
			log.Fatalf("go-twtw: cannot create output directory %q: %v", *outputDir, err)
		}

		for _, pkg := range pkgs {
			if pkg.Name == "" {
				continue
			}
			svgContent := svg.GenerateSVG(pkg)
			filename := filepath.Join(*outputDir, pkg.Name+"-diagram.svg")
			if err := os.WriteFile(filename, []byte(svgContent), 0o644); err != nil {
				log.Fatalf("go-twtw: writing SVG %q: %v", filename, err)
			}
			fmt.Fprintf(os.Stderr, "  SVG written: %s\n", filename)
		}
	}
}

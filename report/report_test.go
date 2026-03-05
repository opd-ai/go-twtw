package report_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/opd-ai/go-twtw/analyzer"
	"github.com/opd-ai/go-twtw/report"
)

// buildSimplePkg returns a minimal PackageInfo for testing.
func buildSimplePkg() *analyzer.PackageInfo {
	return &analyzer.PackageInfo{
		Name:       "example",
		ImportPath: "github.com/example/example",
		Dir:        "/tmp/example",
		Doc:        "Package example provides an example.",
		Imports:    []string{"errors", "fmt"},
		Constructs: []analyzer.ConstructInfo{
			{
				Name:     "MaxItems",
				Kind:     analyzer.KindConst,
				Doc:      "MaxItems is the maximum number of items.",
				Exported: true,
			},
			{
				Name:     "Config",
				Kind:     analyzer.KindStruct,
				Doc:      "Config holds configuration.",
				Exported: true,
				Fields: []analyzer.FieldInfo{
					{Name: "Timeout", Type: "int", Doc: "Timeout in ms."},
					{Name: "Workers", Type: "int", Doc: "Number of workers."},
				},
			},
			{
				Name:      "Processor",
				Kind:      analyzer.KindInterface,
				Doc:       "Processor defines the processing interface.",
				Exported:  true,
				Methods:   []string{"Process", "Reset"},
			},
			{
				Name:     "NewConfig",
				Kind:     analyzer.KindFunction,
				Doc:      "NewConfig creates a new Config.",
				Exported: true,
				Results:  []analyzer.ParamInfo{{Type: "*Config"}},
			},
			{
				Name:             "RunWorker",
				Kind:             analyzer.KindFunction,
				Doc:              "RunWorker starts a worker goroutine.",
				Exported:         true,
				SpawnsGoroutines: true,
				HasChannels:      true,
				Params:           []analyzer.ParamInfo{{Names: []string{"results"}, Type: "chan<- string"}},
			},
			{
				Name:        "Process",
				Kind:        analyzer.KindMethod,
				Doc:         "Process processes the input.",
				Exported:    true,
				Receiver:    "*Config",
				Params:      []analyzer.ParamInfo{{Names: []string{"input"}, Type: "string"}},
				Results:     []analyzer.ParamInfo{{Type: "string"}, {Type: "error"}},
			},
			{
				Name:        "defaultWorkers",
				Kind:        analyzer.KindVar,
				Doc:         "defaultWorkers is the default worker count.",
				Exported:    false,
			},
		},
	}
}

func TestGenerate_NotNil(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})
	if r == nil {
		t.Fatal("Generate returned nil")
	}
	if len(r.Packages) != 1 {
		t.Fatalf("expected 1 package report, got %d", len(r.Packages))
	}
}

func TestGenerate_ComponentInventory(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})
	pr := r.Packages[0]

	if len(pr.Components) != len(pkg.Constructs) {
		t.Errorf("expected %d components, got %d", len(pkg.Constructs), len(pr.Components))
	}

	byName := make(map[string]report.ComponentEntry)
	for _, c := range pr.Components {
		byName[c.Name] = c
	}

	// Const → weight
	if e := byName["MaxItems"]; e.PartKind != "weight" {
		t.Errorf("MaxItems: expected part_kind=weight, got %q", e.PartKind)
	}
	// Struct → container
	if e := byName["Config"]; e.PartKind != "container" {
		t.Errorf("Config: expected part_kind=container, got %q", e.PartKind)
	}
	// Interface → adapter
	if e := byName["Processor"]; e.PartKind != "adapter" {
		t.Errorf("Processor: expected part_kind=adapter, got %q", e.PartKind)
	}
	// Function → gear
	if e := byName["NewConfig"]; e.PartKind != "gear" {
		t.Errorf("NewConfig: expected part_kind=gear, got %q", e.PartKind)
	}
	// Goroutine function → belt
	if e := byName["RunWorker"]; e.PartKind != "belt" {
		t.Errorf("RunWorker: expected part_kind=belt, got %q", e.PartKind)
	}
	// Method → lever
	if e := byName["Process"]; e.PartKind != "lever" {
		t.Errorf("Process: expected part_kind=lever, got %q", e.PartKind)
	}
}

func TestGenerate_DiagramsPresent(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})
	pr := r.Packages[0]
	if len(pr.Diagrams) == 0 {
		t.Fatal("expected at least one diagram specification")
	}
	for _, d := range pr.Diagrams {
		if d.Title == "" {
			t.Error("diagram title should not be empty")
		}
		if d.SVGFile == "" {
			t.Error("diagram SVGFile should not be empty")
		}
		if len(d.Components) == 0 {
			t.Error("diagram should have components")
		}
	}
}

func TestGenerate_CrossRefs(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})
	if len(r.CrossRefs) == 0 {
		t.Error("expected cross-refs from imports")
	}
	for _, ref := range r.CrossRefs {
		if ref.From == "" || ref.To == "" {
			t.Error("cross-ref From and To must not be empty")
		}
		if ref.Kind == "" {
			t.Error("cross-ref Kind must not be empty")
		}
	}
}

func TestRenderText_ContainsKeyStrings(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})

	var buf bytes.Buffer
	if err := report.RenderText(r, &buf); err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	text := buf.String()

	checks := []string{
		"MACHINE BLUEPRINT",
		"example",
		"PURPOSE",
		"COMPONENT INVENTORY",
		"DIAGRAM SPECIFICATIONS",
		"Config",
		"Processor",
		"NewConfig",
	}
	for _, s := range checks {
		if !strings.Contains(text, s) {
			t.Errorf("expected %q in text output", s)
		}
	}
}

func TestRenderText_CrossRefSection(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})

	var buf bytes.Buffer
	if err := report.RenderText(r, &buf); err != nil {
		t.Fatalf("RenderText: %v", err)
	}
	text := buf.String()
	if !strings.Contains(text, "CROSS-REFERENCE MAP") {
		t.Error("expected CROSS-REFERENCE MAP section in text output")
	}
}

func TestRenderJSON_ValidJSON(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})

	var buf bytes.Buffer
	if err := report.RenderJSON(r, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var out interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("JSON output is invalid: %v\n%s", err, buf.String())
	}
}

func TestRenderJSON_ContainsExpectedKeys(t *testing.T) {
	pkg := buildSimplePkg()
	r := report.Generate([]*analyzer.PackageInfo{pkg})

	var buf bytes.Buffer
	if err := report.RenderJSON(r, &buf); err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	text := buf.String()
	for _, key := range []string{`"packages"`, `"name"`, `"components"`, `"diagrams"`} {
		if !strings.Contains(text, key) {
			t.Errorf("JSON output should contain %q", key)
		}
	}
}

func TestGenerate_EmptyPackage(t *testing.T) {
	pkg := &analyzer.PackageInfo{
		Name:       "empty",
		ImportPath: "github.com/example/empty",
		Dir:        "/tmp/empty",
	}
	r := report.Generate([]*analyzer.PackageInfo{pkg})
	if r == nil {
		t.Fatal("expected non-nil report for empty package")
	}
	pr := r.Packages[0]
	if len(pr.Components) != 0 {
		t.Error("expected zero components for empty package")
	}

	var buf bytes.Buffer
	if err := report.RenderText(r, &buf); err != nil {
		t.Fatalf("RenderText on empty package: %v", err)
	}
}

func TestGenerate_MultiplePackages(t *testing.T) {
	pkg1 := buildSimplePkg()
	pkg2 := &analyzer.PackageInfo{
		Name:       "other",
		ImportPath: "github.com/example/other",
		Dir:        "/tmp/other",
		Imports:    []string{"github.com/example/example"},
		Constructs: []analyzer.ConstructInfo{
			{Name: "Helper", Kind: analyzer.KindFunction, Exported: true},
		},
	}
	r := report.Generate([]*analyzer.PackageInfo{pkg1, pkg2})
	if len(r.Packages) != 2 {
		t.Fatalf("expected 2 package reports, got %d", len(r.Packages))
	}
}

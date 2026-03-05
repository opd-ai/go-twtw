package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/opd-ai/go-twtw/analyzer"
)

// testdataDir returns the absolute path to the named subdirectory under testdata/.
func testdataDir(sub string) string {
	abs, _ := filepath.Abs(filepath.Join("testdata", sub))
	return abs
}

func TestAnalyze_Simple(t *testing.T) {
	pkg, err := analyzer.Analyze(testdataDir("simple"))
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	// Package metadata
	if pkg.Name != "simple" {
		t.Errorf("Name = %q, want %q", pkg.Name, "simple")
	}
	if pkg.Doc == "" {
		t.Error("expected non-empty package doc comment")
	}

	// Collect constructs by name for easy lookup.
	byName := make(map[string]analyzer.ConstructInfo)
	for _, c := range pkg.Constructs {
		byName[c.Name] = c
	}

	// Constant
	maxItems, ok := byName["MaxItems"]
	if !ok {
		t.Fatal("expected construct MaxItems")
	}
	if maxItems.Kind != analyzer.KindConst {
		t.Errorf("MaxItems.Kind = %q, want %q", maxItems.Kind, analyzer.KindConst)
	}
	if !maxItems.Exported {
		t.Error("MaxItems should be exported")
	}

	// Variable
	dflt, ok := byName["defaultTimeout"]
	if !ok {
		t.Fatal("expected construct defaultTimeout")
	}
	if dflt.Kind != analyzer.KindVar {
		t.Errorf("defaultTimeout.Kind = %q, want %q", dflt.Kind, analyzer.KindVar)
	}
	if dflt.Exported {
		t.Error("defaultTimeout should not be exported")
	}

	// Type alias
	status, ok := byName["Status"]
	if !ok {
		t.Fatal("expected construct Status")
	}
	if status.Kind != analyzer.KindType {
		t.Errorf("Status.Kind = %q, want %q", status.Kind, analyzer.KindType)
	}

	// Struct
	cfg, ok := byName["Config"]
	if !ok {
		t.Fatal("expected construct Config")
	}
	if cfg.Kind != analyzer.KindStruct {
		t.Errorf("Config.Kind = %q, want %q", cfg.Kind, analyzer.KindStruct)
	}
	if len(cfg.Fields) == 0 {
		t.Error("Config should have fields")
	}
	// One field is a channel
	foundChanField := false
	for _, f := range cfg.Fields {
		if f.Name == "ResultChan" {
			foundChanField = true
		}
	}
	if !foundChanField {
		t.Error("Config.Fields should include ResultChan")
	}

	// Interface
	proc, ok := byName["Processor"]
	if !ok {
		t.Fatal("expected construct Processor")
	}
	if proc.Kind != analyzer.KindInterface {
		t.Errorf("Processor.Kind = %q, want %q", proc.Kind, analyzer.KindInterface)
	}
	if len(proc.Methods) == 0 {
		t.Error("Processor interface should have methods")
	}

	// Function
	newCfg, ok := byName["NewConfig"]
	if !ok {
		t.Fatal("expected construct NewConfig")
	}
	if newCfg.Kind != analyzer.KindFunction {
		t.Errorf("NewConfig.Kind = %q, want %q", newCfg.Kind, analyzer.KindFunction)
	}
	if newCfg.Doc == "" {
		t.Error("NewConfig should have doc comment")
	}

	// Method with receiver
	processMethod, ok := byName["Process"]
	if !ok {
		t.Fatal("expected construct Process")
	}
	if processMethod.Kind != analyzer.KindMethod {
		t.Errorf("Process.Kind = %q, want %q", processMethod.Kind, analyzer.KindMethod)
	}
	if processMethod.Receiver == "" {
		t.Error("Process method should have a Receiver")
	}

	// Function that spawns goroutines
	run, ok := byName["RunProcessor"]
	if !ok {
		t.Fatal("expected construct RunProcessor")
	}
	if !run.SpawnsGoroutines {
		t.Error("RunProcessor should report SpawnsGoroutines = true")
	}
	if !run.HasChannels {
		t.Error("RunProcessor should report HasChannels = true (chan<- Result param)")
	}
}

func TestAnalyze_EmptyDir(t *testing.T) {
	t.TempDir() // just ensure TempDir works
	dir := t.TempDir()
	pkg, err := analyzer.Analyze(dir)
	if err != nil {
		t.Fatalf("unexpected error for empty dir: %v", err)
	}
	if pkg.Name != "" {
		t.Errorf("expected empty Name for empty dir, got %q", pkg.Name)
	}
}

func TestAnalyzeRecursive(t *testing.T) {
	// AnalyzeRecursive skips directories named "testdata", so we start from
	// the simple sub-directory (which is not named "testdata") and verify
	// that the package inside is found.
	pkgs, err := analyzer.AnalyzeRecursive(testdataDir("simple"))
	if err != nil {
		t.Fatalf("AnalyzeRecursive returned error: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("expected at least one package from testdata/simple")
	}
	found := false
	for _, p := range pkgs {
		if p.Name == "simple" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find 'simple' package in recursive analysis")
	}
}

func TestTypeString_ChanTypes(t *testing.T) {
	// Verify HasChannels is set for channel-typed vars/params.
	pkg, err := analyzer.Analyze(testdataDir("simple"))
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	byName := make(map[string]analyzer.ConstructInfo)
	for _, c := range pkg.Constructs {
		byName[c.Name] = c
	}

	// RunProcessor has a chan<- Result param.
	run := byName["RunProcessor"]
	if !run.HasChannels {
		t.Error("RunProcessor.HasChannels should be true")
	}
}

func TestAnalyze_Imports(t *testing.T) {
	pkg, err := analyzer.Analyze(testdataDir("simple"))
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	found := false
	for _, imp := range pkg.Imports {
		if imp == "errors" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'errors' in Imports")
	}
}

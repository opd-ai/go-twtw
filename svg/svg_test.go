package svg_test

import (
	"strings"
	"testing"

	"github.com/opd-ai/go-twtw/analyzer"
	"github.com/opd-ai/go-twtw/svg"
)

func buildTestPkg() *analyzer.PackageInfo {
	return &analyzer.PackageInfo{
		Name:       "widget",
		ImportPath: "github.com/example/widget",
		Dir:        "/tmp/widget",
		Doc:        "Package widget provides reusable UI widgets.",
		Imports:    []string{"fmt", "errors"},
		Constructs: []analyzer.ConstructInfo{
			{Name: "NewWidget", Kind: analyzer.KindFunction, Doc: "NewWidget creates a widget.", Exported: true,
				Results: []analyzer.ParamInfo{{Type: "*Widget"}}},
			{Name: "Widget", Kind: analyzer.KindStruct, Doc: "Widget is a UI component.", Exported: true,
				Fields: []analyzer.FieldInfo{{Name: "ID", Type: "int"}, {Name: "Label", Type: "string"}}},
			{Name: "Renderer", Kind: analyzer.KindInterface, Doc: "Renderer renders widgets.", Exported: true,
				Methods: []string{"Render", "Flush"}},
			{Name: "Render", Kind: analyzer.KindMethod, Doc: "Render draws the widget.", Exported: true,
				Receiver: "*Widget"},
			{Name: "EventChan", Kind: analyzer.KindVar, Doc: "EventChan receives widget events.",
				Exported: true, HasChannels: true},
			{Name: "MaxWidgets", Kind: analyzer.KindConst, Doc: "MaxWidgets is the max count.", Exported: true},
			{Name: "RunLoop", Kind: analyzer.KindFunction, Doc: "RunLoop runs the event loop.",
				Exported: true, SpawnsGoroutines: true},
			{Name: "defaultColor", Kind: analyzer.KindVar, Doc: "defaultColor is the default.", Exported: false},
		},
	}
}

func TestGenerateSVG_NotEmpty(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	if out == "" {
		t.Fatal("GenerateSVG returned empty string")
	}
}

func TestGenerateSVG_ValidSVGHeader(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	if !strings.HasPrefix(out, `<?xml`) {
		t.Error("SVG should start with XML declaration")
	}
	if !strings.Contains(out, `<svg `) {
		t.Error("SVG should contain <svg> root element")
	}
	if !strings.HasSuffix(strings.TrimSpace(out), `</svg>`) {
		t.Error("SVG should end with </svg>")
	}
}

func TestGenerateSVG_ContainsPackageName(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	if !strings.Contains(out, "widget") {
		t.Error("SVG should contain the package name")
	}
}

func TestGenerateSVG_ContainsComponentNames(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	for _, name := range []string{"Widget", "Renderer", "NewWidget"} {
		if !strings.Contains(out, name) {
			t.Errorf("SVG should contain component name %q", name)
		}
	}
}

func TestGenerateSVG_NilPackage(t *testing.T) {
	out := svg.GenerateSVG(nil)
	if out == "" {
		t.Error("GenerateSVG(nil) should return a non-empty fallback SVG")
	}
	if !strings.Contains(out, "<svg") {
		t.Error("GenerateSVG(nil) should return valid SVG")
	}
}

func TestGenerateSVG_EmptyPackage(t *testing.T) {
	pkg := &analyzer.PackageInfo{
		Name:       "empty",
		ImportPath: "github.com/example/empty",
	}
	out := svg.GenerateSVG(pkg)
	if !strings.Contains(out, "<svg") {
		t.Error("SVG for empty package should still contain <svg>")
	}
}

func TestGenerateSVG_ContainsGearShape(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	// Gear is rendered as a polygon.
	if !strings.Contains(out, "<polygon") {
		t.Error("SVG should contain polygon elements (gears/adapters)")
	}
}

func TestGenerateSVG_ContainsLegend(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	if !strings.Contains(out, "Legend") {
		t.Error("SVG should contain the machine-part legend")
	}
}

func TestGenerateSVG_Deterministic(t *testing.T) {
	pkg := buildTestPkg()
	out1 := svg.GenerateSVG(pkg)
	out2 := svg.GenerateSVG(pkg)
	if out1 != out2 {
		t.Error("GenerateSVG should be deterministic (same input → same output)")
	}
}

func TestGenerateSVG_NoXMLInjection(t *testing.T) {
	pkg := &analyzer.PackageInfo{
		Name:       "xss",
		ImportPath: "github.com/example/xss",
		Doc:        `Doc with <script>alert("xss")</script> inside`,
		Constructs: []analyzer.ConstructInfo{
			{Name: `Evil<>&"Name`, Kind: analyzer.KindFunction, Exported: true,
				Doc: `Doc with <b>bold</b> & "quotes"`},
		},
	}
	out := svg.GenerateSVG(pkg)
	// Raw angle brackets from user data should be escaped.
	if strings.Contains(out, "<script>") {
		t.Error("SVG should escape < in text content")
	}
	if strings.Contains(out, `alert("xss")`) {
		t.Error("SVG should escape quotes in text content")
	}
}

func TestGenerateSVG_HasDefs(t *testing.T) {
	pkg := buildTestPkg()
	out := svg.GenerateSVG(pkg)
	if !strings.Contains(out, "<defs>") {
		t.Error("SVG should contain <defs> block with gradients and filters")
	}
	if !strings.Contains(out, "gearGrad") {
		t.Error("SVG defs should define gearGrad gradient")
	}
}

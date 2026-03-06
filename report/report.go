// Package report produces structured, human-readable blueprints that describe
// how a Go package would look as an illustrated machine in the style of
// The Way Things Work. It takes the output of the analyzer and metaphor
// packages and formats them into a text or JSON representation containing:
//
//   - A package overview (name, import path, purpose from doc comments).
//   - A component inventory mapping each construct to its machine-part metaphor.
//   - Diagram specifications listing visual components, data-flow connections,
//     and callout labels derived from source comments.
//   - A cross-reference map showing how packages relate to one another.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/opd-ai/go-twtw/analyzer"
	"github.com/opd-ai/go-twtw/metaphor"
)

// ComponentEntry describes one Go construct in the component inventory table.
type ComponentEntry struct {
	// Name is the construct identifier.
	Name string `json:"name"`
	// Kind is the Go construct kind (function, struct, interface, …).
	Kind string `json:"kind"`
	// PartKind is the machine-part category (gear, pipe, container, …).
	PartKind string `json:"part_kind"`
	// PartName is the human-readable machine-part label (e.g. "Bronze Gear").
	PartName string `json:"part_name"`
	// Visual is the Unicode icon for the machine part.
	Visual string `json:"visual"`
	// Doc is the trimmed documentation comment for this construct.
	Doc string `json:"doc,omitempty"`
	// Exported reports whether the construct is exported.
	Exported bool `json:"exported"`
}

// DiagramComponent is one component shown in a diagram specification.
type DiagramComponent struct {
	// Name is the construct identifier.
	Name string `json:"name"`
	// PartName is the machine-part label.
	PartName string `json:"part_name"`
	// PartKind is the machine-part category.
	PartKind string `json:"part_kind"`
	// Icon is the Unicode icon.
	Icon string `json:"icon"`
	// Callout is the source-comment text used as the diagram annotation.
	Callout string `json:"callout,omitempty"`
}

// Connection describes a data or control-flow link between two components.
type Connection struct {
	// From is the source component name.
	From string `json:"from"`
	// To is the destination component name.
	To string `json:"to"`
	// Kind is the type of connection ("data", "control", "channel", "calls").
	Kind string `json:"kind"`
	// Description is a plain-language description of the link.
	Description string `json:"description"`
}

// DiagramSpec fully specifies one machine diagram for a package.
type DiagramSpec struct {
	// Title is the diagram heading.
	Title string `json:"title"`
	// Components lists all parts shown in this diagram.
	Components []DiagramComponent `json:"components"`
	// Connections lists the data/control-flow links.
	Connections []Connection `json:"connections"`
	// SVGFile is the suggested output filename for the SVG rendering.
	SVGFile string `json:"svg_file"`
	// AsciiLayout is a rough ASCII-art sketch of the diagram arrangement.
	AsciiLayout string `json:"ascii_layout,omitempty"`
}

// PackageReport is the complete blueprint report for one package.
type PackageReport struct {
	// Name is the package name.
	Name string `json:"name"`
	// ImportPath is the package import path (or filesystem dir).
	ImportPath string `json:"import_path"`
	// Dir is the source directory.
	Dir string `json:"dir"`
	// Purpose is the package-level doc comment.
	Purpose string `json:"purpose,omitempty"`
	// Components is the component inventory.
	Components []ComponentEntry `json:"components"`
	// Diagrams is the list of diagram specifications.
	Diagrams []DiagramSpec `json:"diagrams"`
}

// CrossRef describes how two packages are related.
type CrossRef struct {
	// From is the name of the importing package.
	From string `json:"from"`
	// To is the name of the imported package.
	To string `json:"to"`
	// Kind describes the relationship ("imports", "implements", "shares_type").
	Kind string `json:"kind"`
	// Description explains the link in plain language.
	Description string `json:"description"`
}

// Report is the top-level output of the report generator.
type Report struct {
	// Packages contains one PackageReport per analysed package.
	Packages []*PackageReport `json:"packages"`
	// CrossRefs lists dependencies and shared constructs across packages.
	CrossRefs []CrossRef `json:"cross_refs,omitempty"`
}

// Generate builds a Report from one or more PackageInfos. All analysis is
// deterministic: descriptions come from source comments and construct names,
// never from an LLM or stochastic source.
func Generate(pkgs []*analyzer.PackageInfo) *Report {
	r := &Report{}
	for _, pkg := range pkgs {
		r.Packages = append(r.Packages, buildPackageReport(pkg))
	}
	r.CrossRefs = buildCrossRefs(pkgs)
	return r
}

// buildPackageReport converts one PackageInfo into a PackageReport.
func buildPackageReport(pkg *analyzer.PackageInfo) *PackageReport {
	pr := &PackageReport{
		Name:       pkg.Name,
		ImportPath: pkg.ImportPath,
		Dir:        pkg.Dir,
		Purpose:    pkg.Doc,
	}

	for _, c := range pkg.Constructs {
		part := metaphor.Map(c)
		pr.Components = append(pr.Components, ComponentEntry{
			Name:     c.Name,
			Kind:     string(c.Kind),
			PartKind: string(part.Kind),
			PartName: part.Name,
			Visual:   part.Icon,
			Doc:      c.Doc,
			Exported: c.Exported,
		})
	}

	pr.Diagrams = buildDiagrams(pkg, pr.Components)
	return pr
}

// buildDiagrams produces diagram specifications for a package.
// For packages with ≤ 12 constructs a single overview diagram is generated.
// Larger packages also get a focused "exported API" diagram.
func buildDiagrams(pkg *analyzer.PackageInfo, entries []ComponentEntry) []DiagramSpec {
	if len(entries) == 0 {
		return nil
	}

	overview := buildOverviewDiagram(pkg, entries)
	diagrams := []DiagramSpec{overview}

	// If there are many constructs, add a focused exported-only diagram.
	exportedCount := 0
	for _, e := range entries {
		if e.Exported {
			exportedCount++
		}
	}
	if exportedCount > 0 && len(entries) > 6 {
		diagrams = append(diagrams, buildExportedDiagram(pkg, entries))
	}

	return diagrams
}

// buildOverviewDiagram creates the main machine overview for a package.
func buildOverviewDiagram(pkg *analyzer.PackageInfo, entries []ComponentEntry) DiagramSpec {
	spec := DiagramSpec{
		Title:   fmt.Sprintf("Package %q – Full Machine Overview", pkg.Name),
		SVGFile: pkg.Name + "-overview.svg",
	}

	// Arrange components in visual sections mirroring the SVG layout:
	// adapters (interfaces) at the top, gears/belts in the middle, containers at the bottom.
	sections := map[string][]ComponentEntry{
		"top":    {},
		"middle": {},
		"bottom": {},
	}
	for _, e := range entries {
		switch metaphor.MachinePartKind(e.PartKind) {
		case metaphor.PartAdapter:
			sections["top"] = append(sections["top"], e)
		case metaphor.PartGear, metaphor.PartLever, metaphor.PartBelt, metaphor.PartPipe:
			sections["middle"] = append(sections["middle"], e)
		default:
			sections["bottom"] = append(sections["bottom"], e)
		}
	}

	for _, e := range entries {
		dc := DiagramComponent{
			Name:     e.Name,
			PartName: e.PartName,
			PartKind: e.PartKind,
			Icon:     e.Visual,
			Callout:  e.Doc,
		}
		spec.Components = append(spec.Components, dc)
	}

	spec.Connections = inferConnections(pkg)
	spec.AsciiLayout = buildAsciiLayout(sections)
	return spec
}

// buildExportedDiagram creates a focused diagram showing only exported constructs.
func buildExportedDiagram(pkg *analyzer.PackageInfo, entries []ComponentEntry) DiagramSpec {
	spec := DiagramSpec{
		Title:   fmt.Sprintf("Package %q – Exported API Blueprint", pkg.Name),
		SVGFile: pkg.Name + "-api.svg",
	}

	sections := map[string][]ComponentEntry{
		"top":    {},
		"middle": {},
		"bottom": {},
	}

	for _, e := range entries {
		if !e.Exported {
			continue
		}
		dc := DiagramComponent{
			Name:     e.Name,
			PartName: e.PartName,
			PartKind: e.PartKind,
			Icon:     e.Visual,
			Callout:  e.Doc,
		}
		spec.Components = append(spec.Components, dc)

		switch metaphor.MachinePartKind(e.PartKind) {
		case metaphor.PartAdapter:
			sections["top"] = append(sections["top"], e)
		case metaphor.PartGear, metaphor.PartLever, metaphor.PartBelt, metaphor.PartPipe:
			sections["middle"] = append(sections["middle"], e)
		default:
			sections["bottom"] = append(sections["bottom"], e)
		}
	}

	spec.Connections = inferConnections(pkg)
	spec.AsciiLayout = buildAsciiLayout(sections)
	return spec
}

// inferConnections deduces data/control-flow connections from the package AST.
// The heuristic is:
//   - A function that returns a type T or *T "feeds into" the struct T (container).
//   - A function that accepts a channel param "connects to" that channel (pipe).
//   - A function that spawns goroutines "drives" a conveyor belt.
//   - Methods are connected to their receiver type.
func inferConnections(pkg *analyzer.PackageInfo) []Connection {
	// Build a name→kind index for fast lookup.
	nameKind := make(map[string]analyzer.ConstructKind)
	for _, c := range pkg.Constructs {
		nameKind[c.Name] = c.Kind
	}

	seen := make(map[string]bool)
	var conns []Connection

	add := func(from, to, kind, desc string) {
		key := from + "|" + to + "|" + kind
		if seen[key] {
			return
		}
		seen[key] = true
		conns = append(conns, Connection{From: from, To: to, Kind: kind, Description: desc})
	}

	for _, c := range pkg.Constructs {
		switch c.Kind {
		case analyzer.KindMethod:
			// Method → receiver type.
			recv := strings.TrimPrefix(c.Receiver, "*")
			if recv != "" {
				add(c.Name, recv, "control",
					fmt.Sprintf("%s operates on its receiver %s", c.Name, c.Receiver))
			}

		case analyzer.KindFunction:
			// Constructor functions (New…) → the type they return.
			for _, r := range c.Results {
				typeName := strings.TrimPrefix(r.Type, "*")
				if _, exists := nameKind[typeName]; exists {
					add(c.Name, typeName, "data",
						fmt.Sprintf("%s produces a %s value", c.Name, r.Type))
				}
			}
			// Functions that accept channel params.
			for _, p := range c.Params {
				if strings.Contains(p.Type, "chan") {
					// Find the channel var/field it refers to.
					for _, other := range pkg.Constructs {
						if other.HasChannels && other.Name != c.Name {
							add(other.Name, c.Name, "channel",
								fmt.Sprintf("pipe %s carries values into %s", other.Name, c.Name))
							break
						}
					}
				}
			}
			// Goroutine-spawning → belt connection.
			if c.SpawnsGoroutines {
				add(c.Name, "goroutine", "belt",
					fmt.Sprintf("%s starts concurrent work on a conveyor belt", c.Name))
			}
		}
	}

	sort.Slice(conns, func(i, j int) bool {
		return conns[i].From < conns[j].From
	})
	return conns
}

// buildAsciiLayout produces a rough ASCII-art sketch of the diagram's three
// horizontal sections (adapters, gears, containers).
func buildAsciiLayout(sections map[string][]ComponentEntry) string {
	var sb strings.Builder
	border := "┌" + strings.Repeat("─", 66) + "┐"
	mid := "├" + strings.Repeat("─", 66) + "┤"
	bottom := "└" + strings.Repeat("─", 66) + "┘"

	sb.WriteString(border + "\n")

	// Top section – adapters (interfaces).
	if len(sections["top"]) > 0 {
		sb.WriteString(sectionLine("ADAPTERS (interfaces)", sections["top"]))
		sb.WriteString(mid + "\n")
	}

	// Middle section – gears, levers, belts, pipes.
	if len(sections["middle"]) > 0 {
		sb.WriteString(sectionLine("GEARS / BELTS / PIPES (functions, goroutines, channels)", sections["middle"]))
		sb.WriteString(mid + "\n")
	}

	// Bottom section – containers, gauges, weights.
	sb.WriteString(sectionLine("CONTAINERS / GAUGES / WEIGHTS (structs, vars, consts)", sections["bottom"]))
	sb.WriteString(bottom + "\n")

	return sb.String()
}

func sectionLine(label string, entries []ComponentEntry) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("│  %-64s│\n", label))
	if len(entries) == 0 {
		sb.WriteString("│  (empty)                                                       │\n")
		return sb.String()
	}
	// Render at most 5 components per row.
	const perRow = 5
	// targetRuneWidth is the number of printable rune columns between the
	// leading "│  " (3 runes) and the closing "│" (1 rune): 68 - 3 - 1 = 64.
	const targetRuneWidth = 64
	for i := 0; i < len(entries); i += perRow {
		end := i + perRow
		if end > len(entries) {
			end = len(entries)
		}
		row := entries[i:end]

		// Build the row content using rune-aware padding.
		// Each cell is 14 rune-columns wide (icon + space + name truncated to 10,
		// padded or trimmed to exactly 14), separated by 2 spaces.
		var cells []string
		for _, e := range row {
			cell := e.Visual + " " + truncate(e.Name, 10)
			// Pad or trim to exactly 14 rune columns.
			cellRunes := []rune(cell)
			switch {
			case len(cellRunes) < 14:
				cell += strings.Repeat(" ", 14-len(cellRunes))
			case len(cellRunes) > 14:
				cell = string(cellRunes[:14])
			}
			cells = append(cells, cell)
		}
		content := strings.Join(cells, "  ")
		contentRunes := []rune(content)

		// Pad content to exactly targetRuneWidth rune columns.
		switch {
		case len(contentRunes) < targetRuneWidth:
			content += strings.Repeat(" ", targetRuneWidth-len(contentRunes))
		case len(contentRunes) > targetRuneWidth:
			content = string(contentRunes[:targetRuneWidth])
		}
		sb.WriteString("│  " + content + "│\n")
	}
	return sb.String()
}

// buildCrossRefs builds the cross-reference map showing which packages import others.
func buildCrossRefs(pkgs []*analyzer.PackageInfo) []CrossRef {
	seen := make(map[string]bool)
	var refs []CrossRef

	for _, p := range pkgs {
		for _, imp := range p.Imports {
			// Cross-reference all imports, since each represents a machine connection
			// between this package and another.
			key := p.Name + "→" + imp
			if seen[key] {
				continue
			}
			seen[key] = true

			kind := "imports"
			desc := fmt.Sprintf("Package %q imports %q – connecting it to an external machine component.", p.Name, imp)
			refs = append(refs, CrossRef{
				From:        p.Name,
				To:          imp,
				Kind:        kind,
				Description: desc,
			})
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].From != refs[j].From {
			return refs[i].From < refs[j].From
		}
		return refs[i].To < refs[j].To
	})
	return refs
}

// ─── Text renderer ───────────────────────────────────────────────────────────

// RenderText writes a human-readable text report to w.
func RenderText(r *Report, w io.Writer) error {
	if r == nil {
		return fmt.Errorf("report: nil report")
	}
	for i, pkg := range r.Packages {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := renderPackageText(pkg, w); err != nil {
			return err
		}
	}
	if len(r.CrossRefs) > 0 {
		if err := renderCrossRefsText(r.CrossRefs, w); err != nil {
			return err
		}
	}
	return nil
}

func renderPackageText(pkg *PackageReport, w io.Writer) error {
	sep := strings.Repeat("=", 72)
	thin := strings.Repeat("─", 72)

	p := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		return err
	}

	if err := p("%s\n", sep); err != nil {
		return err
	}
	if err := p("MACHINE BLUEPRINT: Package %q\n", pkg.Name); err != nil {
		return err
	}
	if err := p("%s\n\n", sep); err != nil {
		return err
	}

	// Package overview
	if err := p("PURPOSE\n%s\n", thin); err != nil {
		return err
	}
	purpose := pkg.Purpose
	if purpose == "" {
		purpose = "(no package-level doc comment found)"
	}
	if err := p("%s\n\n", purpose); err != nil {
		return err
	}
	if err := p("Import Path : %s\n", pkg.ImportPath); err != nil {
		return err
	}
	if err := p("Directory   : %s\n\n", pkg.Dir); err != nil {
		return err
	}

	// Component inventory
	if err := p("COMPONENT INVENTORY\n%s\n", thin); err != nil {
		return err
	}
	if len(pkg.Components) == 0 {
		if err := p("  (no constructs found)\n\n"); err != nil {
			return err
		}
	} else {
		header := fmt.Sprintf("  %-20s %-12s %-18s %-6s  %s\n",
			"Name", "Kind", "Machine Part", "Icon", "Exported")
		if _, err := fmt.Fprint(w, header); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, "  "+strings.Repeat("-", 68)+"\n"); err != nil {
			return err
		}
		for _, e := range pkg.Components {
			exp := "no"
			if e.Exported {
				exp = "yes"
			}
			line := fmt.Sprintf("  %-20s %-12s %-18s %-6s  %s\n",
				truncate(e.Name, 19), e.Kind, e.PartName, e.Visual, exp)
			if _, err := fmt.Fprint(w, line); err != nil {
				return err
			}
			if e.Doc != "" {
				doc := "    └─ " + firstSentence(e.Doc)
				if len(doc) > 78 {
					doc = doc[:75] + "…"
				}
				if _, err := fmt.Fprint(w, doc+"\n"); err != nil {
					return err
				}
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Diagram specifications
	if err := p("DIAGRAM SPECIFICATIONS\n%s\n", thin); err != nil {
		return err
	}
	for i, d := range pkg.Diagrams {
		if err := renderDiagramText(d, i+1, w); err != nil {
			return err
		}
	}

	return nil
}

func renderDiagramText(d DiagramSpec, n int, w io.Writer) error {
	p := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		return err
	}

	if err := p("\n[Diagram %d: %s]\n", n, d.Title); err != nil {
		return err
	}
	if err := p("  SVG Output: %s\n\n", d.SVGFile); err != nil {
		return err
	}

	if d.AsciiLayout != "" {
		lines := strings.Split(d.AsciiLayout, "\n")
		for _, l := range lines {
			if l == "" {
				continue
			}
			if err := p("  %s\n", l); err != nil {
				return err
			}
		}
		if err := p("\n"); err != nil {
			return err
		}
	}

	if len(d.Components) > 0 {
		if err := p("  Components:\n"); err != nil {
			return err
		}
		for _, c := range d.Components {
			if err := p("    %s %-20s (%s)\n", c.Icon, c.Name, c.PartName); err != nil {
				return err
			}
			if c.Callout != "" {
				callout := firstSentence(c.Callout)
				if err := p("        Callout: %q\n", callout); err != nil {
					return err
				}
			}
		}
		if err := p("\n"); err != nil {
			return err
		}
	}

	if len(d.Connections) > 0 {
		if err := p("  Connections:\n"); err != nil {
			return err
		}
		for _, c := range d.Connections {
			if err := p("    • %s ──[%s]──▶ %s\n", c.From, c.Kind, c.To); err != nil {
				return err
			}
			if c.Description != "" {
				if err := p("        %s\n", c.Description); err != nil {
					return err
				}
			}
		}
		if err := p("\n"); err != nil {
			return err
		}
	}

	return nil
}

func renderCrossRefsText(refs []CrossRef, w io.Writer) error {
	thin := strings.Repeat("─", 72)
	if _, err := fmt.Fprintf(w, "\nCROSS-REFERENCE MAP\n%s\n", thin); err != nil {
		return err
	}
	for _, ref := range refs {
		if _, err := fmt.Fprintf(w, "  %s  ──[%s]──▶  %s\n", ref.From, ref.Kind, ref.To); err != nil {
			return err
		}
		if ref.Description != "" {
			if _, err := fmt.Fprintf(w, "      %s\n", ref.Description); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── JSON renderer ───────────────────────────────────────────────────────────

// RenderJSON writes the report as indented JSON to w.
func RenderJSON(r *Report, w io.Writer) error {
	if r == nil {
		return fmt.Errorf("report: nil report")
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// truncate shortens s to at most maxLen runes, appending "…" if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

// firstSentence returns the first sentence of s (up to the first period or newline).
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, ".\n"); i >= 0 {
		return s[:i+1]
	}
	return s
}

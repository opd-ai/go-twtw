# Goal-Achievement Assessment

## Project Context

- **What it claims to do**: A deterministic Go package visualization tool that represents Go constructs as illustrated physical machines in the style of David Macaulay's "The Way Things Work" and Sierra's "The Incredible Machine." Maps functions to gears, methods to levers, interfaces to adapter plugs, structs to storage containers, channels to pipes, goroutine-spawning functions to conveyor belts, variables to gauges, and constants to calibrated weights.

- **Target audience**: Go developers wanting to visualize package structure, teachers/learners exploring Go codebases, and documentation authors needing machine-style diagrams.

- **Architecture**:
  | Package    | Role |
  |------------|------|
  | `main`     | CLI entry point with flags `-format`, `-output`, `-no-svg`, `-recursive` |
  | `analyzer` | Parses Go source using `go/ast`, extracts constructs with metadata (channels, goroutines, docs) |
  | `metaphor` | Maps each construct kind to a physical machine-part metaphor with rationale |
  | `report`   | Generates text/JSON reports with component inventory, diagram specs, cross-refs |
  | `svg`      | Renders deterministic SVG diagrams using pure procedural math (no external libs) |

- **Existing CI/quality gates**: None (no `.github/workflows/`, `Makefile`, or CI config)

---

## Goal-Achievement Summary

| Stated Goal | Status | Evidence | Gap Description |
|-------------|--------|----------|-----------------|
| Parse Go packages using go/ast | ✅ Achieved | `analyzer/analyzer.go:119-144` — uses `parser.ParseDir` with comment parsing | — |
| Extract functions, methods, types, interfaces, variables, constants | ✅ Achieved | Tests pass; analyzer extracts all 7 construct kinds | — |
| Extract doc comments and structural metadata | ✅ Achieved | `ConstructInfo.Doc`, `Fields`, `Methods` populated; 100% doc coverage per metrics | — |
| Detect channel types | ✅ Achieved | `HasChannels` flag set correctly; `isChanExpr()` implementation | — |
| Detect goroutine-spawning functions | ✅ Achieved | `SpawnsGoroutines` via `bodyHasGoStmt()` AST inspection | — |
| Map constructs to machine-part metaphors | ✅ Achieved | `metaphor.Map()` deterministically assigns 8 part types with rationale | — |
| Generate text reports | ✅ Achieved | `report.RenderText()` outputs structured blueprints | — |
| Generate JSON reports | ✅ Achieved | `report.RenderJSON()` produces valid JSON | — |
| Generate SVG diagrams | ✅ Achieved | `svg.GenerateSVG()` renders all 8 shape types procedurally | — |
| Deterministic output (no LLM/stochastic) | ✅ Achieved | Test `TestGenerateSVG_Deterministic` confirms same input → same output | — |
| Recursive package analysis | ✅ Achieved | `-recursive` flag; `AnalyzeRecursive()` walks subdirectories | — |
| Style inspired by "The Way Things Work" | ⚠️ Partial | SVG uses gears, levers, containers; lacks Macaulay's signature whimsy (mammoths, cutaways, hand-drawn feel) | Visual style is functional but plain; no cross-section views or narrative annotations |
| Cross-reference map | ⚠️ Partial | Shows imports only; does not detect interface implementations or shared types across packages | Missing "implements" and "shares_type" relationship detection |
| Data-flow connections in diagrams | ⚠️ Partial | Connects constructors→types and methods→receivers; misses function call chains | No static call-graph analysis |

**Overall: 11/14 goals fully achieved**

---

## Roadmap

### Priority 1: Add CI Pipeline and Release Automation

The project has no CI, which poses risk for a tool claiming determinism and correctness.

- [ ] Create `.github/workflows/ci.yml`:
  - Run `go test -race ./...`
  - Run `go vet ./...`
  - Run `staticcheck` or `golangci-lint`
  - Test on Go 1.24 (declared in `go.mod`)
- [ ] Add release workflow to build binaries for linux/darwin/windows on tags
- [ ] **Validation**: All tests pass in CI; releases auto-publish to GitHub Releases

### Priority 2: Expand Cross-Reference Detection

Current cross-refs only show imports. The stated goal includes richer relationships.

- [ ] In `report/report.go`, add detection for:
  - **Interface implementations**: Scan types with method sets matching interface signatures
  - **Shared types**: Identify when two packages both reference the same type name
- [ ] Update `CrossRef.Kind` enum to include `"implements"` and `"shares_type"`
- [ ] Add tests in `report/report_test.go` for new relationship types
- [ ] **Validation**: Running on a multi-package repo shows "implements" links

**Files**: `report/report.go:414-446`, new helper functions

### Priority 3: Enrich SVG Visual Style

The current style is schematic but lacks the illustrative warmth of Macaulay's work.

- [ ] Add optional cross-section/cutaway views for struct internals (show fields as labeled shelves inside containers)
- [ ] Add subtle hatching or hand-drawn effect filter to SVG `<defs>` for optional "sketch" mode
- [ ] Include callout leader lines from doc comments to components
- [ ] Consider adding a mascot element (small helper figure) for large diagrams
- [ ] **Validation**: Visual comparison of output shows increased illustrative detail

**Files**: `svg/svg.go:475-507` (drawContainer), `svg/svg.go:150-209` (svgDefs)

### Priority 4: Add Static Call-Graph Analysis

Data-flow connections currently rely on heuristics (return types, receivers). Full call-graph would strengthen diagrams.

- [ ] Integrate `golang.org/x/tools/go/callgraph` for intra-package call analysis
- [ ] Add `Calls []string` field to `ConstructInfo` for function/method calls
- [ ] In `report/report.go:inferConnections()`, add `"calls"` connection type
- [ ] Draw call edges in SVG with distinct dash pattern
- [ ] **Validation**: Running on analyzer package shows Analyze→buildPackageInfo→extractFromFile chain

**Files**: `analyzer/analyzer.go`, `report/report.go:269-331`, `svg/svg.go:278-336`

### Priority 5: Reduce Complexity in Report Rendering

`go-stats-generator` flagged high cyclomatic complexity in rendering functions:
- `renderPackageText`: 33.2 overall (24 cyclomatic)
- `renderDiagramText`: 30.6 overall (22 cyclomatic)
- `inferConnections`: 19.9 overall (13 cyclomatic)

- [ ] Extract `renderPurposeSection()`, `renderInventorySection()`, `renderDiagramsSection()` from `renderPackageText`
- [ ] Extract `renderComponentsList()`, `renderConnectionsList()` from `renderDiagramText`
- [ ] Break `inferConnections` into `inferMethodConnections`, `inferConstructorConnections`, `inferChannelConnections`
- [ ] **Validation**: Re-run `go-stats-generator`; no function exceeds 20 cyclomatic complexity

**Files**: `report/report.go:473-563`, `report/report.go:565-632`, `report/report.go:269-331`

### Priority 6: Add Interactive HTML Output

SVG is static; an HTML wrapper would enable hover tooltips and click-to-navigate.

- [ ] Create `html/html.go` package
- [ ] Wrap SVG in HTML with:
  - Hover tooltips showing full doc comments
  - Click handlers linking to source file:line
  - Collapsible legend
- [ ] Add `-format html` flag
- [ ] **Validation**: Output opens in browser with working hover/click behavior

### Priority 7: Improve README Documentation

The README is a single line. Users need installation, usage examples, and sample output.

- [ ] Add installation instructions (`go install github.com/opd-ai/go-twtw@latest`)
- [ ] Add example command output (text, JSON, SVG snippets)
- [ ] Add screenshot or embedded SVG of example diagram
- [ ] Document the metaphor mappings (gear = function, etc.)
- [ ] **Validation**: README contains > 100 lines with all sections

---

## Metrics Summary (from go-stats-generator)

| Metric | Value |
|--------|-------|
| Total Lines of Code | 1,447 |
| Total Functions | 71 |
| Documentation Coverage | 100% |
| Functions > 50 lines | 7 (9.6%) |
| High Complexity (>10) | 5 functions |
| Magic Numbers | 647 (many are SVG coordinates/colors — acceptable) |
| Circular Dependencies | None |
| Test Results | All pass, including race detector |

---

## Appendix: Highest-Risk Functions

| Function | File | Lines | Cyclomatic | Risk |
|----------|------|-------|------------|------|
| `renderPackageText` | report/report.go | 89 | 24 | High — large render function |
| `renderDiagramText` | report/report.go | 67 | 22 | High — nested conditional printing |
| `extractTypeSpec` | analyzer/analyzer.go | 56 | 14 | Medium — switch on AST types |
| `inferConnections` | report/report.go | 62 | 13 | Medium — heuristic matching |
| `main` | main.go | 86 | 14 | Medium — CLI orchestration (acceptable) |

All flagged functions are on non-critical paths (rendering, not core logic). The analyzer extraction functions handle complex AST patterns appropriately given their purpose.

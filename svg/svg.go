// Package svg generates deterministic SVG machine-diagram images for Go
// packages. Each Go construct is drawn as a physical machine component in the
// style of David Macaulay's The Way Things Work:
//
//   - Functions       → Bronze toothed gears
//   - Methods         → Steel levers on a fulcrum
//   - Interfaces      → Golden hexagonal adapter plugs
//   - Structs         → Green 3-D storage containers
//   - Channels/pipes  → Blue cylindrical pipes
//   - Goroutine funcs → Dark conveyor belts
//   - Variables       → Brass pressure gauges
//   - Constants       → Slate-grey calibrated weights
//
// All drawing is purely procedural (math + string formatting); no external
// image libraries or LLM calls are used.
package svg

import (
	"fmt"
	"math"
	"strings"

	"github.com/opd-ai/go-twtw/analyzer"
	"github.com/opd-ai/go-twtw/metaphor"
)

// ─── Layout constants ─────────────────────────────────────────────────────────

const (
	canvasWidth   = 960
	compSize      = 110 // bounding-box size for each component
	hGap          = 36  // horizontal gap between components
	vGap          = 70  // vertical gap between rows
	marginX       = 60
	marginY       = 30
	headerH       = 90
	legendH       = 130
	sectionGap    = 60
	labelFontSize = 10
)

// ─── Component layout ─────────────────────────────────────────────────────────

type pos struct{ x, y float64 }

type compLayout struct {
	c    analyzer.ConstructInfo
	part metaphor.MachinePart
	p    pos
}

// GenerateSVG creates a full SVG document for a package. The returned string
// is a self-contained SVG file ready to write to disk or embed in HTML.
func GenerateSVG(pkg *analyzer.PackageInfo) string {
	if pkg == nil {
		return emptySVG()
	}

	// Assign metaphors and group by visual section.
	var adapters, middles, bottoms []compLayout
	for _, c := range pkg.Constructs {
		part := metaphor.Map(c)
		cl := compLayout{c: c, part: part}
		switch part.Kind {
		case metaphor.PartAdapter:
			adapters = append(adapters, cl)
		case metaphor.PartGear, metaphor.PartLever, metaphor.PartBelt, metaphor.PartPipe:
			middles = append(middles, cl)
		default:
			bottoms = append(bottoms, cl)
		}
	}

	// Assign positions row by row.
	curY := float64(marginY + headerH + sectionGap)
	adapters, curY = layoutRow(adapters, curY)
	if len(adapters) > 0 {
		curY += float64(compSize + sectionGap)
	}
	middles, curY = layoutRow(middles, curY)
	if len(middles) > 0 {
		curY += float64(compSize + sectionGap)
	}
	bottoms, curY = layoutRow(bottoms, curY)
	if len(bottoms) > 0 {
		curY += float64(compSize)
	}

	totalH := int(curY) + legendH + marginY
	all := append(append(adapters, middles...), bottoms...)

	var b strings.Builder
	writef(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	writef(&b, `<svg xmlns="http://www.w3.org/2000/svg" `+
		`width="%d" height="%d" viewBox="0 0 %d %d">`+"\n",
		canvasWidth, totalH, canvasWidth, totalH)

	b.WriteString(svgDefs())
	b.WriteString(background(canvasWidth, totalH))
	b.WriteString(drawHeader(pkg, canvasWidth, marginY))

	// Draw connections behind components.
	b.WriteString(drawConnections(all))

	// Draw each component.
	for _, cl := range all {
		b.WriteString(drawComponent(cl))
	}

	b.WriteString(drawLegend(marginX, float64(totalH-legendH-marginY/2)))
	b.WriteString("</svg>\n")
	return b.String()
}

// layoutRow assigns x,y positions to a slice of compLayouts in a wrapping grid.
// It returns the updated slice and the Y coordinate of the top of the last row.
func layoutRow(cls []compLayout, startY float64) ([]compLayout, float64) {
	if len(cls) == 0 {
		return cls, startY
	}
	maxPerRow := (canvasWidth - 2*marginX + hGap) / (compSize + hGap)
	if maxPerRow < 1 {
		maxPerRow = 1
	}
	for i := range cls {
		row := i / maxPerRow
		col := i % maxPerRow
		cls[i].p = pos{
			x: float64(marginX + col*(compSize+hGap)),
			y: startY + float64(row)*(float64(compSize)+float64(vGap)),
		}
	}
	rows := (len(cls) + maxPerRow - 1) / maxPerRow
	lastRowY := startY + float64(rows-1)*(float64(compSize)+float64(vGap))
	return cls, lastRowY
}

// ─── SVG building blocks ─────────────────────────────────────────────────────

func writef(b *strings.Builder, format string, args ...interface{}) {
	fmt.Fprintf(b, format, args...)
}

func emptySVG() string {
	return `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"></svg>`
}

// svgDefs returns the <defs> block containing gradients, filters, and patterns
// used throughout the diagram.
func svgDefs() string {
	return `  <defs>
    <!-- Parchment background gradient -->
    <linearGradient id="parchment" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%"   stop-color="#f8f4ea"/>
      <stop offset="100%" stop-color="#ede5cf"/>
    </linearGradient>
    <!-- Component drop-shadow -->
    <filter id="shadow" x="-10%" y="-10%" width="130%" height="130%">
      <feDropShadow dx="3" dy="3" stdDeviation="3" flood-color="#00000033"/>
    </filter>
    <!-- Gear gradient -->
    <radialGradient id="gearGrad" cx="50%" cy="40%" r="60%">
      <stop offset="0%"   stop-color="#e8a855"/>
      <stop offset="100%" stop-color="#9a5a1a"/>
    </radialGradient>
    <!-- Pipe gradient -->
    <linearGradient id="pipeGrad" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%"   stop-color="#7ab0d8"/>
      <stop offset="50%"  stop-color="#2d6fa0"/>
      <stop offset="100%" stop-color="#1a4870"/>
    </linearGradient>
    <!-- Container gradient -->
    <linearGradient id="containerGrad" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%"   stop-color="#a8c8a8"/>
      <stop offset="100%" stop-color="#557a55"/>
    </linearGradient>
    <!-- Adapter gradient -->
    <radialGradient id="adapterGrad" cx="50%" cy="40%" r="60%">
      <stop offset="0%"   stop-color="#f0d060"/>
      <stop offset="100%" stop-color="#a07010"/>
    </radialGradient>
    <!-- Belt gradient -->
    <linearGradient id="beltGrad" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%"   stop-color="#888"/>
      <stop offset="100%" stop-color="#333"/>
    </linearGradient>
    <!-- Gauge gradient -->
    <radialGradient id="gaugeGrad" cx="50%" cy="40%" r="60%">
      <stop offset="0%"   stop-color="#f0d8a0"/>
      <stop offset="100%" stop-color="#b07820"/>
    </radialGradient>
    <!-- Weight gradient -->
    <linearGradient id="weightGrad" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%"   stop-color="#9ab0c0"/>
      <stop offset="100%" stop-color="#445566"/>
    </linearGradient>
    <!-- Lever gradient -->
    <linearGradient id="leverGrad" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0%"   stop-color="#d0d0d0"/>
      <stop offset="100%" stop-color="#707070"/>
    </linearGradient>
    <!-- Arrow marker for connection lines -->
    <marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5"
            markerWidth="6" markerHeight="6" orient="auto-start-reverse">
      <path d="M 0 0 L 10 5 L 0 10 z" fill="#8B7355" opacity="0.7"/>
    </marker>
  </defs>
`
}

// background renders the parchment fill and decorative double-border.
func background(w, h int) string {
	var b strings.Builder
	writef(&b, `  <rect width="%d" height="%d" fill="url(#parchment)"/>`+"\n", w, h)
	writef(&b, `  <rect x="8" y="8" width="%d" height="%d" `+
		`fill="none" stroke="#8B7355" stroke-width="3" rx="6"/>`+"\n", w-16, h-16)
	writef(&b, `  <rect x="13" y="13" width="%d" height="%d" `+
		`fill="none" stroke="#8B7355" stroke-width="1" rx="4"/>`+"\n", w-26, h-26)
	return b.String()
}

// drawHeader renders the machine-blueprint title banner.
func drawHeader(pkg *analyzer.PackageInfo, w, y int) string {
	var b strings.Builder
	cx := w / 2
	// Banner rectangle
	writef(&b, `  <rect x="30" y="%d" width="%d" height="60" rx="4" `+
		`fill="#3a2a1a" stroke="#8B7355" stroke-width="2"/>`+"\n",
		y+5, w-60)
	// Gear decorations in the banner
	for _, gx := range []int{50, w - 50} {
		writef(&b, `  %s`+"\n", miniGear(float64(gx), float64(y+35), 18))
	}
	// Package name
	writef(&b, `  <text x="%d" y="%d" `+
		`text-anchor="middle" font-family="Georgia, serif" `+
		`font-size="22" font-weight="bold" fill="#f5e8c0">%s</text>`+"\n",
		cx, y+30, xmlEscape("Package: "+pkg.Name))
	// Doc comment (first sentence) – truncate rune-aware to avoid splitting UTF-8.
	// maxDocRunes is the total allowed runes; truncated strings end in "…" (1 rune),
	// so the body is capped at maxDocRunes-3 runes to leave room for the ellipsis.
	const maxDocRunes = 90
	doc := pkg.Doc
	docRunes := []rune(doc)
	if len(docRunes) > maxDocRunes {
		doc = string(docRunes[:maxDocRunes-3]) + "…"
	}
	if doc == "" {
		doc = "No package-level documentation."
	}
	writef(&b, `  <text x="%d" y="%d" `+
		`text-anchor="middle" font-family="Georgia, serif" `+
		`font-size="11" fill="#c8b890">%s</text>`+"\n",
		cx, y+50, xmlEscape(doc))
	return b.String()
}

// layoutKey returns a stable key for indexing component layouts.
// Methods are keyed by "<receiver>.<name>" to avoid collisions when the same
// method name appears on different receiver types. Functions are prefixed with
// "func:" so they do not collide with type names that share the same identifier.
// All other constructs use their bare Name, which keeps type-name lookups working.
func layoutKey(c analyzer.ConstructInfo) string {
	switch c.Kind {
	case analyzer.KindMethod:
		recv := strings.TrimPrefix(c.Receiver, "*")
		if recv != "" {
			return recv + "." + c.Name
		}
		return "method:" + c.Name
	case analyzer.KindFunction:
		return "func:" + c.Name
	default:
		return c.Name
	}
}

// drawConnections draws arrows between related components.
func drawConnections(all []compLayout) string {
	if len(all) == 0 {
		return ""
	}
	// Build a stable key→layout index.
	idx := make(map[string]compLayout)
	for _, cl := range all {
		idx[layoutKey(cl.c)] = cl
	}

	var b strings.Builder
	seen := make(map[string]bool)

	for _, cl := range all {
		switch cl.c.Kind {
		case analyzer.KindMethod:
			// Connect method to its receiver type (bare name, no prefix).
			recv := strings.TrimPrefix(cl.c.Receiver, "*")
			if target, ok := idx[recv]; ok {
				drawArrow(&b, cl.p, target.p, "#8B7355", "control", seen)
			}
		case analyzer.KindFunction:
			// Constructor → result type (bare name, no prefix).
			for _, r := range cl.c.Results {
				typeName := strings.TrimPrefix(r.Type, "*")
				if target, ok := idx[typeName]; ok {
					drawArrow(&b, cl.p, target.p, "#4682B4", "data", seen)
				}
			}
		}
	}
	return b.String()
}

// drawArrow renders a dashed arrow from one component centre to another.
func drawArrow(b *strings.Builder, from, to pos, color, kind string, seen map[string]bool) {
	key := fmt.Sprintf("%.0f,%.0f→%.0f,%.0f", from.x, from.y, to.x, to.y)
	if seen[key] {
		return
	}
	seen[key] = true

	cx := float64(compSize) / 2
	x1 := from.x + cx
	y1 := from.y + cx
	x2 := to.x + cx
	y2 := to.y + cx

	dash := "6,4"
	if kind == "data" {
		dash = "8,3"
	}

	writef(b, `  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" `+
		`stroke="%s" stroke-width="1.5" stroke-dasharray="%s" `+
		`marker-end="url(#arrow)" opacity="0.6"/>`+"\n",
		x1, y1, x2, y2, color, dash)
}

// drawComponent dispatches to the appropriate shape renderer.
func drawComponent(cl compLayout) string {
	cx := cl.p.x + float64(compSize)/2
	cy := cl.p.y + float64(compSize)/2

	var shape string
	switch cl.part.Kind {
	case metaphor.PartGear:
		shape = drawGear(cx, cy, float64(compSize)*0.42, float64(compSize)*0.30,
			float64(compSize)*0.13, toothCount(cl.c))
	case metaphor.PartLever:
		shape = drawLever(cl.p.x, cl.p.y, float64(compSize))
	case metaphor.PartAdapter:
		shape = drawHexAdapter(cx, cy, float64(compSize)*0.43)
	case metaphor.PartContainer:
		shape = drawContainer(cl.p.x, cl.p.y, float64(compSize))
	case metaphor.PartPipe:
		shape = drawPipe(cl.p.x, cl.p.y, float64(compSize))
	case metaphor.PartBelt:
		shape = drawBelt(cl.p.x, cl.p.y, float64(compSize))
	case metaphor.PartGauge:
		shape = drawGauge(cx, cy, float64(compSize)*0.42)
	case metaphor.PartWeight:
		shape = drawWeight(cl.p.x, cl.p.y, float64(compSize))
	default:
		shape = drawGauge(cx, cy, float64(compSize)*0.42)
	}

	label := drawLabel(cx, cl.p.y+float64(compSize)+14, cl.c.Name, cl.c.Exported)
	icon := drawIcon(cl.p.x+float64(compSize)-18, cl.p.y+2, cl.part.Icon)
	return shape + label + icon
}

// ─── Shape renderers ──────────────────────────────────────────────────────────

// drawGear renders a toothed gear using a polygon path.
func drawGear(cx, cy, rOuter, rInner, rHub float64, nTeeth int) string {
	points := gearPoints(cx, cy, rOuter, rInner, nTeeth)
	var b strings.Builder
	writef(&b, `  <polygon points="%s" fill="url(#gearGrad)" `+
		`stroke="#6B3E10" stroke-width="1.5" filter="url(#shadow)"/>`+"\n", points)
	// Spoke lines
	for i := 0; i < nTeeth; i += 2 {
		a := float64(i) * math.Pi * 2 / float64(nTeeth)
		writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
			`stroke="#6B3E10" stroke-width="1" opacity="0.5"/>`+"\n",
			cx+rHub*math.Cos(a), cy+rHub*math.Sin(a),
			cx+(rInner-4)*math.Cos(a), cy+(rInner-4)*math.Sin(a))
	}
	// Hub circle
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="%.2f" `+
		`fill="#f0d8a0" stroke="#6B3E10" stroke-width="1.5"/>`+"\n", cx, cy, rHub)
	// Axle
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="3" fill="#3a2a1a"/>`+"\n", cx, cy)
	return b.String()
}

// gearPoints computes the polygon point string for a gear.
func gearPoints(cx, cy, rOuter, rInner float64, nTeeth int) string {
	step := math.Pi * 2 / float64(nTeeth)
	half := step * 0.22 // half-width of each tooth at the pitch circle

	pts := make([]string, 0, nTeeth*4)
	for i := 0; i < nTeeth; i++ {
		base := float64(i)*step - math.Pi/2
		// Ascending flank
		a1 := base - half*0.9
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", cx+rInner*math.Cos(a1), cy+rInner*math.Sin(a1)))
		a2 := base - half
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", cx+rOuter*math.Cos(a2), cy+rOuter*math.Sin(a2)))
		// Tooth top
		a3 := base + half
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", cx+rOuter*math.Cos(a3), cy+rOuter*math.Sin(a3)))
		// Descending flank
		a4 := base + half*0.9
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", cx+rInner*math.Cos(a4), cy+rInner*math.Sin(a4)))
	}
	return strings.Join(pts, " ")
}

// drawLever renders a steel lever on a triangular fulcrum.
func drawLever(x, y, size float64) string {
	mx := x + size/2
	my := y + size/2
	halfLen := size * 0.45

	var b strings.Builder
	// Fulcrum triangle
	triH := size * 0.20
	writef(&b, `  <polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f" `+
		`fill="#888" stroke="#555" stroke-width="1.5" filter="url(#shadow)"/>`+"\n",
		mx, my+triH*0.1, mx-triH*0.6, my+triH, mx+triH*0.6, my+triH)
	// Lever bar
	writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="4" `+
		`fill="url(#leverGrad)" stroke="#555" stroke-width="1.5" filter="url(#shadow)"/>`+"\n",
		mx-halfLen, my-7, halfLen*2, 14.0)
	// Pivot circle
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="6" `+
		`fill="#c0c0c0" stroke="#555" stroke-width="1.5"/>`+"\n", mx, my)
	// End balls
	for _, ex := range []float64{mx - halfLen + 6, mx + halfLen - 6} {
		writef(&b, `  <circle cx="%.2f" cy="%.2f" r="5" fill="#aaa" stroke="#555" stroke-width="1"/>`+"\n", ex, my)
	}
	return b.String()
}

// drawHexAdapter renders a golden hexagonal adapter socket.
func drawHexAdapter(cx, cy, r float64) string {
	pts := hexPoints(cx, cy, r)
	innerPts := hexPoints(cx, cy, r*0.65)

	var b strings.Builder
	writef(&b, `  <polygon points="%s" fill="url(#adapterGrad)" `+
		`stroke="#7a5000" stroke-width="2" filter="url(#shadow)"/>`+"\n", pts)
	writef(&b, `  <polygon points="%s" fill="none" `+
		`stroke="#7a5000" stroke-width="1" opacity="0.6"/>`+"\n", innerPts)
	// Contact pins
	for i := 0; i < 6; i++ {
		a := float64(i)*math.Pi/3 - math.Pi/6
		px := cx + r*0.82*math.Cos(a)
		py := cy + r*0.82*math.Sin(a)
		writef(&b, `  <circle cx="%.2f" cy="%.2f" r="3" fill="#ffe080" stroke="#7a5000" stroke-width="1"/>`+"\n", px, py)
	}
	// Centre hole
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="#3a2a0a" opacity="0.8"/>`+"\n", cx, cy, r*0.18)
	return b.String()
}

func hexPoints(cx, cy, r float64) string {
	pts := make([]string, 6)
	for i := 0; i < 6; i++ {
		a := float64(i)*math.Pi/3 - math.Pi/6
		pts[i] = fmt.Sprintf("%.2f,%.2f", cx+r*math.Cos(a), cy+r*math.Sin(a))
	}
	return strings.Join(pts, " ")
}

// drawContainer renders a 3-D storage box using isometric faces.
func drawContainer(x, y, size float64) string {
	w := size * 0.75
	h := size * 0.55
	d := size * 0.20 // depth offset

	// Front face
	fx, fy := x+(size-w)/2, y+(size-h)/2+d*0.5
	// Top face
	tx, ty := fx, fy-d*0.8

	var b strings.Builder
	// Side face (right)
	writef(&b, `  <polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" `+
		`fill="#3a6a3a" stroke="#2a4a2a" stroke-width="1" filter="url(#shadow)"/>`+"\n",
		fx+w, fy, fx+w+d, fy-d*0.8, fx+w+d, fy+h-d*0.8, fx+w, fy+h)
	// Top face
	writef(&b, `  <polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" `+
		`fill="#7ab87a" stroke="#2a4a2a" stroke-width="1"/>`+"\n",
		tx, ty, tx+d, ty-d*0.8, tx+w+d, ty-d*0.8, tx+w, ty)
	// Front face
	writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" `+
		`fill="url(#containerGrad)" stroke="#2a4a2a" stroke-width="1.5"/>`+"\n",
		fx, fy, w, h)
	// Label line on front face
	writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
		`stroke="#2a4a2a" stroke-width="0.8" opacity="0.5"/>`+"\n",
		fx+4, fy+h*0.35, fx+w-4, fy+h*0.35)
	writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
		`stroke="#2a4a2a" stroke-width="0.8" opacity="0.5"/>`+"\n",
		fx+4, fy+h*0.60, fx+w-4, fy+h*0.60)
	return b.String()
}

// drawPipe renders a blue cylindrical pipe with flanged end-caps.
func drawPipe(x, y, size float64) string {
	rx := size * 0.42 // half-length of pipe
	ry := size * 0.16 // ellipse y-radius
	cx := x + size/2
	cy := y + size/2

	var b strings.Builder
	// Pipe body
	writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" `+
		`fill="url(#pipeGrad)" stroke="#1a4870" stroke-width="1.5" filter="url(#shadow)"/>`+"\n",
		cx-rx, cy-ry, rx*2, ry*2)
	// Right cap
	writef(&b, `  <ellipse cx="%.2f" cy="%.2f" rx="%.2f" ry="%.2f" `+
		`fill="#2d6fa0" stroke="#1a4870" stroke-width="1.5"/>`+"\n",
		cx+rx, cy, ry*0.6, ry)
	// Left cap (dark – interior)
	writef(&b, `  <ellipse cx="%.2f" cy="%.2f" rx="%.2f" ry="%.2f" `+
		`fill="#1a3a5a" stroke="#1a4870" stroke-width="1.5"/>`+"\n",
		cx-rx, cy, ry*0.6, ry)
	// Flow arrow inside pipe
	arrowY := cy
	writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
		`stroke="#c8e8ff" stroke-width="1.5" stroke-dasharray="6,4" opacity="0.7"/>`+"\n",
		cx-rx*0.6, arrowY, cx+rx*0.5, arrowY)
	writef(&b, `  <polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="#c8e8ff" opacity="0.7"/>`+"\n",
		cx+rx*0.5, arrowY-4, cx+rx*0.5+8, arrowY, cx+rx*0.5, arrowY+4)
	// Flange lines
	for _, fx := range []float64{cx - rx*0.75, cx + rx*0.75} {
		writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
			`stroke="#1a4870" stroke-width="2"/>`+"\n",
			fx, cy-ry, fx, cy+ry)
	}
	return b.String()
}

// drawBelt renders a conveyor belt with two drive rollers and items on the belt.
func drawBelt(x, y, size float64) string {
	rr := size * 0.14 // roller radius
	bw := size * 0.35 // belt half-width (distance between roller centres)
	cy := y + size/2
	lx := x + size/2 - bw // left roller cx
	rx := x + size/2 + bw // right roller cx

	var b strings.Builder
	// Belt top and bottom runs
	writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" `+
		`fill="url(#beltGrad)" stroke="#2a2a2a" stroke-width="1.5" filter="url(#shadow)"/>`+"\n",
		lx, cy-rr, bw*2, rr*2)
	// Rollers
	for _, rcx := range []float64{lx, rx} {
		writef(&b, `  <circle cx="%.2f" cy="%.2f" r="%.2f" `+
			`fill="#909090" stroke="#2a2a2a" stroke-width="1.5"/>`+"\n", rcx, cy, rr)
		writef(&b, `  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="#2a2a2a"/>`+"\n", rcx, cy, rr*0.3)
	}
	// Items (small boxes) on the belt
	for i := 0; i < 3; i++ {
		ix := lx + rr*0.8 + float64(i)*(bw*2-rr*1.6)/2.5
		writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" `+
			`fill="#e8b860" stroke="#885520" stroke-width="1" rx="1"/>`+"\n",
			ix, cy-rr+2, rr*0.9, rr*1.1)
	}
	// Motion arrow
	writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
		`stroke="#fff" stroke-width="1" stroke-dasharray="4,3" opacity="0.5"/>`+"\n",
		lx+rr, cy+rr+6, rx-rr, cy+rr+6)
	return b.String()
}

// drawGauge renders a circular pressure gauge with a needle.
func drawGauge(cx, cy, r float64) string {
	var b strings.Builder
	// Outer ring
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="%.2f" `+
		`fill="url(#gaugeGrad)" stroke="#6a4800" stroke-width="2" filter="url(#shadow)"/>`+"\n",
		cx, cy, r)
	// Dial face (lighter)
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="#fff8e8" stroke="#9a7030" stroke-width="1"/>`+"\n",
		cx, cy, r*0.78)
	// Tick marks
	for i := 0; i <= 8; i++ {
		a := math.Pi*0.75 + float64(i)*math.Pi*1.5/8
		r1 := r * 0.78
		r2 := r * 0.65
		if i%2 == 0 {
			r2 = r * 0.60
		}
		writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#6a4800" stroke-width="1"/>`+"\n",
			cx+r1*math.Cos(a), cy+r1*math.Sin(a),
			cx+r2*math.Cos(a), cy+r2*math.Sin(a))
	}
	// Needle (pointing at ~40% of range)
	needleA := math.Pi * 0.75 * 1.5
	writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
		`stroke="#8B0000" stroke-width="2" stroke-linecap="round"/>`+"\n",
		cx, cy, cx+r*0.60*math.Cos(needleA), cy+r*0.60*math.Sin(needleA))
	// Centre bolt
	writef(&b, `  <circle cx="%.2f" cy="%.2f" r="3.5" fill="#6a4800"/>`+"\n", cx, cy)
	return b.String()
}

// drawWeight renders a calibrated cast-iron weight (trapezoid shape).
func drawWeight(x, y, size float64) string {
	w := size * 0.62
	h := size * 0.50
	bx := x + (size-w)/2
	by := y + (size-h)/2 + 5
	narrow := w * 0.60
	handleH := h * 0.22

	var b strings.Builder
	// Handle at top
	writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="3" `+
		`fill="url(#weightGrad)" stroke="#3a4a5a" stroke-width="1.5" filter="url(#shadow)"/>`+"\n",
		bx+(w-narrow)/2, by, narrow, handleH)
	// Body (trapezoid via polygon)
	writef(&b, `  <polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" `+
		`fill="url(#weightGrad)" stroke="#3a4a5a" stroke-width="1.5"/>`+"\n",
		bx+(w-narrow)/2, by+handleH,
		bx+(w+narrow)/2, by+handleH,
		bx+w, by+h,
		bx, by+h)
	// Etched value line
	writef(&b, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" `+
		`stroke="#3a4a5a" stroke-width="0.8" opacity="0.6"/>`+"\n",
		bx+6, by+h*0.65, bx+w-6, by+h*0.65)
	return b.String()
}

// miniGear draws a small decorative gear for the header.
func miniGear(cx, cy, r float64) string {
	pts := gearPoints(cx, cy, r, r*0.70, 8)
	return fmt.Sprintf(`<polygon points="%s" fill="#c8a860" stroke="#8B6820" stroke-width="1" opacity="0.7"/>`,
		pts)
}

// drawLabel renders the construct name below a component.
func drawLabel(cx, y float64, name string, exported bool) string {
	style := `font-family="Georgia, serif" font-size="11"`
	fill := `fill="#2a1a0a"`
	if exported {
		style = `font-family="Georgia, serif" font-size="11" font-weight="bold"`
		fill = `fill="#1a0a00"`
	}
	display := name
	if len([]rune(display)) > 14 {
		display = string([]rune(display)[:13]) + "…"
	}
	return fmt.Sprintf(`  <text x="%.2f" y="%.2f" text-anchor="middle" %s %s>%s</text>`+"\n",
		cx, y, style, fill, xmlEscape(display))
}

// drawIcon places the Unicode icon glyph in the top-right corner of a component.
func drawIcon(x, y float64, icon string) string {
	return fmt.Sprintf(`  <text x="%.2f" y="%.2f" `+
		`font-family="sans-serif" font-size="13" fill="#3a2a1a" opacity="0.7">%s</text>`+"\n",
		x, y+13, xmlEscape(icon))
}

// drawLegend renders the machine-part legend in the lower-left corner.
func drawLegend(x, y float64) string {
	var b strings.Builder
	lw := float64(440)
	lh := float64(legendH - 10)

	writef(&b, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="4" `+
		`fill="#f0e8d0" stroke="#8B7355" stroke-width="1.5" opacity="0.9"/>`+"\n",
		x, y, lw, lh)
	writef(&b, `  <text x="%.2f" y="%.2f" `+
		`font-family="Georgia, serif" font-size="12" font-weight="bold" fill="#2a1a0a">`+
		`Machine-Part Legend</text>`+"\n",
		x+8, y+16)

	items := []struct{ icon, label string }{
		{"⚙", "Gear – function"},
		{"⇋", "Lever – method"},
		{"🔌", "Adapter – interface"},
		{"📦", "Container – struct"},
		{"≋", "Pipe – channel"},
		{"⟶", "Belt – goroutine func"},
		{"🔆", "Gauge – variable"},
		{"⚖", "Weight – constant"},
	}
	cols := 2
	perCol := (len(items) + cols - 1) / cols
	for i, item := range items {
		col := i / perCol
		row := i % perCol
		tx := x + 12 + float64(col)*(lw/float64(cols))
		ty := y + 30 + float64(row)*17
		writef(&b, `  <text x="%.2f" y="%.2f" font-family="sans-serif" font-size="12" fill="#2a1a0a">%s  %s</text>`+"\n",
			tx, ty, xmlEscape(item.icon), xmlEscape(item.label))
	}
	return b.String()
}

// ─── Helper functions ────────────────────────────────────────────────────────

// toothCount returns the number of gear teeth based on the function arity.
func toothCount(c analyzer.ConstructInfo) int {
	n := len(c.Params)
	switch {
	case n == 0:
		return 6
	case n <= 2:
		return 8
	case n <= 4:
		return 10
	default:
		return 12
	}
}

// xmlEscape replaces characters that are invalid in SVG text content.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

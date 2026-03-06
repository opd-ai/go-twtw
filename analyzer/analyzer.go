// Package analyzer parses Go source packages using go/ast and extracts
// constructs (functions, methods, types, interfaces, variables, constants)
// together with their documentation comments and structural metadata.
// The extracted data is used downstream by the metaphor and report packages
// to produce machine-part illustrations inspired by The Way Things Work.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ConstructKind identifies the syntactic category of a Go construct.
type ConstructKind string

const (
	// KindFunction is a top-level function declaration.
	KindFunction ConstructKind = "function"
	// KindMethod is a function with a receiver (method on a type).
	KindMethod ConstructKind = "method"
	// KindStruct is a named struct type.
	KindStruct ConstructKind = "struct"
	// KindInterface is a named interface type.
	KindInterface ConstructKind = "interface"
	// KindType is any other named type (alias, channel type, basic type, etc.).
	KindType ConstructKind = "type"
	// KindVar is a package-level variable declaration.
	KindVar ConstructKind = "var"
	// KindConst is a package-level constant declaration.
	KindConst ConstructKind = "const"
)

// ParamInfo describes a single parameter or result in a function signature.
type ParamInfo struct {
	// Names holds the parameter names (may be empty for unnamed params/results).
	Names []string
	// Type is the string representation of the parameter type.
	Type string
}

// FieldInfo describes a single field in a struct type.
type FieldInfo struct {
	// Name is the field name. For embedded fields this is the type name.
	Name string
	// Type is the string representation of the field type.
	Type string
	// Doc is the documentation comment for the field.
	Doc string
	// Tag is the struct tag value (raw string literal, including backticks).
	Tag string
}

// ConstructInfo holds all extracted information about a single Go construct.
type ConstructInfo struct {
	// Name is the identifier declared by this construct.
	Name string
	// Kind is the syntactic category.
	Kind ConstructKind
	// Doc is the trimmed doc-comment text from the source.
	Doc string
	// Exported reports whether the construct name begins with an uppercase letter.
	Exported bool

	// Receiver is the type name for method constructs (empty for functions).
	Receiver string
	// Params are the function/method parameter descriptions.
	Params []ParamInfo
	// Results are the function/method result descriptions.
	Results []ParamInfo

	// Fields holds the struct fields for KindStruct constructs.
	Fields []FieldInfo
	// Methods holds the method names for KindInterface constructs.
	Methods []string

	// Underlying is the underlying type string for KindType/KindVar/KindConst.
	Underlying string

	// HasChannels is true when any parameter, result, or field involves a channel type.
	HasChannels bool
	// SpawnsGoroutines is true when the function body contains at least one go statement.
	SpawnsGoroutines bool
}

// PackageInfo holds all information extracted from a single Go package directory.
type PackageInfo struct {
	// Name is the package name as declared in the source.
	Name string
	// ImportPath is the module-relative import path (best-effort; may be the dir).
	ImportPath string
	// Dir is the filesystem directory of the package.
	Dir string
	// Doc is the trimmed package-level doc comment.
	Doc string
	// Constructs is the list of extracted constructs in declaration order.
	Constructs []ConstructInfo
	// Files lists the base names of the parsed source files.
	Files []string
	// Imports lists the unique import paths used by the package, sorted.
	Imports []string
}

// Analyze parses the Go package located at dir and returns a PackageInfo.
// Test files (_test.go) are excluded. It is not an error if the directory
// contains no Go source files – an empty PackageInfo is returned instead.
func Analyze(dir string) (*PackageInfo, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("analyzer: resolving dir %q: %w", dir, err)
	}

	fset := token.NewFileSet()
	pkgMap, err := parser.ParseDir(fset, abs, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("analyzer: parsing %q: %w", abs, err)
	}

	// Pick the first non-test package (prefer the non-"main" package when
	// multiple are present, which is unusual but valid).
	var chosen *ast.Package
	for name, p := range pkgMap {
		if strings.HasSuffix(name, "_test") {
			continue
		}
		if chosen == nil || name != "main" {
			chosen = p
		}
	}
	if chosen == nil {
		// No non-test package found; return a minimal stub.
		return &PackageInfo{Dir: abs, ImportPath: abs}, nil
	}

	return buildPackageInfo(chosen, abs, fset), nil
}

// AnalyzeRecursive walks dir and all its subdirectories, calling Analyze on
// each directory that contains at least one non-test .go file. It returns all
// successfully analysed packages and the first error encountered, if any.
func AnalyzeRecursive(dir string) ([]*PackageInfo, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("analyzer: resolving dir %q: %w", dir, err)
	}

	var pkgs []*PackageInfo
	var firstErr error

	err = filepath.Walk(abs, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() {
			return nil
		}
		// Skip hidden directories (e.g. .git) and common vendor/testdata paths.
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
			return filepath.SkipDir
		}

		pkg, analyzeErr := Analyze(path)
		if analyzeErr != nil {
			if firstErr == nil {
				firstErr = analyzeErr
			}
			return nil
		}
		if pkg.Name != "" {
			pkgs = append(pkgs, pkg)
		}
		return nil
	})
	if err != nil {
		return pkgs, err
	}
	return pkgs, firstErr
}

// buildPackageInfo constructs a PackageInfo from a parsed ast.Package.
func buildPackageInfo(pkg *ast.Package, dir string, fset *token.FileSet) *PackageInfo {
	info := &PackageInfo{
		Name:       pkg.Name,
		Dir:        dir,
		ImportPath: dir,
	}

	// Collect ordered file list and ast.File pointers.
	var files []*ast.File
	var fileNames []string
	for name, f := range pkg.Files {
		fileNames = append(fileNames, filepath.Base(name))
		files = append(files, f)
	}
	sort.Strings(fileNames)
	info.Files = fileNames

	// Extract constructs BEFORE doc.NewFromFiles, because that function may
	// trim unexported declarations from the AST in-place when AllDecls mode
	// is not set, causing unexported constructs to disappear.
	for _, f := range files {
		info.Constructs = append(info.Constructs, extractFromFile(f)...)
	}

	// Use go/doc for the package-level documentation comment.
	// NOTE: must be called after construct extraction (see above).
	if docPkg, err := doc.NewFromFiles(fset, files, ""); err == nil {
		info.Doc = strings.TrimSpace(docPkg.Doc)
	}

	// Collect unique imports.
	importSet := make(map[string]bool)
	for _, f := range files {
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			importSet[path] = true
		}
	}
	for imp := range importSet {
		info.Imports = append(info.Imports, imp)
	}
	sort.Strings(info.Imports)

	return info
}

// extractFromFile iterates over all top-level declarations in a file.
func extractFromFile(f *ast.File) []ConstructInfo {
	var out []ConstructInfo
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			out = append(out, extractFunc(d))
		case *ast.GenDecl:
			out = append(out, extractGenDecl(d)...)
		}
	}
	return out
}

// extractFunc extracts a ConstructInfo from a function or method declaration.
func extractFunc(d *ast.FuncDecl) ConstructInfo {
	c := ConstructInfo{
		Name:     d.Name.Name,
		Exported: ast.IsExported(d.Name.Name),
	}
	if d.Doc != nil {
		c.Doc = strings.TrimSpace(d.Doc.Text())
	}

	if d.Recv != nil && len(d.Recv.List) > 0 {
		c.Kind = KindMethod
		c.Receiver = typeString(d.Recv.List[0].Type)
	} else {
		c.Kind = KindFunction
	}

	if d.Type.Params != nil {
		for _, field := range d.Type.Params.List {
			p := ParamInfo{Type: typeString(field.Type)}
			for _, name := range field.Names {
				p.Names = append(p.Names, name.Name)
			}
			c.Params = append(c.Params, p)
		}
	}
	if d.Type.Results != nil {
		for _, field := range d.Type.Results.List {
			r := ParamInfo{Type: typeString(field.Type)}
			for _, name := range field.Names {
				r.Names = append(r.Names, name.Name)
			}
			c.Results = append(c.Results, r)
		}
	}

	c.HasChannels = fieldListHasChan(d.Type.Params) || fieldListHasChan(d.Type.Results)
	if d.Body != nil {
		c.SpawnsGoroutines = bodyHasGoStmt(d.Body)
	}
	return c
}

// extractGenDecl extracts ConstructInfos from a general declaration (type,
// var, or const group).
func extractGenDecl(d *ast.GenDecl) []ConstructInfo {
	groupDoc := ""
	if d.Doc != nil {
		groupDoc = strings.TrimSpace(d.Doc.Text())
	}

	var out []ConstructInfo
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			c := extractTypeSpec(s, groupDoc)
			out = append(out, c)
		case *ast.ValueSpec:
			kind := KindVar
			if d.Tok == token.CONST {
				kind = KindConst
			}
			specDoc := groupDoc
			if s.Doc != nil {
				specDoc = strings.TrimSpace(s.Doc.Text())
			}
			for _, name := range s.Names {
				c := ConstructInfo{
					Name:     name.Name,
					Kind:     kind,
					Doc:      specDoc,
					Exported: ast.IsExported(name.Name),
				}
				if s.Comment != nil {
					c.Doc = strings.TrimSpace(s.Comment.Text())
				}
				if s.Type != nil {
					c.Underlying = typeString(s.Type)
					c.HasChannels = isChanExpr(s.Type)
				}
				out = append(out, c)
			}
		}
	}
	return out
}

// extractTypeSpec converts a single *ast.TypeSpec into a ConstructInfo.
func extractTypeSpec(s *ast.TypeSpec, groupDoc string) ConstructInfo {
	c := ConstructInfo{
		Name:     s.Name.Name,
		Exported: ast.IsExported(s.Name.Name),
		Doc:      groupDoc,
	}
	if s.Comment != nil {
		c.Doc = strings.TrimSpace(s.Comment.Text())
	}

	switch t := s.Type.(type) {
	case *ast.StructType:
		c.Kind = KindStruct
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fi := FieldInfo{Type: typeString(field.Type)}
				if field.Doc != nil {
					fi.Doc = strings.TrimSpace(field.Doc.Text())
				} else if field.Comment != nil {
					fi.Doc = strings.TrimSpace(field.Comment.Text())
				}
				if field.Tag != nil {
					fi.Tag = field.Tag.Value
				}
				if len(field.Names) == 0 {
					// Embedded field: use type name as field name.
					fi.Name = typeString(field.Type)
					c.Fields = append(c.Fields, fi)
				} else {
					for _, name := range field.Names {
						f2 := fi
						f2.Name = name.Name
						c.Fields = append(c.Fields, f2)
					}
				}
			}
		}
	case *ast.InterfaceType:
		c.Kind = KindInterface
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				if len(method.Names) > 0 {
					for _, name := range method.Names {
						c.Methods = append(c.Methods, name.Name)
					}
				} else {
					// Embedded interface.
					c.Methods = append(c.Methods, typeString(method.Type))
				}
			}
		}
	default:
		c.Kind = KindType
		c.Underlying = typeString(s.Type)
		c.HasChannels = isChanExpr(s.Type)
	}
	return c
}

// typeString converts an ast.Expr to a human-readable type string.
func typeString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return "[N]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + typeString(t.Value)
		case ast.RECV:
			return "<-chan " + typeString(t.Value)
		default:
			return "chan " + typeString(t.Value)
		}
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	case *ast.ParenExpr:
		return "(" + typeString(t.X) + ")"
	}
	return "unknown"
}

// isChanExpr reports whether expr is (or contains) a channel type at the top level.
func isChanExpr(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.ChanType:
		return true
	case *ast.StarExpr:
		return isChanExpr(t.X)
	}
	return false
}

// fieldListHasChan reports whether any field in the list is a channel type.
func fieldListHasChan(fl *ast.FieldList) bool {
	if fl == nil {
		return false
	}
	for _, field := range fl.List {
		if isChanExpr(field.Type) {
			return true
		}
	}
	return false
}

// bodyHasGoStmt reports whether the function body contains at least one go statement.
func bodyHasGoStmt(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.GoStmt); ok {
			found = true
			return false
		}
		return !found
	})
	return found
}

package scanner

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/jtoloui/depviz/internal/config"
)

var _ Scanner = (*GoScanner)(nil)

type GoScanner struct {
	cfg *config.Config
}

func NewGoScanner(cfg *config.Config) *GoScanner {
	return &GoScanner{cfg: cfg}
}

func (g *GoScanner) Scan(root string) ([]FileImports, error) {
	skip := toSet(g.cfg.Exclude)

	include := func(path string, info os.FileInfo) bool {
		return !info.IsDir() &&
			strings.HasSuffix(path, ".go") &&
			!strings.HasSuffix(path, "_test.go")
	}

	return walkAndParse(root, skip, include, parseGoFile)
}

func parseGoFile(root, path string) (*FileImports, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}

	var imports []string
	var details []ImportDetail
	for _, imp := range file.Imports {
		modPath := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, modPath)

		d := ImportDetail{Path: modPath, Kind: ImportNamed}
		if imp.Name != nil {
			switch imp.Name.Name {
			case "_":
				d.Kind = ImportBlank
			case ".":
				d.Kind = ImportDot
			default:
				d.Kind = ImportAlias
				d.Alias = imp.Name.Name
			}
		}

		start := fset.Position(imp.Pos()).Offset
		end := fset.Position(imp.End()).Offset
		if start >= 0 && end <= len(src) {
			d.Snippet = string(src[start:end])
		}
		d.Line = fset.Position(imp.Pos()).Line
		details = append(details, d)
	}

	var exports []ExportDetail
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				continue // skip methods
			}
			exports = append(exports, ExportDetail{
				Name: d.Name.Name, Kind: ExportFunction, Private: !d.Name.IsExported(),
				Line: fset.Position(d.Pos()).Line,
			})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					exports = append(exports, ExportDetail{
						Name: s.Name.Name, Kind: ExportType, Private: !s.Name.IsExported(),
						Line: fset.Position(s.Pos()).Line,
					})
				case *ast.ValueSpec:
					kind := ExportVar
					if d.Tok == token.CONST {
						kind = ExportConst
					}
					for _, name := range s.Names {
						if name.Name == "_" {
							continue
						}
						exports = append(exports, ExportDetail{
							Name: name.Name, Kind: kind, Private: !name.IsExported(),
							Line: fset.Position(name.Pos()).Line,
						})
					}
				}
			}
		}
	}

	if len(imports) == 0 && len(exports) == 0 {
		return nil, nil
	}

	rel, _ := filepath.Rel(root, path)
	return &FileImports{File: rel, Lang: "go", Imports: imports, Details: details, Exports: exports, Lines: bytes.Count(src, []byte{'\n'}) + 1}, nil
}

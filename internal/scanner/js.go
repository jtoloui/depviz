package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jtoloui/depviz/internal/config"
)

var _ Scanner = (*JSScanner)(nil)

var jsImportRe = regexp.MustCompile(`(?:import\s.*?from\s+['"](.+?)['"]|require\s*\(\s*['"](.+?)['"]\s*\))`)

type JSScanner struct {
	cfg *config.Config
}

func NewJSScanner(cfg *config.Config) *JSScanner {
	return &JSScanner{cfg: cfg}
}

func (j *JSScanner) Scan(root string) ([]FileImports, error) {
	exts := map[string]bool{".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".mjs": true}
	skip := toSet(j.cfg.Exclude)

	include := func(path string, info os.FileInfo) bool {
		return !info.IsDir() && exts[filepath.Ext(path)]
	}

	return walkAndParse(root, skip, include, parseJSFile)
}

func parseJSFile(root, path string) (*FileImports, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var imports []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		for _, m := range jsImportRe.FindAllStringSubmatch(s.Text(), -1) {
			if m[1] != "" {
				imports = append(imports, m[1])
			} else if m[2] != "" {
				imports = append(imports, m[2])
			}
		}
	}

	if len(imports) == 0 {
		return nil, nil
	}

	rel, _ := filepath.Rel(root, path)
	return &FileImports{File: rel, Imports: imports}, nil
}

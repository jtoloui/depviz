package scanner

import "github.com/jtoloui/depviz/internal/config"

var _ Scanner = (*MultiScanner)(nil)

// MultiScanner delegates to GoScanner and TreeSitterScanner, merging results.
type MultiScanner struct {
	go_ *GoScanner
	js  *TreeSitterScanner
}

func NewMultiScanner(cfg *config.Config) *MultiScanner {
	return &MultiScanner{go_: NewGoScanner(cfg), js: NewTreeSitterScanner(cfg)}
}

func (m *MultiScanner) Scan(root string) ([]FileImports, error) {
	goFiles, err := m.go_.Scan(root)
	if err != nil {
		return nil, err
	}
	jsFiles, err := m.js.Scan(root)
	if err != nil {
		return nil, err
	}
	return append(goFiles, jsFiles...), nil
}

package scanner

// ImportKind describes how a module is imported.
type ImportKind string

const (
	ImportDefault     ImportKind = "default"
	ImportNamed       ImportKind = "named"
	ImportNamespace   ImportKind = "namespace"
	ImportSideEffect  ImportKind = "side-effect"
	ImportRequire     ImportKind = "require"
	ImportDynamic     ImportKind = "dynamic"
	ImportReExport    ImportKind = "re-export"
	ImportReExportAll ImportKind = "re-export-all"
	ImportType        ImportKind = "type"
	// Go-specific
	ImportBlank ImportKind = "blank"
	ImportDot   ImportKind = "dot"
	ImportAlias ImportKind = "alias"
)

// ImportDetail captures what is imported from a module.
type ImportDetail struct {
	Path    string     `json:"path"`
	Kind    ImportKind `json:"kind"`
	Names   []string   `json:"names,omitempty"`
	Alias   string     `json:"alias,omitempty"`
	Snippet string     `json:"snippet,omitempty"`
	Line    int        `json:"line,omitempty"`
}

// ExportKind describes how a symbol is exported.
type ExportKind string

const (
	ExportFunction  ExportKind = "func"
	ExportClass     ExportKind = "class"
	ExportConst     ExportKind = "const"
	ExportVar       ExportKind = "var"
	ExportType      ExportKind = "type"
	ExportInterface ExportKind = "interface"
	ExportDefault   ExportKind = "default"
	ExportNamed     ExportKind = "named"
	ExportReExport  ExportKind = "re-export"
)

// ExportDetail captures a single exported symbol.
type ExportDetail struct {
	Name    string     `json:"name"`
	Kind    ExportKind `json:"kind"`
	Private bool       `json:"private,omitempty"`
	Line    int        `json:"line,omitempty"`
}

// FileImports represents a file and its imports.
type FileImports struct {
	File    string         `json:"file"`
	Lang    string         `json:"-"`
	Imports []string       `json:"imports"`
	Details []ImportDetail `json:"details,omitempty"`
	Exports []ExportDetail `json:"exports,omitempty"`
	Lines   int            `json:"lines,omitempty"`
}

// Scanner scans a project directory for imports.
type Scanner interface {
	Scan(root string) ([]FileImports, error)
}

// toSet converts a string slice to a map for O(1) lookups.
func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

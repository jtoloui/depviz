package render

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"io"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/scanner"
)

//go:embed template.html
var tmplHTML string

//go:embed styles.css
var cssContent string

//go:embed app.js
var jsContent string

var tmpl = template.Must(template.New("depviz").Parse(tmplHTML))

type classifiedImport struct {
	Name     string             `json:"name"`
	Category config.Category    `json:"category"`
	Kind     scanner.ImportKind `json:"kind,omitempty"`
	Names    []string           `json:"names,omitempty"`
	Alias    string             `json:"alias,omitempty"`
	Snippet  string             `json:"snippet,omitempty"`
	Line     int                `json:"line,omitempty"`
}

type exportData struct {
	Name    string             `json:"name"`
	Kind    scanner.ExportKind `json:"kind"`
	Private bool               `json:"private,omitempty"`
	Line    int                `json:"line,omitempty"`
}

type fileData struct {
	File    string             `json:"file"`
	Imports []classifiedImport `json:"imports"`
	Exports []exportData       `json:"exports,omitempty"`
	Lines   int                `json:"lines,omitempty"`
}

type templateData struct {
	DataJSON template.JS
	Root     string
	CSS      template.CSS
	JS       template.JS
}

// HTML writes a dependency visualisation to w.
func HTML(w io.Writer, root string, results []scanner.FileImports, cl *classify.Classifier) error {
	files := make([]fileData, len(results))
	for i, r := range results {
		imps := make([]classifiedImport, len(r.Imports))
		for j, imp := range r.Imports {
			ci := classifiedImport{Name: imp, Category: cl.ClassifyWithLang(imp, r.Lang)}
			if j < len(r.Details) {
				d := r.Details[j]
				ci.Kind = d.Kind
				ci.Names = d.Names
				ci.Alias = d.Alias
				ci.Snippet = d.Snippet
				ci.Line = d.Line
			}
			imps[j] = ci
		}
		files[i] = fileData{File: r.File, Imports: imps, Lines: r.Lines}
		if len(r.Exports) > 0 {
			exports := make([]exportData, len(r.Exports))
			for k, e := range r.Exports {
				exports[k] = exportData{Name: e.Name, Kind: e.Kind, Private: e.Private, Line: e.Line}
			}
			files[i].Exports = exports
		}
	}

	data, err := json.Marshal(files)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, templateData{
		DataJSON: template.JS(data),
		Root:     root,
		CSS:      template.CSS(cssContent),
		JS:       template.JS(jsContent),
	})
}

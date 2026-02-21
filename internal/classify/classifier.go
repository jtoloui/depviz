package classify

import (
	"regexp"
	"strings"

	"github.com/jtoloui/depviz/internal/config"
)

// Classifier categorises import paths based on config rules.
type Classifier struct {
	lang     string
	internal []*regexp.Regexp
	private  []*regexp.Regexp
}

// New compiles the patterns from cfg and returns a ready Classifier.
func New(cfg *config.Config) (*Classifier, error) {
	internal, err := compileAll(cfg.Classify.Internal)
	if err != nil {
		return nil, err
	}

	private, err := compileAll(cfg.Classify.Private)
	if err != nil {
		return nil, err
	}

	return &Classifier{
		lang:     cfg.Language,
		internal: internal,
		private:  private,
	}, nil
}

// Classify returns the category for an import path.
func (c *Classifier) Classify(imp string) config.Category {
	return c.ClassifyWithLang(imp, c.lang)
}

// ClassifyWithLang categorises an import using the given language for stdlib detection.
func (c *Classifier) ClassifyWithLang(imp, lang string) config.Category {
	if isStdlibFor(imp, lang) {
		return config.Stdlib
	}
	if matchesAny(imp, c.internal) {
		return config.Internal
	}
	if matchesAny(imp, c.private) {
		return config.Private
	}
	return config.External
}

var nodeBuiltins = map[string]bool{
	"assert": true, "assert/strict": true, "async_hooks": true,
	"buffer": true, "child_process": true, "cluster": true,
	"console": true, "constants": true, "crypto": true,
	"dgram": true, "diagnostics_channel": true, "dns": true, "dns/promises": true,
	"domain": true, "events": true,
	"fs": true, "fs/promises": true,
	"http": true, "http2": true, "https": true,
	"inspector": true, "module": true, "net": true, "os": true,
	"path": true, "path/posix": true, "path/win32": true,
	"perf_hooks": true, "process": true, "punycode": true, "querystring": true,
	"readline": true, "readline/promises": true, "repl": true,
	"stream": true, "stream/consumers": true, "stream/promises": true, "stream/web": true,
	"string_decoder": true, "sys": true, "test": true,
	"timers": true, "timers/promises": true,
	"tls": true, "trace_events": true, "tty": true,
	"url": true, "util": true, "util/types": true,
	"v8": true, "vm": true, "wasi": true, "worker_threads": true, "zlib": true,
}

func isStdlibFor(imp, lang string) bool {
	switch lang {
	case "js":
		return nodeBuiltins[strings.TrimPrefix(imp, "node:")]
	case "go":
		return !strings.Contains(imp, ".")
	}
	return false
}

func matchesAny(s string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func compileAll(patterns []string) ([]*regexp.Regexp, error) {
	res := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		res[i] = re
	}
	return res, nil
}

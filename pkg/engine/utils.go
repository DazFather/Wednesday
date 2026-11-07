package engine

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
)

func splitExt(name string) (base, ext string) {
	ext = strings.ToLower(filepath.Ext(name))
	base = name[:len(name)-len(ext)]
	if ext == ".html" {
		ext = strings.ToLower(filepath.Ext(base)) + ext
		base = name[:len(name)-len(ext)]
	}
	return
}

func validHTML(content string) error {
	var dec = xml.NewDecoder(strings.NewReader(content))
	dec.Strict = false
	return dec.Decode(new(interface{}))
}

func (s Settings) bundleJS(opt *es.BuildOptions, mod ModuleType, page string, entries []string) error {
	outName := page + "-mini"
	if mod == ecmaModule {
		outName = page + "-ecma-mini"
	}
	opt.Outdir = ""
	opt.Outfile = s.ScriptPath(outName)

	if len(entries) <= 1 {
		return nil
	}

	var b strings.Builder

	if mod == ecmaModule {
		for _, e := range entries {
			b.WriteString(`import "./` + filepath.ToSlash(e) + "\";\n")
		}
	} else {
		for i, e := range entries {
			content, err := os.ReadFile(e)
			if err != nil {
				return fmt.Errorf("read JS entry %q for page %q: %w", e, page, err)
			}
			if i > 0 {
				b.WriteString("\n;")
			}
			b.Write(content)
			b.WriteString("\n")
		}
	}

	opt.EntryPoints = nil
	opt.Stdin = &es.StdinOptions{
		Contents:   b.String(),
		ResolveDir: ".",
		Sourcefile: outName + "-bundle.js",
		Loader:     es.LoaderJS,
	}
	return nil
}

func (s Settings) minifyJS(page string, mod ModuleType, entries []string) ([]string, error) {
	var opt = defESBuildOpt(entries)
	opt.Outdir = s.ScriptPath()

	switch mod {
	case ecmaModule:
		opt.Format = es.FormatESModule
		opt.Alias = map[string]string{
			"@wed/utils": "./" + filepath.ToSlash(s.ScriptPath("wed-utils.mjs")),
			"@wed/http":  "./" + filepath.ToSlash(s.ScriptPath("wed-http.mjs")),
		}
	case noModule:
		opt.Format = es.FormatDefault
		if !s.Minify {
			opt.Bundle = false
		}
	}

	if s.Minify {
		if err := s.bundleJS(opt, mod, page, entries); err != nil {
			return nil, err
		}
	}

	rgx := regexp.MustCompile(`\w+\.map$`)
	res, err := esbuild(opt, func(s string) bool { return !rgx.MatchString(s) })
	if err != nil {
		spec := "global"
		if s.Minify {
			spec = "single file"
		}
		return nil, fmt.Errorf("error douring %s JS minification of page %q: %w", spec, page, err)
	}
	return res, nil
}

func (s Settings) bundleCSS(opt *es.BuildOptions, page string, entries []string) {
	opt.Outfile = s.StylePath(page + "-mini")
	if len(entries) <= 1 {
		return
	}

	var b strings.Builder
	for _, e := range entries {
		b.WriteString(`@import "./` + filepath.ToSlash(e) + "\";\n")
	}
	opt.EntryPoints = nil
	opt.Stdin = &es.StdinOptions{
		Contents:   b.String(),
		ResolveDir: ".",
		Sourcefile: page + "-mini-entry.css",
		Loader:     es.LoaderCSS,
	}
}

func (s Settings) minifyCSS(page string, entries []string) ([]string, error) {
	var opt = defESBuildOpt(entries)

	if s.Minify {
		s.bundleCSS(opt, page, entries)
	} else {
		opt.Outdir = s.StylePath()
	}

	res, err := esbuild(opt, regexp.MustCompile(`\w+\.css$`).MatchString)
	if err != nil {
		spec := "global"
		if s.Minify {
			spec = "single file"
		}
		return nil, fmt.Errorf("error douring %s CSS minification of page %q: %w", spec, page, err)
	}
	return res, nil
}

func defESBuildOpt(entries []string) *es.BuildOptions {
	return &es.BuildOptions{
		Write:             true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		AllowOverwrite:    true,
		Bundle:            true,
		LogLevel:          es.LogLevelWarning,
		Sourcemap:         es.SourceMapLinked,
		EntryPoints:       entries,
	}
}

func esbuild(opt *es.BuildOptions, isAllowed func(string) bool) ([]string, error) {
	res := es.Build(*opt)

	if size := len(res.Errors); size > 0 {
		errs := make([]error, size)
		for i := range res.Errors {
			errs[i] = errors.New(res.Errors[i].Text)
		}
		return nil, errors.Join(errs...)
	}

	var output []string
	for _, f := range res.OutputFiles {
		if isAllowed(f.Path) {
			output = append(output, filepath.Base(f.Path))
		}
	}
	return output, nil
}

func (p *page) assetURL(asset string) string {
	prefix, err := filepath.Rel(filepath.Dir(p.Location), ".")
	if err != nil || prefix == "." {
		return "./" + asset
	}
	return filepath.ToSlash(prefix) + "/" + asset
}

func (p *page) StyleURL(name string) string {
	return p.assetURL(p.Settings.StyleURL(name))
}

func (p *page) ScriptURL(name string) string {
	return p.assetURL(p.Settings.ScriptURL(name))
}

func (p *page) StyleTag(name string) string {
	return `<link rel="stylesheet" href="` + p.StyleURL(name) + `" />`
}

func (p *page) ScriptTag(name string, deferred bool, modType ModuleType) string {
	d := ""
	if deferred {
		d = "defer "
	}

	return `<script ` + d + `type="` + string(modType) + `" src="` + p.ScriptURL(name) + `"></script>`
}

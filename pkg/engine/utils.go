package engine

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
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

func (s Settings) minifyJS(page string, mod ModuleType, entries []string, defers func(string) bool) (string, error) {
	var opt = esbuild.BuildOptions{
		EntryPoints:       entries,
		Bundle:            true,
		Write:             true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		AllowOverwrite:    true,
		Sourcemap:         esbuild.SourceMapLinked,
		Alias: map[string]string{
			"@wed/utils": "./" + filepath.ToSlash(s.ScriptPath("wed-utils.mjs")),
			"@wed/http":  "./" + filepath.ToSlash(s.ScriptPath("wed-http.mjs")),
		},
		LogLevel: esbuild.LogLevelWarning,
	}

	switch mod {
	case ecmaModule:
		opt.Format = esbuild.FormatESModule
		if s.Minify {
			opt.Outfile = s.ScriptPath(page + "-ecma-mini")
		} else {
			opt.Outdir = s.ScriptPath()
		}
	case noModule:
		opt.Format = esbuild.FormatDefault
		if s.Minify {
			opt.Outfile = s.ScriptPath(page + "-mini")
		} else {
			opt.Bundle = false
			opt.Alias = nil
			opt.Outdir = s.ScriptPath()
		}
	}

	res := esbuild.Build(opt)

	if size := len(res.Errors); size > 0 {
		errs := make([]error, size)
		for i := range res.Errors {
			errs[i] = errors.New(res.Errors[i].Text)
		}
		spec := "global"
		if s.Minify {
			spec = "single file"
		}
		return "", fmt.Errorf("%d esbuild errors douring %s JS minification of page %s: %w", size, spec, page, errors.Join(errs...))
	}

	var output strings.Builder
	for _, f := range res.OutputFiles {
		if name := filepath.Base(f.Path); strings.ToLower(filepath.Ext(name)) != ".map" {
			output.WriteString(s.ScriptTag(name, defers(name), &mod))
		}
	}

	return output.String(), nil
}

func (s Settings) minifyCSS(page string, entries []string) ([]string, error) {
	var opt = esbuild.BuildOptions{
		EntryPoints:       entries,
		Bundle:            true,
		Write:             true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		AllowOverwrite:    true,
		Sourcemap:         esbuild.SourceMapLinked,
		LogLevel:          esbuild.LogLevelWarning,
	}

	if s.Minify {
		opt.Outfile = s.StylePath(page + "-mini")
	} else {
		opt.Outdir = s.StylePath()
	}

	res := esbuild.Build(opt)

	if size := len(res.Errors); size > 0 {
		errs := make([]error, size)
		for i := range res.Errors {
			errs[i] = errors.New(res.Errors[i].Text)
		}

		spec := "global"
		if s.Minify {
			spec = "single file"
		}
		return nil, fmt.Errorf("%d esbuild errors douring CSS %s minification of page %s: %w", size, spec, page, errors.Join(errs...))
	}

	var output []string
	for _, f := range res.OutputFiles {
		if name := filepath.Base(f.Path); strings.ToLower(filepath.Ext(name)) == ".css" {
			output = append(output, name)
		}
	}

	return output, nil
}

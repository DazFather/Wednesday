package engine

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
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

func (s Settings) minifyJS(page string, entries []string) ([]string, error) {
	var opt = api.BuildOptions{
		EntryPoints:       entries,
		Bundle:            true,
		Write:             true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		AllowOverwrite:    true,
		Format:            api.FormatESModule,
		Sourcemap:         api.SourceMapLinked,
		Alias: map[string]string{
			"@wed/utils": "./" + filepath.ToSlash(s.ScriptPath("wed-utils.mjs")),
			"@wed/http":  "./" + filepath.ToSlash(s.ScriptPath("wed-http.mjs")),
		},
		LogLevel: api.LogLevelWarning,
	}

		opt.Outdir = s.ScriptPath()

	res := api.Build(opt)

	if size := len(res.Errors); size > 0 {
		errs := make([]error, size)
		for i := range res.Errors {
			errs[i] = errors.New(res.Errors[i].Text)
		}
		spec := "global"
		return nil, fmt.Errorf("%d esbuild errors douring %s JS minification of page %s: %w", size, spec, page, errors.Join(errs...))
	}

	var output []string
	for _, f := range res.OutputFiles {
		if name := filepath.Base(f.Path); strings.ToLower(filepath.Ext(name)) == ".js" {
			output = append(output, name)
		}
	}

	return output, nil
}

func (s Settings) minifyCSS(page string, entries []string) ([]string, error) {
	var opt = api.BuildOptions{
		EntryPoints:      entries,
		Bundle:           true,
		Write:            true,
		MinifyWhitespace: true,
		MinifySyntax:     true,
		AllowOverwrite:   true,
		Sourcemap:        api.SourceMapLinked,
		LogLevel:         api.LogLevelWarning,
	}

		opt.Outdir = s.StylePath()

	res := api.Build(opt)

	if size := len(res.Errors); size > 0 {
		errs := make([]error, size)
		for i := range res.Errors {
			errs[i] = errors.New(res.Errors[i].Text)
		}

		spec := "global"
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

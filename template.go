package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
)

type TemplateData struct {
	FileSettings
	StylePaths  []string
	ScriptPaths []string
	pages       []*template.Template
	components  []Component
}

var TemplateFuncs = template.FuncMap{
	"args": func(v ...any) []any { return v },
}

func NewTemplateData(s FileSettings) (td TemplateData) {
	return TemplateData{FileSettings: s}
}

func (td *TemplateData) WriteComponent(t *template.Template, c Component) (err error) {
	if err = c.WriteStyle(td.StylePath(c.Name + ".css")); err != nil {
		return
	}
	td.appendStyle(c.Name)

	if err = c.WriteScript(td.ScriptPath(c.Name + ".js")); err != nil {
		return
	}
	td.appendScript(c.Name)

	if !c.IsDynamic {
		_, err = t.New(c.Name).Parse(c.WrappedHTML())
	}
	return
}

func (td *TemplateData) appendStyle(name string) {
	link, err := url.JoinPath("style", name+".css")
	if err != nil {
		panic(err)
	}
	td.StylePaths = append(td.StylePaths, link)
}

func (td *TemplateData) appendScript(name string) {
	link, err := url.JoinPath("script", name+".js")
	if err != nil {
		panic(err)
	}
	td.ScriptPaths = append(td.ScriptPaths, link)
}

func (td TemplateData) Build() error {
	t := template.New("temp")
	for _, c := range td.components {
		if err := td.WriteComponent(t, c); err != nil {
			return err
		}
	}

	for _, page := range td.pages {
		for _, c := range t.Templates() {
			page.AddParseTree(c.Name(), c.Tree)
		}
		f, err := os.Create(filepath.Join(td.OutputDir, page.Name()+".html"))
		if err != nil {
			return err
		}
		defer f.Close()

		if err = page.Execute(f, td); err != nil {
			return err
		}
	}

	return nil
}

func (td *TemplateData) Walk() error {
	if td.InputDir == "" {
		td.InputDir = "."
	}

	return filepath.WalkDir(td.InputDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		name, ext := splitExt(info.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		switch ext {
		case ".tmpl":
			t, err := template.New(name).Funcs(TemplateFuncs).Parse(string(content))
			if err != nil {
				return err
			}
			td.pages = append(td.pages, t)
		case ".wed.html":
			c, err := NewComponent(name, string(content))
			if err != nil {
				return err
			}
			td.components = append(td.components, c)
		}

		return nil
	})
}

func splitExt(name string) (base, ext string) {
	ext = filepath.Ext(name)
	base = name[:len(name)-len(ext)]
	if ext == ".html" {
		b, e := splitExt(base)
		return b, e + ext
	}
	return
}

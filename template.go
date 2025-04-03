package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type TemplateData struct {
	FileSettings
	StylePaths  []string
	ScriptPaths []string
	pages       []*template.Template
	components  []Component
	dynamics    map[string]string
	funcs       template.FuncMap
}

func NewTemplateData(s FileSettings) *TemplateData {
	var td = TemplateData{FileSettings: s, dynamics: make(map[string]string)}

	td.funcs = template.FuncMap{
		"args": func(v ...any) []any {
			return v
		},
		"hold": func(names ...string) (any, error) {
			values := make([]template.HTML, len(names))
			for i, name := range names {
				str := new(strings.Builder)
				if err := td.pages[0].ExecuteTemplate(str, name, td); err != nil {
					return "", fmt.Errorf("cannot find component %q to hold", names[i])
				}
				values[i] = template.HTML(name)
			}
			if len(values) == 1 {
				return values[0], nil
			}
			return values, nil
		},
		"embed": func(link string) (emb template.HTML, err error) {
			content, err := getContent(link)
			if err == nil {
				emb = template.HTML(content)
			}
			return
		},
		"importDynamics": func(names ...string) (template.HTML, error) {
			out, notFound := "", []string{}

			if len(names) == 0 {
				for _, c := range td.dynamics {
					out += c
				}
				return template.HTML(out), nil
			}

			for _, name := range names {
				c, ok := td.dynamics[name]
				if !ok {
					notFound = append(notFound, name)
				} else {
					out += c
				}
			}

			if len(notFound) > 0 {
				return template.HTML(out), fmt.Errorf("cannot dynamically import %v", notFound)
			}
			return template.HTML(out), nil
		},
	}

	return &td
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

	switch c.Type {
	case static:
		_, err = t.New(c.Name).Parse(c.WrappedStaticHTML())
	case dynamic:
		td.dynamics[c.Name] = c.WrappedDynamicHTML()
	case hybrid:
		td.dynamics[c.Name] = c.WrappedDynamicHTML()
		_, err = t.New(c.Name).Parse(c.WrappedStaticHTML())
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

func (td *TemplateData) Build() error {
	t := template.New("temp")
	for _, c := range td.components {
		if err := td.WriteComponent(t, c); err != nil {
			return err
		}
	}

	for name, c := range td.dynamics {
		if _, err := t.New(name).Parse(c); err != nil {
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

		switch name, ext := splitExt(info.Name()); ext {
		case ".tmpl":
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			t, err := template.New(name).Funcs(td.funcs).Parse(string(content))
			if err != nil {
				return err
			}
			td.pages = append(td.pages, t)
		case ".wed.html":
			f, err := os.Open(path)
			if err == nil {
				defer f.Close()
				var c Component
				if c, err = NewComponentReader(name, f); err == nil {
					td.components = append(td.components, c)
				}
			}
			return err
		}

		return nil
	})
}

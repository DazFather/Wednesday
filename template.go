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
		"list": func(v ...any) []any { return v },
		"embed": func(link string) (emb template.HTML, err error) {
			content, err := getContent(link)
			if err == nil {
				emb = template.HTML(content)
			}
			return
		},
		"use":     td.use,
		"props":   td.props,
		"hold":    td.hold,
		"drop":    td.drop,
		"var":     td.getVar,
		"dynamic": td.dynamic,
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

func (td *TemplateData) Build() error {
	t := td.newTempl("temp")
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
			t, err := td.newTempl(name).Parse(string(content))
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

func (td *TemplateData) newTempl(name string) *template.Template {
	return template.New(name).Funcs(td.funcs).Delims("<!--{", "}-->")
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

func (td *TemplateData) getVar(name string, def ...any) (any, error) {
	value, found := td.Var[name]
	if found {
		return value, nil
	}

	switch len(def) {
	case 0:
		return nil, fmt.Errorf("Value %q requested but not provided", name)
	case 1:
		return def[0], nil
	}
	return def, nil
}

func (td *TemplateData) dynamic(names ...string) (template.HTML, error) {
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
}

type ComponentInfo struct {
	holds map[string]template.HTML
	Props map[string]any
}

func (c *ComponentInfo) merge(another ComponentInfo) {
	if len(another.holds) > 0 {
		if c.holds == nil {
			c.holds = make(map[string]template.HTML)
		}
		for key, val := range another.holds {
			c.holds[key] = val
		}
	}
	if len(another.Props) > 0 {
		if c.Props == nil {
			c.Props = make(map[string]any)
		}
		for key, val := range another.Props {
			c.Props[key] = val
		}
	}
}

func (td *TemplateData) use(name string, opt ...ComponentInfo) (template.HTML, error) {
	var (
		data ComponentInfo
		str  = new(strings.Builder)
	)
	if nopts := len(opt); nopts == 1 {
		data = opt[0]
	} else if nopts != 0 {
		for _, info := range opt {
			data.merge(info)
		}
	}

	if err := td.pages[0].ExecuteTemplate(str, name, data); err != nil {
		return "", err
	}
	return template.HTML(str.String()), nil
}

func (td *TemplateData) props(props ...any) ComponentInfo {
	var (
		lastKey string
		skipped int
		res     = ComponentInfo{Props: make(map[string]any)}
	)
	for i, val := range props {
		if (i-skipped)%2 == 0 {
			switch v := val.(type) {
			case ComponentInfo:
				res.merge(v)
				skipped++
			case string:
				lastKey = v
			default:
				lastKey = fmt.Sprint(v)
			}
		} else {
			res.Props[lastKey] = val
		}
	}

	return res
}

func (td *TemplateData) hold(templs ...any) (res ComponentInfo, err error) {
	var (
		last     string
		lastOpts []ComponentInfo
	)
	res.holds = make(map[string]template.HTML)

	for _, val := range templs {
		switch v := val.(type) {
		case ComponentInfo:
			lastOpts = append(lastOpts, v)
		case string:
			if last != "" {
				if res.holds[last], err = td.use(last, lastOpts...); err != nil {
					return
				}
			}
			last = v
		default:
			err = fmt.Errorf("Invalid given args type %T expected 'string' or 'ComponentInfo'", v)
			return
		}
	}
	if res.holds[last], err = td.use(last, lastOpts...); err != nil {
		return
	}
	return
}

func (td *TemplateData) drop(info ComponentInfo, names ...string) template.HTML {
	var str template.HTML = ""
	if len(names) == 0 {
		for _, val := range info.holds {
			str += val
		}
		return str
	}

	for name, val := range info.holds {
		for _, n := range names {
			if n == name {
				str += val
				break
			}
		}
	}
	return str
}

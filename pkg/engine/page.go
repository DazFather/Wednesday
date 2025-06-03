package engine

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	util "github.com/DazFather/Wednesday/pkg/shared"
)

type ComponentDependency = util.Dependency[string, Component]

type page struct {
	components *[]Component
	collected  **template.Template
	*Settings
	*template.Template
	deps []ComponentDependency
}

func (td *TemplateData) newPage(name string) *page {
	var p = page{
		components: &td.components,
		Settings:   &td.Settings,
		collected:  &td.collected,
	}
	p.Template = template.New(name).Funcs(template.FuncMap{
		"list": func(v ...any) []any { return v },
		"embed": func(link string) (emb template.HTML, err error) {
			content, err := util.FetchContent(link)
			if err == nil {
				emb = template.HTML(content)
			}
			return
		},
		"use":   p.use,
		"props": p.props,
		"hold":  p.hold,
		"drop":  p.drop,
		"var":   p.getVar,
	})

	td.pages = append(td.pages, &p)

	return &p
}

func (p *page) Execute(w io.Writer, data any) error {
	var (
		raw                       strings.Builder
		scripts, styles, dynamics []string
	)

	// Run first template engine
	for _, c := range (*p.collected).Templates() {
		p.AddParseTree(c.Name(), c.Tree)
	}
	if err := p.Template.Execute(&raw, data); err != nil {
		return err
	}

	// Dependency check and collect imports
	if names := util.HasCircularDep(p.deps); len(names) > 0 {
		return fmt.Errorf("detected circular dependency in: %s", strings.Join(names, ", "))
	}
	for dep := range util.Inverse(p.deps) {
		c := dep.Data
		if c.Script != "" {
			scripts = append(scripts, p.ScriptURL(c.Name))
		}
		if c.Style != "" {
			styles = append(styles, p.StyleURL(c.Name))
		}
		if c.Type != static {
			dynamics = append(dynamics, c.Name)
		}
		break
	}

	// Run second template engine to apply imports
	templ, err := template.New(p.Template.Name()).Delims("{!import{", "}!}").Funcs(template.FuncMap{
		"dynamics": func() template.HTML {
			s := new(strings.Builder)
			for _, name := range dynamics {
				p.ExecuteTemplate(s, "wed-dynamic-"+name, p)
			}
			return template.HTML(s.String())
		},
		"styles": func() template.HTML {
			s := `<link rel="stylesheet" href="style/wed-style.css" />`
			for _, style := range styles {
				s += `<link rel="stylesheet" href="` + style + `"/>`
			}
			return template.HTML(s)
		},
		"scripts": func() template.HTML {
			s := `<script type="module" src="script/wed-utils.js"></script>`
			for _, script := range scripts {
				s += `<script defer type="module" src="` + script + `"></script>`
			}
			return template.HTML(s)
		},
	}).Parse(raw.String())

	if err == nil {
		err = templ.Execute(w, p)
	}
	return err
}

func (p *page) getVar(name string, def ...any) (any, error) {
	value, found := p.Var[name]
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

func (p *page) use(name string, opt ...ComponentInfo) (template.HTML, error) {
	var (
		data ComponentInfo
		str  strings.Builder
	)
	if nopts := len(opt); nopts == 1 {
		data = opt[0]
	} else if nopts != 0 {
		for _, info := range opt {
			data.merge(info)
		}
	}

	if err := p.ExecuteTemplate(&str, "wed-static-"+name, data); err != nil {
		return "", err
	}

	for _, c := range *p.components {
		if c.Name == name {
			p.deps = append(p.deps, p.toDepencency(c))
		}
	}

	return template.HTML(str.String()), nil

}

func (p page) toDepencency(comp Component) (dep ComponentDependency) {
	dep.Data = comp
	//dep.Imports = []ComponentDependency{{Data: comp, Imports: nil}}
	dep.Imports = make([]ComponentDependency, len(comp.Imports))

	for i, name := range comp.Imports {
		for _, c := range *p.components {
			if c.Name == name {
				dep.Imports[i] = p.toDepencency(c)
			}
		}
	}

	return
}

func (p *page) props(props ...any) ComponentInfo {
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

func (p *page) hold(templs ...any) (res ComponentInfo, err error) {
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
				if res.holds[last], err = p.use(last, lastOpts...); err != nil {
					return
				}
			}
			last = v
		default:
			err = fmt.Errorf("Invalid given args type %T expected 'string' or 'ComponentInfo'", v)
			return
		}
	}
	if res.holds[last], err = p.use(last, lastOpts...); err != nil {
		return
	}
	return
}

func (p *page) drop(info ComponentInfo, names ...string) template.HTML {
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

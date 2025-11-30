package engine

import (
	"bytes"
	"fmt"
	"html/template"
	"slices"
	"strings"

	util "github.com/DazFather/Wednesday/pkg/shared"
)

type ComponentDependency = util.Dependency[string, Component]

type page struct {
	components *[]Component
	collected  **template.Template
	*Settings
	*template.Template
	deps     []ComponentDependency
	Location string
}

func (td *TemplateData) newPage(name string) *page {
	var p = page{
		components: &td.components,
		Settings:   &td.Settings,
		collected:  &td.collected,
		Location:   name + ".html",
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

func (p *page) Build(data any) ([]byte, error) {
	var buf bytes.Buffer

	// Run first template engine
	for _, c := range (*p.collected).Templates() {
		p.AddParseTree(c.Name(), c.Tree)
	}
	if err := p.Template.Execute(&buf, data); err != nil {
		return nil, err
	}

	// Run second template engine to apply imports
	templ, err := p.importTemplate()
	if err == nil {
		if _, err = templ.Parse(buf.String()); err == nil {
			buf.Reset()
			err = templ.Execute(&buf, p)
		}
	}
	return buf.Bytes(), err
}

func (p *page) genImportDynamic(dynamics []string) func() template.HTML {
	if len(dynamics) == 0 {
		return func() template.HTML { return "" }
	}

	return func() template.HTML {
		s := new(strings.Builder)
		for _, name := range util.Compact(dynamics) {
			p.ExecuteTemplate(s, "wed-dynamic-"+name, p)
		}
		return template.HTML(s.String())
	}
}

func (p *page) genImportStyle(styles []string) (func() template.HTML, error) {
	var tags string = p.StyleTag("wed-style")

	styles, err := p.minifyCSS(p.Name(), util.Compact(styles))
	if err != nil {
		return nil, err
	}

	for _, name := range styles {
		tags += p.StyleTag(name)
	}

	return func() template.HTML { return template.HTML(tags) }, nil
}

func (p *page) genImportScript(components []*Component) (func() template.HTML, error) {
	var (
		scripts, preScripts []string
		modules, preModules []string
		tags                = `<script type="text/javascript" src="` + p.ScriptURL("wed-utils") + `"></script>`
	)

	for _, c := range components {
		modType := p.Module
		if c.Module != nil {
			modType = *c.Module
		}

		spath := p.ScriptPath(c.Name)
		switch modType {
		case "", noModule:
			if scripts = append(scripts, spath); c.Preload {
				preScripts = append(preScripts, spath)
			}
		case "ecma", ecmaModule:
			if c.Entry {
				if modules = append(modules, spath); c.Preload {
					preModules = append(preModules, spath)
				}
			}
		}
	}

	if len(scripts) > 0 {
		var def func(n string) bool
		switch preScripts = util.Compact(preScripts); len(preScripts) {
		case 0:
			def = func(n string) bool { return true }
		case 1:
			def = func(n string) bool { return n != preScripts[0] }
		default:
			def = func(n string) bool { return !slices.Contains(preScripts, n) }
		}

		tag, err := p.minifyJS(p.Name(), noModule, util.Compact(scripts), def)
		if err != nil {
			return nil, err
		}
		tags += tag
	}

	if len(modules) > 0 {
		var def func(n string) bool
		switch preModules = util.Compact(preModules); len(preModules) {
		case 0:
			def = func(n string) bool { return true }
		case 1:
			def = func(n string) bool { return n != preModules[0] }
		default:
			def = func(n string) bool { return !slices.Contains(preModules, n) }
		}

		tag, err := p.minifyJS(p.Name(), ecmaModule, util.Compact(modules), def)
		if err != nil {
			return nil, err
		}
		tags += `<script type="importmap">{ "imports": {
	"@wed/utils": "` + p.ScriptURL("wed-utils.mjs") + `",
	"@wed/http": "` + p.ScriptURL("wed-http.mjs") + `"
}}</script>` + tag
	}

	return func() template.HTML { return template.HTML(tags) }, nil
}

func (p *page) importTemplate() (*template.Template, error) {
	var (
		scripts                   []*Component
		styles, dynamics          []string
		importStyle, importScript func() template.HTML
	)

	// Dependency check and collect imports
	if names := util.HasCircularDep(p.deps); len(names) > 0 {
		return nil, fmt.Errorf("detected circular dependency in: %s", strings.Join(names, ", "))
	}

	for _, dep := range util.Inverse(p.deps) {
		c := dep.Data
		if c.Script != "" {
			scripts = append(scripts, &c)
		}
		if c.Style != "" {
			styles = append(styles, p.StylePath(c.Name))
		}
		if c.Type != static {
			dynamics = append(dynamics, c.Name)
		}
	}

	var errch = make(chan error, 2)

	go func() {
		var err error
		importStyle, err = p.genImportStyle(styles)
		errch <- err
	}()

	go func() {
		var err error
		importScript, err = p.genImportScript(scripts)
		errch <- err
	}()

	importDynamic := p.genImportDynamic(dynamics)
	if err := <-errch; err != nil {
		return nil, err
	}
	if err := <-errch; err != nil {
		return nil, err
	}
	close(errch)

	templ := template.New(p.Name()).
		Delims("{!{", "}!}").
		Funcs(template.FuncMap{
			"import": func(val string) (content template.HTML, err error) {
				switch val {StyleURL
				case "dynamics":
					content = importDynamic()
				case "styles":
					content = importStyle()
				case "scripts":
					content = importScript()
				default:
					err = fmt.Errorf("invalid import: %q not supported", val)
				}
				return
			},
			"page": func(val string) error {
				p.Location = val
				return nil
			},
		})

	return templ, nil
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
			dep, err := p.toDepencency(c)
			if err != nil {
				return "", err
			}
			p.deps = append(p.deps, dep)
			break
		}
	}

	return template.HTML(str.String()), nil
}

func (p page) toDepencency(comp Component) (dep ComponentDependency, err error) {
	dep.Data = comp
	//dep.Imports = []ComponentDependency{{Data: comp, Imports: nil}}
	dep.Imports = make([]ComponentDependency, len(comp.Imports))

	for i, name := range comp.Imports {
		if p.Lookup("wed-static-"+name) == nil && p.Lookup("wed-dynamic-"+name) == nil {
			err = fmt.Errorf("on component '%s' trying to require at place %d non existing component '%s'", comp.Name, i+1, name)
			return
		}
		for _, c := range *p.components {
			if c.Name == name {
				if dep.Imports[i], err = p.toDepencency(c); err != nil {
					return
				}
				break
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

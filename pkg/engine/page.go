package engine

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path/filepath"
	"slices"
	"strings"
	"sync"

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
		Location:   filepath.Clean(name + ".html"),
	}
	p.Template = template.New(name).Funcs(template.FuncMap{
		"list": func(v ...any) []any { return v },
		"embed": func(link string) (emb template.HTML, err error) {
			content, err := util.FetchContent(link)
			if err == nil {
				emb = template.HTML(content) // #nosec G203 -- embed intentionally returns build-time content as trusted HTML
			}
			return
		},
		"use":  p.use,
		"args": p.args,
		"hold": p.hold,
		"drop": p.drop,
		"var":  p.getVar,
	})

	td.pages = append(td.pages, &p)

	return &p
}

func (p *page) Build(data any) ([]byte, error) {
	var buf bytes.Buffer

	// Run first template engine
	for _, c := range (*p.collected).Templates() {
		if _, err := p.AddParseTree(c.Name(), c.Tree); err != nil {
			return nil, err
		}
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

func (p *page) genImportDynamic(dynamics []string) (func() template.HTML, error) {
	if len(dynamics) == 0 {
		return func() template.HTML { return "" }, nil
	}

	return func() template.HTML {
		s := new(strings.Builder)
		for _, name := range util.Compact(dynamics) {
			if err := p.ExecuteTemplate(s, "wed-dynamic-"+name, p); err != nil {
				s.WriteString("<!-- template error: " + template.HTMLEscapeString(err.Error()) + " -->")
			}
		}
		return template.HTML(s.String()) // #nosec G203 -- renders trusted, build-generated template output
	}, nil
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

	return func() template.HTML { return template.HTML(tags) }, nil // #nosec G203 -- generated, trusted style import tags
}

func (p *page) genImportScript(components []*Component) (func() template.HTML, error) {
	var (
		scripts, preScripts []string
		modules, preModules []string
		tags                strings.Builder
	)

	tags.WriteString(p.ScriptTag("wed-utils", false, noModule))
	if p.LiveServer != "" {
		tags.WriteString(p.SSEClientTag())
	}

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
		preScripts = util.Compact(preScripts)
		assets, err := p.minifyJS(p.Name(), noModule, util.Compact(scripts))
		if err != nil {
			return nil, err
		}
		for _, asset := range assets {
			tags.WriteString(p.ScriptTag(asset, !slices.Contains(preScripts, asset), noModule))
		}
	}

	if len(modules) > 0 {
		preModules = util.Compact(preModules)
		assets, err := p.minifyJS(p.Name(), ecmaModule, util.Compact(modules))
		if err != nil {
			return nil, err
		}
		tags.WriteString(`<script type="importmap">{ "imports": {
	"@wed/utils": "` + p.ScriptURL("wed-utils.mjs") + `",
	"@wed/http": "` + p.ScriptURL("wed-http.mjs") + `"
}}</script>`)
		for _, asset := range assets {
			tags.WriteString(p.ScriptTag(asset, !slices.Contains(preModules, asset), ecmaModule))
		}
	}

	return func() template.HTML { return template.HTML(tags.String()) }, nil // #nosec G203 -- generated, trusted script import tags
}

func (p *page) importTemplate() (*template.Template, error) {
	var (
		scripts                                  []*Component
		styles, dynamics                         []string
		importStyle, importScript, importDynamic func() template.HTML
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

	var (
		errs [3]error
		wg   sync.WaitGroup
	)

	wg.Go(func() {
		importStyle, errs[0] = p.genImportStyle(styles)
	})
	wg.Go(func() {
		importScript, errs[1] = p.genImportScript(scripts)
	})
	wg.Go(func() {
		importDynamic, errs[2] = p.genImportDynamic(dynamics)
	})

	wg.Wait()
	if err := errors.Join(errs[:]...); err != nil {
		return nil, err
	}

	templ := template.New(p.Name()).
		Delims("{!{", "}!}").
		Funcs(template.FuncMap{
			"import": func(val string) (content template.HTML, err error) {
				switch val {
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
			"location": func(val string) error {
				loc := filepath.Clean(val)
				if filepath.IsAbs(loc) {
					return fmt.Errorf("location must be a relative path: %q", val)
				}
				target := filepath.Join(p.OutputDir, loc)
				if !util.IsOnDir(target, p.OutputDir) {
					return fmt.Errorf("location escapes output directory: %q", val)
				}
				p.Location = loc
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
		return nil, fmt.Errorf("value %q requested but not provided", name)
	case 1:
		return def[0], nil
	}
	return def, nil
}

func (p *page) Props(key string, def any) any {
	return def
}

type ComponentInfo struct {
	hybrid bool
	holds  map[string]template.HTML
	props  map[string]any
}

func (c ComponentInfo) Props(key string, def ...any) (val any, err error) {
	switch len(def) {
	case 0:
		if c.hybrid {
			return nil, fmt.Errorf("cannot access hybrid component props without default value (key: %s)", key)
		}
		found := false
		if val, found = c.props[key]; !found {
			return nil, fmt.Errorf("missing property %q or a default value", key)
		}
	case 1:
		found := false
		if val, found = c.props[key]; !found {
			val = def[0]
		}
	default:
		return nil, fmt.Errorf("too many default values (%d) on .Props %q", len(def), key)
	}
	return val, nil
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
	if len(another.props) > 0 {
		if c.props == nil {
			c.props = make(map[string]any)
		}
		for key, val := range another.props {
			c.props[key] = val
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

	for _, c := range *p.components {
		if c.Name == name {
			dep, err := p.toDepencency(c)
			if err != nil {
				return "", err
			}
			p.deps = append(p.deps, dep)
			data.hybrid = c.Type == hybrid
			break
		}
	}

	if err := p.ExecuteTemplate(&str, "wed-static-"+name, data); err != nil {
		return "", err
	}
	return template.HTML(str.String()), nil // #nosec G203 -- renders trusted component templates
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

func (p *page) args(props ...any) ComponentInfo {
	var (
		lastKey string
		skipped int
		res     = ComponentInfo{props: make(map[string]any)}
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
			res.props[lastKey] = val
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
			err = fmt.Errorf("invalid given args type %T expected 'string' or 'ComponentInfo'", v)
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

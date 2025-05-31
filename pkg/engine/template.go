package engine

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	util "github.com/DazFather/Wednesday/pkg/shared"
)

type TemplateData struct {
	Settings
	StylePaths  []string
	ScriptPaths []string
	pages       []*template.Template
	components  []Component
	dynamics    util.Smap[string, string]
	funcs       template.FuncMap
	collected   *template.Template
}

func NewTemplateData(s Settings) *TemplateData {
	var td = TemplateData{Settings: s}

	td.funcs = template.FuncMap{
		"list": func(v ...any) []any { return v },
		"embed": func(link string) (emb template.HTML, err error) {
			content, err := util.FetchContent(link)
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

	td.collected = td.newTempl("temp")

	return &td
}

func (td *TemplateData) AddComponent(c Component) (err error) {
	td.components = append(td.components, c)
	if c.Style != "" {
		td.appendStyle(c.Name)
	}
	if c.Script != "" {
		td.appendScript(c.Name)
	}

	switch c.Type {
	case dynamic:
		td.dynamics.Store(c.Name, c.WrappedDynamicHTML())
	case hybrid:
		td.dynamics.Store(c.Name, c.WrappedDynamicHTML())
		fallthrough
	case static:
		_, err = td.collected.New(c.Name).Parse(c.WrappedStaticHTML())
	}

	return
}

func (td TemplateData) WriteComponent(c Component) (err error) {
	if c.Style != "" {
		if err = c.WriteStyle(td.StylePath(c.Name + ".css")); err != nil {
			return err
		}
	}

	if c.Script != "" {
		return c.WriteScript(td.ScriptPath(c.Name + ".js"))
	}

	return nil
}

func (td *TemplateData) buildStatics(errch chan<- error) bool {
	var (
		wg      sync.WaitGroup
		success = true
	)

	wg.Add(len(td.components))
	for _, c := range td.components {
		go func() {
			if err := td.WriteComponent(c); err != nil {
				success = false
				errch <- err
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return success
}

func (td *TemplateData) buildDynamics(errch chan<- error) bool {
	var (
		wg      sync.WaitGroup
		success = true
	)

	td.dynamics.Range(func(name, content string) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			templ, err := td.newTempl(name).Parse(content)
			if err != nil {
				errch <- err
				success = false
				return
			}
			for _, c := range td.collected.Templates() {
				templ.AddParseTree(c.Name(), c.Tree)
			}

			sb := new(strings.Builder)
			if err = templ.Execute(sb, td); err == nil {
				td.dynamics.Store(name, sb.String())
			} else {
				success = false
				errch <- err
			}
		}()
		return false
	})
	wg.Wait()

	return success
}

func (td *TemplateData) buildPages(errch chan<- error) {
	var wg sync.WaitGroup

	wg.Add(len(td.pages))
	for _, page := range td.pages {
		go func() {
			defer wg.Done()
			for _, c := range td.collected.Templates() {
				page.AddParseTree(c.Name(), c.Tree)
			}

			f, err := os.Create(filepath.Join(td.OutputDir, page.Name()+".html"))
			if err != nil {
				errch <- err
				return
			}
			defer f.Close()

			if err = page.Execute(f, td); err != nil {
				errch <- err
			}
		}()
	}

	wg.Wait()
}

func (td *TemplateData) Build() chan error {
	var errch = make(chan error)

	if !td.buildStatics(errch) {
		close(errch)
		return errch
	}

	go func() {
		if td.buildDynamics(errch) {
			td.buildPages(errch)
		}
		close(errch)
	}()

	return errch
}

func (td *TemplateData) Walk() (errch chan error) {
	if td.InputDir == "" {
		td.InputDir = "."
	}

	errch = make(chan error)
	go func() {
		err := filepath.WalkDir(td.InputDir, func(path string, info fs.DirEntry, err error) error {
			if info.IsDir() {
				return nil
			}

			switch name, ext := splitExt(info.Name()); ext {
			case ".tmpl":
				content, err := os.ReadFile(path)
				if err != nil {
					errch <- fmt.Errorf("cannot read template %q: %s", path, err)
				}
				if t, err := td.newTempl(name).Parse(string(content)); err == nil {
					td.pages = append(td.pages, t)
				} else {
					errch <- fmt.Errorf("cannot parse template %q: %s", path, err)
				}
			case ".wed.html":
				content, err := os.ReadFile(path)
				if err != nil {
					errch <- fmt.Errorf("cannot read component %q: %s", path, err)
				}
				if c, err := NewComponent(name, content); err == nil {
					td.AddComponent(c)
				} else {
					errch <- fmt.Errorf("cannot parse component %q: %s", path, err)
				}
			}

			return nil
		})
		if err != nil {
			errch <- err
		}
		close(errch)
	}()

	return
}

func (td *TemplateData) newTempl(name string) *template.Template {
	return template.New(name).Funcs(td.funcs).Delims("{{", "}}")
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
		td.dynamics.Range(func(name, content string) bool {
			out += content
			return false
		})
		return template.HTML(out), nil
	}

	for _, name := range names {
		c, ok := td.dynamics.Load(name)
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

	if err := td.collected.ExecuteTemplate(str, name, data); err != nil {
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

func Build(s Settings) chan error {
	var (
		td    = NewTemplateData(s)
		errch = make(chan error)
	)

	go func() {
		defer close(errch)
		for _, fn := range []func() chan error{td.Walk, td.Build} {
			failed := false
			for err := range fn() {
				errch <- err
				if !failed {
					failed = true
				}
			}
			if failed {
				return
			}
		}
	}()

	return errch
}

package engine

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/DazFather/Wednesday/pkg/shared"
)

type TemplateData struct {
	collected *template.Template
	Settings
	pages      []*page
	components []Component
}

func NewTemplateData(s Settings) *TemplateData {
	mock := func() error {
		return fmt.Errorf("CALLING MOCKED CALL")
	}

	return &TemplateData{
		Settings: s,
		collected: template.New("temp").Funcs(template.FuncMap{
			"list":  mock,
			"embed": mock,
			"use":   mock,
			"args":  mock,
			"hold":  mock,
			"drop":  mock,
			"var":   mock,
		}),
	}
}

func (td *TemplateData) AddComponent(c Component) (err error) {
	for _, existing := range td.components {
		if existing.Name == c.Name {
			return fmt.Errorf("duplicate component name %q", c.Name)
		}
	}

	switch c.Type {
	case static:
		_, err = td.collected.New("wed-static-" + c.Name).Parse(c.WrappedStaticHTML())
	case hybrid:
		if _, err = td.collected.New("wed-static-" + c.Name).Parse(c.WrappedStaticHTML()); err != nil {
			return
		}
		fallthrough
	case dynamic:
		_, err = td.collected.New("wed-dynamic-" + c.Name).Parse(c.WrappedDynamicHTML())
	default:
		return fmt.Errorf("%w allowed only 'static' (default), 'dynamic' and 'hybrid'", ErrInvalidTypeAttr)
	}

	if err == nil {
		td.components = append(td.components, c)
	}

	return
}

func (td *TemplateData) WriteComponent(c Component) (err error) {
	if c.Style != "" {
		if err = c.WriteStyle(td.StylePath(c.Name)); err != nil {
			return err
		}
	}

	if c.Script != "" {
		return c.WriteScript(td.ScriptPath(c.Name))
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
		go func(comp Component) {
			if err := td.WriteComponent(comp); err != nil {
				success = false
				errch <- err
			}
			wg.Done()
		}(c)
	}
	wg.Wait()

	return success
}

func (td *TemplateData) buildPages(errch chan<- error) {
	var wg sync.WaitGroup

	wg.Add(len(td.pages))
	for _, page := range td.pages {
		go func() {
			defer wg.Done()

			content, err := page.Build(td)
			if err != nil {
				errch <- err
				return
			}

			if dir := filepath.Dir(page.Location); dir != "" {
				if err = os.MkdirAll(filepath.Join(td.OutputDir, dir), 0755); err != nil {
					errch <- err
					return
				}
			}

			if err = os.WriteFile(filepath.Join(td.OutputDir, page.Location), content, 0644); err != nil {
				errch <- err
				return
			}
		}()
	}

	wg.Wait()
}

func (td *TemplateData) Build() <-chan error {
	var errch = make(chan error)

	go func() {
		defer close(errch)
		if td.buildStatics(errch) {
			td.buildPages(errch)
		}
	}()

	return errch
}

func (td *TemplateData) Walk() <-chan error {
	var (
		splitter  = regexp.MustCompile(`^(.*)(\.tmpl|\.wed\.html)$`)
		validator = regexp.MustCompile(`^[a-zA-Z_]([\w\-\.]*\w)?$`)
		errch     = make(chan error)
	)

	if td.InputDir == "" {
		td.InputDir = "."
	}

	go func() {
		defer close(errch)
		err := filepath.WalkDir(td.InputDir, func(path string, info fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if info.IsDir() {
				if shared.IsOnDir(path, td.OutputDir) {
					return filepath.SkipDir
				}
				return nil
			}

			ext, name := "", ""
			if pices := splitter.FindStringSubmatch(info.Name()); len(pices) == 3 {
				name, ext = pices[1], pices[2]
				if !validator.MatchString(name) {
					errch <- fmt.Errorf("invalid file name %q special chars allowed are only '-', '.' and '_'", path)
					return nil
				}
			} else {
				return nil
			}

			switch ext {
			case ".tmpl":
				if content, err := os.ReadFile(path); err != nil {
					errch <- fmt.Errorf("cannot read page template %q: %w", path, err)
				} else {
					for _, p := range td.pages {
						if p.Name() == name {
							errch <- fmt.Errorf("duplicate page name %q: %q conflicts with an already loaded page", name, path)
							return nil
						}
					}
					if _, err := td.newPage(name).Parse(string(content)); err != nil {
						errch <- fmt.Errorf("cannot parse page template %q: %w", path, err)
					}
				}
			case ".wed.html":
				if content, err := os.ReadFile(path); err != nil {
					errch <- fmt.Errorf("cannot read component %q: %w", path, err)
				} else if c, err := NewComponent(name, content); err != nil {
					errch <- fmt.Errorf("cannot parse component %q: %w", path, err)
				} else if err = td.AddComponent(c); err != nil {
					errch <- fmt.Errorf("cannot create component %q: %w", path, err)
				}
			}

			return nil
		})
		if err != nil {
			errch <- err
		}
	}()

	return errch
}

func Build(s Settings, errch chan error) {
	var td = NewTemplateData(s)

	for _, fn := range []func() <-chan error{td.Walk, td.Build} {
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
}

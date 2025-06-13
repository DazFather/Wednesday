package engine

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
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
			"props": mock,
			"hold":  mock,
			"drop":  mock,
			"var":   mock,
		}),
	}
}

func (td *TemplateData) AddComponent(c Component) (err error) {
	switch c.Type {
	case static:
		_, err = td.collected.New("wed-static-" + c.Name).Parse(c.WrappedStaticHTML())
	case hybrid:
		if _, err = td.collected.New("wed-static-" + c.Name).Parse(c.WrappedStaticHTML()); err != nil {
			return
		}
		fallthrough
	case dynamic:
		content := c.WrappedDynamicHTML()
		_, err = td.collected.New("wed-dynamic-" + c.Name).Parse(content)
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

	go func() {
		defer close(errch)
		if td.buildStatics(errch) {
			td.buildPages(errch)
		}
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

				if _, err := td.newPage(name).Parse(string(content)); err != nil {
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

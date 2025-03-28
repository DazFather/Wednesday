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
	Settings
	StylePaths  []string
	ScriptPaths []string
	t           *template.Template
}

func NewTemplateData(s Settings) (td TemplateData, err error) {
	index, err := os.ReadFile(filepath.Join(s.InputDir, indexTemplateName))
	if err != nil {
		return
	}

	t, err := template.New("index").Parse(string(index))
	if err == nil {
		td = TemplateData{Settings: s, t: t}
	}
	return
}

func (td *TemplateData) AddComponent(c Component) (err error) {
	if err = c.WriteStyle(td.StylePath(c.Name + ".css")); err != nil {
		return
	}
	td.appendStyle(c.Name)

	if err = c.WriteScript(td.ScriptPath(c.Name + ".js")); err != nil {
		return
	}
	td.appendScript(c.Name)

	_, err = td.t.New(c.Name).Parse(c.WrappedHTML())
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

func (td TemplateData) Build(home string) error {
	f, err := os.Create(filepath.Join(td.OutputDir, home))
	if err != nil {
		fmt.Println("here")
		return err
	}
	defer f.Close()

	return td.t.Execute(f, td)
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

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".wed.html") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		compName := strings.TrimSuffix(info.Name(), ".wed.html")
		c, err := NewComponentReader(compName, f)
		if err != nil {
			return err
		}

		return td.AddComponent(c)
	})
}

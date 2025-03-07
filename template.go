package main

import (
	"fmt"
	"html/template"
	"io/fs"
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
	index, err := os.ReadFile(filepath.Join(s.InputDir, "index.tmpl"))
	if err != nil {
		return
	}

	t, err := template.New("index").Parse(string(index))
	if err == nil {
		td = TemplateData{Settings: s, t: t}
	}
	return
}

func (td *TemplateData) AddComponent(c Component) error {
	stylePath := filepath.Join("style", c.Name+".css")
	if err := c.WriteStyle(stylePath); err != nil {
		return err
	}
	td.StylePaths = append(td.StylePaths, stylePath)

	scriptPath := filepath.Join("script", c.Name+".js")
	if err := c.WriteScript(scriptPath); err != nil {
		return err
	}
	td.ScriptPaths = append(td.ScriptPaths, stylePath)

	_, err := td.t.New(c.Name).Parse(c.WrappedHTML())
	return err
}

func (td TemplateData) Build(home string) error {
	f, err := os.OpenFile(filepath.Join(td.OutputDir, home), os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	return td.t.Execute(f, td)
}

func (td TemplateData) Walk() error {
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

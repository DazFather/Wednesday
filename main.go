package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "embed"
)

type TemplateData struct {
	Settings
	StylePaths  []string
	ScriptPaths []string
	t           *template.Template
}

func NewTemplateData(s Settings) (TemplateData, error) {
	t, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return TemplateData{}, err
	}
	return TemplateData{Settings: s, t: t}, nil
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

func (td TemplateData) Build(wr io.Writer) error {
	return td.t.Execute(wr, td)
}

func (td TemplateData) Walk() {
	if td.InputDir == "" {
		td.InputDir = "."
	}

	filepath.WalkDir(td.InputDir, func(path string, info fs.DirEntry, err error) error {
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

func main() {
	var components []Component

	for _, fname := range os.Args[1:] {
		f, err := os.Open(fname)
		if err != nil {
			panic(err)
		}

		name := filepath.Base(fname[:len(fname)-len(filepath.Ext(fname))])

		c, err := NewComponentReader(name, f)
		if err != nil {
			panic(err)
		}

		components = append(components, c)
		f.Close()
	}

	td, err := NewTemplateData(Settings{})
	if err != nil {
		panic(err)
	}

	for _, c := range components {
		if err := td.AddComponent(c); err != nil {
			panic(err)
		}
	}

	f, err := os.OpenFile("index.html", os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}

	if err := td.Build(f); err != nil {
		panic(err)
	}
}

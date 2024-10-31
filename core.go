package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	home = "home"

	homeFileName     = home + ".tmpl"
	appFileName      = "app" + COMPONENT_EXT
	settingsFileName = "wed-settings.json"
	scriptFileName   = "wed-utils.js"
	styleFileName    = "wed-style.css"
	composedFileName = "index.html"

	COMPONENT_EXT   = ".wed.html"
	COMPONENT_CLASS = "wed-component"

	parsingFlags = "(?s)"
)

var (
	html  = regexp.MustCompile(parsingFlags + `<html>(.*)</html>`)
	css   = regexp.MustCompile(parsingFlags + `<style>(.*)</style>`)
	js    = regexp.MustCompile(parsingFlags + `<script>(.*)</script>`)
	endln = regexp.MustCompile("\n")

	//go:embed resources/home.default.tmpl
	homeTemplateContent []byte
	//go:embed resources/app.default.wed.html
	appComponentContent []byte
	//go:embed resources/style.default.css
	defStyleContent []byte
	//go:embed resources/utils.default.js
	defScriptContent []byte
)

type Settings struct {
	HomeTempl string         `json:"HomeTempl,omitempty"`
	HomeDir   string         `json:"HomeDir,omitempty"`
	ScriptDir string         `json:"ScriptDir,omitempty"`
	StyleDir  string         `json:"StyleDir,omitempty"`
	Scripts   []string       `json:"Scripts,omitempty"`
	Styles    []string       `json:"Styles,omitempty"`
	Run       []string       `json:"Run,omitempty"`
	Var       map[string]any `json:"Var,omitempty"`

	home  *template.Template
	funcs template.FuncMap
}

func (s Settings) genDirs(mountDir string) (err error) {
	for _, dir := range slices.Compact([]string{s.ScriptDir, s.StyleDir}) {
		if err = os.MkdirAll(filepath.Join(mountDir, s.HomeDir, dir), 0750); err != nil {
			return
		}
	}
	return
}

func LoadSettings(fileName string) (s Settings, err error) {
	var content []byte

	switch fileName = filepath.Clean(fileName); fileName {
	case ".":
		fileName = settingsFileName
		fallthrough
	default:
		if content, err = os.ReadFile(fileName); err == nil {
			err = json.Unmarshal(content, &s)
		}
		if err != nil {
			err = errors.New(`Cannot load "` + fileName + `" as settings: ` + err.Error())
		}
	}

	s.HomeDir = filepath.Clean(s.HomeDir)
	s.ScriptDir = filepath.Clean(s.ScriptDir)
	s.StyleDir = filepath.Clean(s.StyleDir)
	if s.HomeTempl = filepath.Clean(s.HomeTempl); s.HomeTempl == "." {
		s.HomeTempl = homeFileName
	}

	s.funcs = template.FuncMap{
		"args": func(v ...any) []any { return v },
	}

	return
}

func genFile(force bool, content []byte, path ...string) (fileName string, err error) {
	fileName = filepath.Join(path...)

	switch filepath.Dir(fileName) {
	case ".", "":
		if wd, e := os.Getwd(); e == nil {
			fileName = filepath.Join(wd, fileName)
		}
	}

	if !force {
		if _, err = os.Stat(fileName); err == nil || !errors.Is(err, os.ErrNotExist) {
			return
		}
	}
	err = os.WriteFile(fileName, content, 0666)
	return
}

func Build(fromPath string, settings *Settings) (err error) {
	content := []byte{}
	if content, err = os.ReadFile(filepath.Join(fromPath, settings.HomeTempl)); err == nil {
		settings.home, err = template.New(home).Funcs(settings.funcs).Parse(string(content))
	}
	if err != nil {
		settings.home = template.Must(template.New(home).Funcs(settings.funcs).Parse(string(homeTemplateContent)))
		warn(`Cannot use "`+settings.HomeTempl+`" as home template`, "Autogenerated default will be used. ", err)
	}

	err = filepath.Walk(filepath.Clean(fromPath), func(path string, info fs.FileInfo, e error) error {
		if e == nil && !info.IsDir() && strings.HasSuffix(info.Name(), COMPONENT_EXT) {
			e = settings.AddComponent(path)
		}
		return e
	})
	if err != nil {
		return
	}

	homePage, err := os.Create(filepath.Join(fromPath, settings.HomeDir, composedFileName))
	if err != nil {
		return
	}
	defer homePage.Close()

	err = settings.home.Execute(homePage, settings)
	return
}

func link(dir, name string) string {
	return url.PathEscape(filepath.ToSlash(filepath.Join(dir, name)))
}

func (s *Settings) AddComponent(fileName string) (err error) {
	// TODO: add component js and css only if component is being used
	rawcontent, err := os.ReadFile(fileName)
	if err != nil {
		return
	}

	dir, name := filepath.Split(fileName)
	name = strings.TrimSuffix(name, COMPONENT_EXT)

	switch matches := js.FindSubmatch(rawcontent); len(matches) {
	case 0, 1:
	case 2:
		if content := bytes.TrimSpace(matches[1]); len(content) > 0 {
			if _, err = genFile(true, content, dir, s.HomeDir, s.ScriptDir, name+".js"); err != nil {
				return
			}
			s.Scripts = append(s.Scripts, link(s.ScriptDir, name+".js"))
		}
	default:
		return errors.New("Invalid component multiple <script> declaration")
	}

	switch matches := css.FindSubmatch(rawcontent); len(matches) {
	case 0, 1:
	case 2:
		if content := bytes.TrimSpace(matches[1]); len(content) > 0 {
			buf := bytes.NewBufferString("." + name + "." + COMPONENT_CLASS + " {\n\t")
			buf.Write(endln.ReplaceAll(content, []byte{'\n', '\t'}))
			buf.WriteString("\n}")
			if _, err = genFile(true, buf.Bytes(), dir, s.HomeDir, s.StyleDir, name+".css"); err != nil {
				return
			}
			s.Styles = append(s.Styles, link(s.StyleDir, name+".css"))
		}
	default:
		return errors.New("Invalid component multiple <style> declaration")
	}

	switch matches := html.FindSubmatch(rawcontent); len(matches) {
	case 0, 1:
	case 2:
		if content := bytes.TrimSpace(matches[1]); len(content) > 0 {
			tmpl := strings.Builder{}
			tmpl.WriteString(`<div class="` + name + ` ` + COMPONENT_CLASS + `">`)
			tmpl.Write(endln.ReplaceAll(content, []byte{'\n', ' ', ' '}))
			tmpl.WriteString("</div>")
			_, err = s.home.New(name).Funcs(s.funcs).Parse(tmpl.String())
		}
	default:
		return errors.New("Invalid component multiple <html> declaration")
	}

	return
}

package main

import (
	"bytes"
	"errors"
	"html/template"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const parsingFlags = "(?s)"

var (
	htmlTag   = regexp.MustCompile(parsingFlags + `<html(\s+type="\s*(\w*)\s*")?\s*>\s*(.*)\s*</html>`)
	styleTag  = regexp.MustCompile(parsingFlags + `<style\s*>\s*(.*)\s*</style>`)
	scriptTag = regexp.MustCompile(parsingFlags + `<script(\s+require="\s*(.*)\s*")?\s*>\s*(.*)\s*</script>`)

	spaces = regexp.MustCompile(parsingFlags + `\s+`)
	endln  = regexp.MustCompile("\n")
)

type wedScript struct {
	imports         []string
	content         []byte
	name, ext, path string
}

func (ws *wedScript) WriteFile(dirs ...string) error {
	path, err := genFile(true, ws.content, append(dirs, ws.name+ws.ext)...)
	if err == nil {
		ws.path = path
	}
	return err
}

func (s *Settings) addJS(name string, rawcontent []byte) (err error) {
	switch matches := scriptTag.FindSubmatch(rawcontent); len(matches) {
	case 0, 1: // no script tag
		break
	case 4:
		ws := wedScript{ext: ".js", name: name, content: matches[3]}
		if len(matches[2]) > 0 {
			ws.imports = spaces.Split(string(matches[2]), -1)
		}
		s.wedScripts = append(s.wedScripts, ws)
	case 2, 3: // invalid tag
		err = errors.New("Invalid component broken <script> definition")
	default:
		err = errors.New("Invalid component multiple <script> declaration")
	}

	return
}

func (s *Settings) addCSS(name string, rawcontent []byte) (err error) {
	switch matches := styleTag.FindSubmatch(rawcontent); len(matches) {
	case 0, 1: // no script tag
		break
	case 2:
		buf := bytes.NewBuffer(nil)
		buf.WriteByte('.')
		buf.WriteString(name)
		buf.WriteByte('.')
		buf.WriteString(COMPONENT_CLASS)
		buf.WriteString(" {\n\t")
		buf.Write(endln.ReplaceAll(matches[1], []byte{'\n', '\t'}))
		buf.WriteString("\n}\n")
		s.wedStyles = append(s.wedStyles, wedScript{ext: ".css", name: name, content: buf.Bytes()})
	default:
		err = errors.New("Invalid component multiple <style> declaration")
	}

	return
}

func (s *Settings) addStatic(name string, content []byte) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(`<div class="` + name + ` ` + COMPONENT_CLASS + `">`)
	buf.Write(endln.ReplaceAll(content, []byte{'\n', '\t'}))
	buf.WriteString("</div>")
	_, err := s.home.New(name).Funcs(s.funcs).Parse(buf.String())
	return err
}

func (s *Settings) addDynamic(name string, content []byte) error {
	if _, found := s.dynamics[name]; found {
		return errors.New(`Cannot redeclare "` + name + `" dynamic component`)
	}

	buf := bytes.NewBuffer(nil)
	buf.WriteString(`<template id="` + name + `">`)
	buf.WriteString(`<div class="` + name + ` ` + COMPONENT_CLASS + `">`)
	buf.Write(endln.ReplaceAll(content, []byte{'\n', '\t'}))
	buf.WriteString("</div></template>")
	s.dynamics[name] = buf.String()
	return nil
}

func (s *Settings) addHTML(name string, rawcontent []byte) (err error) {
	switch matches := htmlTag.FindSubmatch(rawcontent); len(matches) {
	case 0, 1: // no script tag
		break
	case 4:
		switch strings.ToLower(string(matches[2])) {
		case "", "static":
			err = s.addStatic(name, matches[3])
		case "dynamic":
			err = s.addDynamic(name, matches[3])
		default:
			err = errors.New(`Invalid component broken <html> definition, undefied "` + string(matches[2]) + `" type`)
		}
	case 2, 3:
		err = errors.New(`Invalid component broken <html> definition, undefied "` + string(matches[2]) + `" type`)
	default:
		err = errors.New("Invalid component multiple <html> declaration")
	}

	return
}

func (s *Settings) genResources() (err error) {
	var errch = make(chan error)

	go func() {
		var wg sync.WaitGroup
		wg.Add(2)

		// Generating scripts
		go func() {
			defer wg.Done()

			scripts := make([]*wedScript, len(s.wedScripts))
			for i, ws := range s.wedScripts {
				if err := ws.WriteFile(s.HomeDir, s.ScriptDir); err != nil {
					errch <- err
					return
				}
				scripts[i] = &ws
			}

			sort.Slice(scripts, func(i, j int) bool {
				if len(scripts[i].imports) == 0 {
					return true
				}

				for _, requirements := range scripts[j].imports {
					if scripts[i].name == requirements {
						return true
					}
				}

				return false
			})

			for _, ws := range scripts {
				s.Scripts = append(s.Scripts, s.link(ws.path))
			}
		}()

		// Generating styles
		go func() {
			defer wg.Done()

			for _, ws := range s.wedStyles {
				if err := ws.WriteFile(s.HomeDir, s.StyleDir); err != nil {
					errch <- err
					return
				}
				s.Styles = append(s.Styles, s.link(ws.path))
			}
		}()

		wg.Wait()
		close(errch)
	}()

	for err = range errch {
		return
	}

	return
}

func (s *Settings) importDynamic(names ...string) (content template.HTML, err error) {
	var t *template.Template
	content = "ERROR"
	if t, err = s.home.Clone(); err != nil {
		return
	}

	if len(names) == 0 {
		names = make([]string, len(s.dynamics))
		i := 0
		for name := range s.dynamics {
			names[i], i = name, i+1
		}
	}

	sb := new(strings.Builder)
	for _, name := range names {
		if t, err = t.New(name).Funcs(s.funcs).Parse(s.dynamics[name]); err == nil {
			if err = t.Execute(sb, s); err == nil {
				err = sb.WriteByte('\n')
			}
		}
		if err != nil {
			return
		}
	}

	content = template.HTML(sb.String())
	return
}

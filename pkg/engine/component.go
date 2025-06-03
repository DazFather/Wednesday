package engine

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

// Component struct to store extracted content
type Component struct {
	Name    string
	HTML    string
	Style   string
	Script  string
	Imports []string
	Type    ComponentType
}

func (c Component) Identifier() string {
	return c.Name
}

type ComponentType uint8

const (
	static ComponentType = iota
	dynamic
	hybrid
)

var (
	ErrNoHTMLData       = errors.New("no 'html' data found")
	ErrDuplicateTagData = errors.New("duplicate tag found")
	ErrInvalidTypeAttr  = errors.New("invalid 'type' attribute")
)

var (
	htmlRgx   = regexp.MustCompile(`(?s)<html(\s+type="\s*(\w+)\s*")?\s*>\s*(.*?)\s*</html>`)
	styleRgx  = regexp.MustCompile(`(?s)<style\s*>\s*(.*?)\s*</style>`)
	scriptRgx = regexp.MustCompile(`(?s)<script(\s+require="\s*(.+?)\s*")?\s*>\s*(.*?)\s*</script>`)
)

func NewComponent(name string, content []byte) (c Component, err error) {
	c.Name = name
	if c.HTML, c.Type, err = parseHTML(content); err != nil {
		return
	}

	if c.Style, err = parseStyle(content); err != nil {
		return
	}

	c.Script, c.Imports, err = parseScript(content)
	return
}

func (c Component) WrappedStyle() string {
	if c.Style == "" {
		return ""
	}
	return fmt.Sprintf(`.%v-component.wed-component{%v}`, c.Name, c.Style)
}

func (c Component) WrappedStaticHTML() string {
	return fmt.Sprintf(`<div class="%v-component wed-component">%v</div>`, c.Name, c.HTML)
}

func (c Component) WrappedDynamicHTML() string {
	return fmt.Sprintf(`<template id="%v-component"><div class="%v-component wed-component">%v</div></template>`, c.Name, c.Name, c.HTML)
}

func (c Component) WriteStyle(fpath string) error {
	return os.WriteFile(fpath, []byte(c.WrappedStyle()), os.ModePerm)
}

func (c Component) WriteScript(fpath string) error {
	return os.WriteFile(fpath, []byte(c.Script), os.ModePerm)
}

func parseHTML(content []byte) (inner string, cType ComponentType, err error) {
	switch matches := htmlRgx.FindAllSubmatch(content, -1); len(matches) {
	case 0:
		err = ErrNoHTMLData
	case 1:
		if len(matches[0]) != 4 {
			panic("INVALID HTML REGEXP")
		}
		switch string(matches[0][2]) {
		case "", "static":
			cType = static
		case "dynamic":
			cType = dynamic
		case "hybrid":
			cType = hybrid
		default:
			err = ErrInvalidTypeAttr
		}
		inner = string(matches[0][3])
	default:
		err = ErrDuplicateTagData
	}
	return
}

func parseScript(content []byte) (inner string, imports []string, err error) {
	switch matches := scriptRgx.FindAllSubmatch(content, -1); len(matches) {
	case 0:
		// skip
	case 1:
		if len(matches[0]) != 4 {
			panic("INVALID JS REGEXP")
		}
		if raw := string(matches[0][2]); raw != "" {
			imports = regexp.MustCompile(`\s+`).Split(raw, -1)
		}
		inner = string(matches[0][3])
	default:
		err = ErrDuplicateTagData
	}
	return
}

func parseStyle(content []byte) (inner string, err error) {
	switch matches := styleRgx.FindAllSubmatch(content, -1); len(matches) {
	case 0:
		// skip
	case 1:
		if len(matches[0]) != 2 {
			panic("INVALID CSS REGEXP")
		}
		inner = string(matches[0][1])
	default:
		err = ErrDuplicateTagData
	}
	return
}

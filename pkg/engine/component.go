package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/DazFather/Wednesday/pkg/shared"
)

var (
	ErrNoHTMLData       = errors.New("no 'html' data found")
	ErrDuplicateTagData = errors.New("duplicate tag found")
	ErrInvalidTypeAttr  = errors.New("invalid 'type' attribute")

	spaces = regexp.MustCompile(`(?s)\s+`)
)

type ComponentType uint8

const (
	static ComponentType = iota
	dynamic
	hybrid
)

func ParseComponentType(raw string) (ComponentType, error) {
	switch raw = strings.ToLower(strings.Trim(raw, `"'`)); raw {
	case "", "static":
		return static, nil
	case "dynamic":
		return dynamic, nil
	case "hybrid":
		return hybrid, nil
	}

	return 0, fmt.Errorf("%w '%s' allowed only 'static' (default), 'dynamic' and 'hybrid'", ErrInvalidTypeAttr, raw)
}

// Component struct to store extracted content
type Component struct {
	Name    string
	HTML    string
	Style   string
	Script  string
	Imports []string
	Type    ComponentType
	Preload bool
}

func (c Component) String() string {
	js, css := "", ""
	if c.Script != "" {
		js = "js"
	}
	if c.Style != "" {
		css = "css"
	}
	return fmt.Sprint("c<", c.Type, ">", c.Name, c.Imports, "{", js, "|", css, "}")
}

func (c Component) Identifier() string {
	return c.Name
}

func ParseComponent(r io.Reader) (c Component, err error) {
	parsed, err := shared.ParsePlainHtml(r, []string{"script", "style", "html"}, false)
	if err != nil {
		return
	}

	for _, block := range parsed {
		switch block.Tag {
		case "html":
			c.HTML = block.InnerHTML
			if val, found := block.GetAttr("type"); found {
				if c.Type, err = ParseComponentType(val); err != nil {
					return
				}
			}
		case "style":
			c.Style = block.InnerHTML
		case "script":
			c.Script = block.InnerHTML
			for _, attr := range block.Attrs {
				switch attr.Key {
				case "preload":
					c.Preload = attr.Val == "" || strings.ToLower(attr.Val) == "true"
				case "require":
					c.Imports = spaces.Split(attr.Val, -1)
				}
			}
		}
	}

	if c.HTML == "" {
		err = ErrNoHTMLData
	}

	return
}

func NewComponent(name string, content []byte) (Component, error) {
	var c, err = ParseComponent(bytes.NewReader(content))
	c.Name = name

	if err != nil {
		err = fmt.Errorf("malformed '%s' component: %w", name, err)
	}
	return c, err
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

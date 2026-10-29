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

	"golang.org/x/net/html"
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
	Module  *ModuleType
	Name    string
	HTML    string
	Style   string
	Script  string
	Imports []string
	Type    ComponentType
	Preload bool
	Entry   bool
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
	parsed, err := shared.ParsePlainHtml(r)
	if err != nil {
		return
	}

	if block, found := parsed["html"]; found {
		err = c.SetHtml(block.Attrs, block.InnerHTML)
	} else {
		err = ErrNoHTMLData
	}

	if block, found := parsed["style"]; found {
		c.Style = block.InnerHTML
	}

	if block, found := parsed["script"]; found {
		err = errors.Join(err, c.SetScript(block.Attrs, block.InnerHTML))
	}

	return
}

func (c *Component) SetHtml(attrs []html.Attribute, content string) (err error) {
	switch len(attrs) {
	case 1:
		if attrs[0].Key != "type" {
			err = errors.New("Unrecognized html attribute '" + attrs[0].Key + "', the only accepted is 'type'")
		} else {
			c.Type, err = ParseComponentType(attrs[0].Val)
		}
		fallthrough
	case 0:
		c.HTML = content
	default:
		err = errors.New("Too many html attributes, the only accepted is 'type' ")
	}
	return
}

func (c *Component) SetScript(attrs []html.Attribute, content string) error {
	c.Script = content
	for _, attr := range attrs {
		switch attr.Key {
		case "preload":
			if c.Preload = attr.Val == "" || strings.ToLower(attr.Val) == "true"; c.Preload {
				c.Entry = true
			}
		case "entrypoint": // TODO: find a better name
			c.Entry = attr.Val == "" || strings.ToLower(attr.Val) == "true"
		case "require":
			c.Imports = spaces.Split(attr.Val, -1)
		case "module":
			if module, err := ParseModuleType(attr.Val); err == nil {
				c.Module = &module
			} else {
				return err
			}
		default:
			return errors.New("Unrecognized script attribute '" + attr.Key + "'")
		}
	}
	return nil
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

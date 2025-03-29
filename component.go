package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"

	_ "embed"
)

// Component struct to store extracted content
type Component struct {
	Name      string
	HTML      string
	Style     string
	Script    string
	IsDynamic bool
}

var ErrNoWHTMLData = errors.New("no whtml data found")

func NewComponent(name, htmlContent string) (c Component, err error) {
	doc, err := xmlquery.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return c, err
	}

	c.Name = name
	whtmlNode := xmlquery.FindOne(doc, "//html")
	if whtmlNode == nil {
		return c, ErrNoWHTMLData
	}
	c.HTML = strings.TrimSpace(whtmlNode.OutputXML(false))
	c.IsDynamic = whtmlNode.SelectAttr("type") == "dynamic"

	styleNode := xmlquery.FindOne(doc, "//style")
	if styleNode != nil {
		c.Style = strings.TrimSpace(styleNode.InnerText())
	}

	scriptNode := xmlquery.FindOne(doc, "//script")
	if scriptNode != nil {
		c.Script = strings.TrimSpace(scriptNode.InnerText())
	}

	return
}

func NewComponentReader(name string, r io.Reader) (c Component, err error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return
	}
	return NewComponent(name, string(b))
}

func (c Component) WrappedStyle() string {
	if c.Style == "" {
		return ""
	}
	return fmt.Sprintf(`.%v.wed-component{%v}`, c.Name, c.Style)
}

func (c Component) WrappedHTML() string {
	if c.IsDynamic {
		return fmt.Sprintf(`<template id="%v"><div class="%v wed-component">%v</div></template>`, c.Name, c.Name, c.HTML)
	}
	return fmt.Sprintf(`<div class="%v wed-component">%v</div>`, c.Name, c.HTML)
}

func (c Component) WriteStyle(fpath string) error {
	return os.WriteFile(fpath, []byte(c.WrappedStyle()), os.ModePerm)
}

func (c Component) WriteScript(fpath string) error {
	return os.WriteFile(fpath, []byte(c.Script), os.ModePerm)
}

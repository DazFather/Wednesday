package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/antchfx/htmlquery"

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

var (
	//go:embed templates/index.wed.html
	indexTemplate string
	//go:embed templates/app.wed.html
	appTemplate string
)

func NewComponent(name, htmlContent string) (c Component, err error) {
	doc, err := htmlquery.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return c, err
	}

	c.Name = name
	whtmlNode := htmlquery.FindOne(doc, "//whtml/*")
	if whtmlNode == nil {
		return c, ErrNoWHTMLData
	}
	c.HTML = strings.TrimSpace(htmlquery.OutputHTML(whtmlNode, true))
	c.IsDynamic = htmlquery.SelectAttr(whtmlNode, "type") == "dynamic"

	styleNode := htmlquery.FindOne(doc, "//style")
	if styleNode != nil {
		c.Style = strings.TrimSpace(htmlquery.InnerText(styleNode))
	}

	scriptNode := htmlquery.FindOne(doc, "//script")
	if scriptNode != nil {
		c.Script = strings.TrimSpace(htmlquery.InnerText(scriptNode))
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
		return fmt.Sprintf(`<template class="%v wed-component">%v</template>`, c.Name, c.HTML)
	}
	return fmt.Sprintf(`<div class="%v wed-component">%v</div>`, c.Name, c.HTML)
}

func (c Component) WriteStyle(fpath string) error {
	return os.WriteFile(fpath, []byte(c.WrappedStyle()), os.ModePerm)
}

func (c Component) WriteScript(fpath string) error {
	return os.WriteFile(fpath, []byte(c.Style), os.ModePerm)
}

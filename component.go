package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
)

// Component struct to store extracted content
type Component struct {
	Name   string
	HTML   string
	Style  string
	Script string
	Type   ComponentType
}

type ComponentType uint8

const (
	static ComponentType = iota
	dynamic
	hybrid
)

var (
	ErrNoWHTMLData     = errors.New("no 'html' data found")
	ErrInvalidTypeAttr = errors.New("invalid 'type' attribute")
)

func NewComponentReader(name string, content io.Reader) (c Component, err error) {
	doc, err := xmlquery.Parse(content)
	if err != nil {
		return c, err
	}

	c.Name = name
	whtmlNode := xmlquery.FindOne(doc, "//html")
	if whtmlNode == nil {
		return c, ErrNoWHTMLData
	}
	c.HTML = strings.TrimSpace(whtmlNode.OutputXMLWithOptions(
		xmlquery.WithEmptyTagSupport(),
		xmlquery.WithoutPreserveSpace(),
	))
	switch whtmlNode.SelectAttr("type") {
	case "", "static":
		c.Type = static
	case "dynamic":
		c.Type = dynamic
	case "hybrid":
		c.Type = hybrid
	default:
		return c, ErrInvalidTypeAttr
	}
	if err = validHTML(c.HTML); err != nil {
		return
	}

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

func (c Component) WrappedStyle() string {
	if c.Style == "" {
		return ""
	}
	return fmt.Sprintf(`.%v.wed-component{%v}`, c.Name, c.Style)
}

func (c Component) WrappedStaticHTML() string {
	return fmt.Sprintf(`<div class="%v wed-component">%v</div>`, c.Name, c.HTML)
}

func (c Component) WrappedDynamicHTML() string {
	return fmt.Sprintf(`<template id="%v"><div class="%v wed-component">%v</div></template>`, c.Name, c.Name, c.HTML)
}

func (c Component) WriteStyle(fpath string) error {
	return os.WriteFile(fpath, []byte(c.WrappedStyle()), os.ModePerm)
}

func (c Component) WriteScript(fpath string) error {
	return os.WriteFile(fpath, []byte(c.Script), os.ModePerm)
}

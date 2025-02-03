package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
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

func main() {
	c, err := NewComponent("pippo", testHTML)
	if err != nil {
		panic(err)
	}
	fmt.Printf("HTML:\n%s\n\nStyle:\n%s\n\nScript:\n%s\n", c.HTML, c.Style, c.Script)
}

const testHTML = `
<style>
    input[readonly] {
        border: none;
        outline: none;
        background: transparent;
        text-decoration: line-through;
    }
</style>
<whtml>
    <div>
        <input type="checkbox" bind="checked:done:input">
        <input type="text" bind="value:task:input readOnly:done">
        {{ .Cazzi }}
    </div>
</whtml>
<script>
    const { clone: newItem } = useTemplate("todo-item", templ => {
        templ._binds = useBinds(templ)
        return templ
    })
</script>
`

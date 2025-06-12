package shared

import (
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"
)

type HtmlBlock struct {
	Tag       string
	InnerHTML string
	Attrs     []html.Attribute
}

func (b HtmlBlock) GetAttr(name string) (val string, ok bool) {
	for _, attr := range b.Attrs {
		if attr.Key == name {
			return attr.Val, true
		}
	}
	return
}

func ParsePlainHtml(r io.Reader, allowedTagNames []string, allowDuplicate bool) (parsed []HtmlBlock, err error) {
	var (
		tokenizer = html.NewTokenizer(r)
		current   *HtmlBlock
	)

	for ttype := tokenizer.Next(); ttype != html.ErrorToken; ttype = tokenizer.Next() {
		switch ttype {
		case html.StartTagToken:
			token := tokenizer.Token()
			tag := token.Data

			if current != nil {
				err = errors.New("cannot open <" + tag + "> tag with unclosed <" + current.Tag + "> tag")
				return
			}

			for i := range allowedTagNames {
				if tag == allowedTagNames[i] {
					parsed = append(parsed, HtmlBlock{Attrs: token.Attr, Tag: tag})
					current = &parsed[len(parsed)-1]
					if !allowDuplicate {
						allowedTagNames[i] = ""
					}
					break
				}
			}

			if current == nil {
				err = errors.New("unallowed tag <" + tag + "> allowed only: '" + strings.Join(allowedTagNames, "', '") + "'")
				return
			}

			current.InnerHTML = string(tokenizer.Raw())

		case html.EndTagToken:
			tag, _ := tokenizer.TagName()

			if current != nil {
				if strings.ToLower(string(tag)) == current.Tag {
					current = nil
				} else {
					current.InnerHTML += string(tokenizer.Raw())
				}
			} else {
				err = errors.New("cannot close <" + string(tag) + "> tag, missing opening")
				return
			}

		default:
			if current != nil {
				current.InnerHTML += string(tokenizer.Raw())
			}
		}
	}

	// handling tokenization error
	if err = tokenizer.Err(); err == io.EOF {
		err = nil
	} else if err != nil {
		return
	}

	return
}

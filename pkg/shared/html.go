package shared

import (
	"errors"
	"io"
	"regexp"
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

func ParsePlainHtml(r io.Reader) (parsed map[string]*HtmlBlock, err error) {
	var (
		rgx       = regexp.MustCompile(`html|script|style`)
		tokenizer = html.NewTokenizer(r)
		unclosed  *HtmlBlock
	)
	parsed = make(map[string]*HtmlBlock)

	for ttype := tokenizer.Next(); ttype != html.ErrorToken; ttype = tokenizer.Next() {
		switch ttype {
		case html.StartTagToken:
			token := tokenizer.Token()
			tag := token.Data
			if _, found := parsed[tag]; found {
				err = errors.New("Duplicate <" + tag + "> found")
				return
			}

			primary := rgx.MatchString(tag)

			if unclosed == nil {
				if !primary {
					err = errors.New("Wrong, unrecognized or misplaced <" + tag + "> found")
					return
				}
				unclosed = &HtmlBlock{Attrs: token.Attr, Tag: tag}
				parsed[tag] = unclosed
				break
			} else if primary {
				err = errors.New("Cannot open <" + tag + "> without closing </" + unclosed.Tag + "> first")
				return
			}
			unclosed.InnerHTML += string(tokenizer.Raw())

		case html.EndTagToken:
			btag, _ := tokenizer.TagName()
			tag := string(btag)

			if unclosed == nil {
				err = errors.New("Cannot close <" + tag + "> tag, missing opening")
				return
			}
			if strings.ToLower(tag) == unclosed.Tag {
				unclosed = nil
			} else if !rgx.Match(btag) {
				unclosed.InnerHTML += string(tokenizer.Raw())
			} else {
				err = errors.New("Cannot close </" + tag + "> wihout opening <" + unclosed.Tag + "> first")
				return
			}
		default:
			if unclosed != nil {
				unclosed.InnerHTML += string(tokenizer.Raw())
			}
		}
	}

	// handling tokenization error
	if unclosed != nil {
		err = errors.New("Missing closing </" + unclosed.Tag + ">")
	} else if err = tokenizer.Err(); err == io.EOF {
		err = nil
	}
	return
}

package parser

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
)

type ParseResult struct {
	Title string
	Links []string
}

func ParseHTML(body []byte) (ParseResult, error) {
	var result ParseResult

	z := html.NewTokenizer(bytes.NewReader(body))

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return result, nil
			}
			return result, z.Err()

		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()

			if string(name) == "title" {
				result.Title = readTitle(z)
				continue
			}

			if string(name) == "a" && hasAttr {
				for {
					key, val, more := z.TagAttr()
					if string(key) == "href" {
						href := strings.TrimSpace(string(val))
						if href != "" {
							result.Links = append(result.Links, href)
						}
					}
					if !more {
						break
					}
				}
			}
		}
	}
}

func readTitle(z *html.Tokenizer) string {
	var b strings.Builder
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return strings.TrimSpace(b.String())
		case html.TextToken:
			b.Write(z.Text())
		case html.EndTagToken:
			name, _ := z.TagName()
			if string(name) == "title" {
				return strings.TrimSpace(b.String())
			}
		}
	}
}

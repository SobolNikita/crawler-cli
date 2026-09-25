package parser

import (
	"reflect"
	"testing"
)

func TestParseHTML_TitleAndLinks(t *testing.T) {
	html := `
<html>
<head><title> Hello World </title></head>
<body>
  <a href="/about">About</a>
  <a href="https://example.com/x">X</a>
  <a href="">Empty</a>
  <a>No href</a>
</body>
</html>`

	got, err := ParseHTML([]byte(html))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Hello World" {
		t.Fatalf("Title = %q", got.Title)
	}
	want := []string{"/about", "https://example.com/x"}
	if !reflect.DeepEqual(got.Links, want) {
		t.Fatalf("Links = %#v, want %#v", got.Links, want)
	}
}

func TestParseHTML_NoTitle(t *testing.T) {
	got, err := ParseHTML([]byte(`<html><body><a href="/a">A</a></body></html>`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "" {
		t.Fatalf("Title = %q, want empty", got.Title)
	}
	if len(got.Links) != 1 || got.Links[0] != "/a" {
		t.Fatalf("Links = %#v", got.Links)
	}
}

func TestParseHTML_Empty(t *testing.T) {
	got, err := ParseHTML(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "" || len(got.Links) != 0 {
		t.Fatalf("got = %#v", got)
	}
}

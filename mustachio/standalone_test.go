package mustachio

import (
	"strings"
	"testing"
)

func TestStandaloneSectionsAndComments(t *testing.T) {
	tpl := "|\n  {{#sec}}\ncontent\n  {{/sec}}\n{{! comment}}\n|"
	out, err := Render(tpl, map[string]any{"sec": true}, nil)
	if err != nil { t.Fatal(err) }
	expected := "|\ncontent\n|"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestIndentedPartials(t *testing.T) {
	partials := MapPartials{"p": "one\n two\nthree"}
	tpl := "Start\n  {{> p}}\nEnd"
	out, err := Render(tpl, map[string]any{}, partials)
	if err != nil { t.Fatal(err) }
	expected := "Start\n  one\n   two\n  three\nEnd"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestLambdaRenderCallback(t *testing.T) {
	tpl := "{{#upper}}{{name}}{{/upper}}"
	ctx := map[string]any{
		"name": "WilLy",
		"upper": func(text string, render func(string) string) string {
			r := render(text)
			return strings.ToUpper(r)
		},
	}
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	if out != "WILLY" { t.Fatalf("got %q want %q", out, "WILLY") }
}

package mustachio

import "testing"

func TestVariableLambda(t *testing.T) {
	tpl := "* {{title}}"
	ctx := map[string]any{
		"year": 1970,
		"month": 1,
		"day": 1,
		"title": func() string { return "{{year}}-{{month}}-{{day}}" },
	}
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "* 1970-1-1"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestSectionLambdaWrap(t *testing.T) {
	tpl := "{{#wrapped}}{{name}} is awesome.{{/wrapped}}"
	ctx := map[string]any{
		"name": "Willy",
		"wrapped": func(text string) string { return "<b>" + text + "</b>" },
	}
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "<b>Willy is awesome.</b>"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

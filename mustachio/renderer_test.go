package mustachio

import (
	"testing"
)

func TestVariablesAndEscaping(t *testing.T) {
	tpl := "* {{name}}\n* {{{company}}}\n* {{& company}}\n* {{missing}}"
	ctx := map[string]any{
		"name": "Chris",
		"company": "<b>GitHub</b>",
	}
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "* Chris\n* <b>GitHub</b>\n* <b>GitHub</b>\n* "
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestDottedNames(t *testing.T) {
	tpl := "* {{client.name}}\n* {{client.company.name}}\n* {{{company.name}}}"
	ctx := map[string]any{
		"client": map[string]any{ "name": "Chris & Friends" },
		"company": map[string]any{ "name": "<b>GitHub</b>" },
	}
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "* Chris &amp; Friends\n* \n* <b>GitHub</b>"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestSectionsAndInverted(t *testing.T) {
	tpl := "Shown.\n{{#person}}Never shown!{{/person}}\n{{^repo}}No repos :({{/repo}}"
	ctx := map[string]any{ "person": false, "repo": []any{} }
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "Shown.\n\nNo repos :("
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestListSectionAndImplicitIterator(t *testing.T) {
	tpl := "{{#repo}}<b>{{.}}</b>{{/repo}}"
	ctx := map[string]any{ "repo": []any{"resque","hub","rip"} }
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "<b>resque</b><b>hub</b><b>rip</b>"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestPartials(t *testing.T) {
	tpl := "<h2>Names</h2>\n{{#names}}{{> user}}{{/names}}"
	partials := MapPartials{"user": "<strong>{{name}}</strong>"}
	ctx := map[string]any{ "names": []any{ map[string]any{"name":"a"}, map[string]any{"name":"b"} } }
	out, err := Render(tpl, ctx, partials)
	if err != nil { t.Fatal(err) }
	expected := "<h2>Names</h2>\n<strong>a</strong><strong>b</strong>"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

func TestSetDelimiters(t *testing.T) {
	tpl := "* {{default}}\n{{=<% %>=}}* <% changed %>\n<%={{ }}=%>* {{back}}"
	ctx := map[string]any{ "default":"A", "changed":"B", "back":"C" }
	out, err := Render(tpl, ctx, nil)
	if err != nil { t.Fatal(err) }
	expected := "* A\n* B\n* C"
	if out != expected { t.Fatalf("got %q want %q", out, expected) }
}

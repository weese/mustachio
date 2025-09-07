// Package mustachio provides a fast, spec-compliant Mustache template engine for Go.
//
// Mustachio implements the Mustache templating language specification
// (https://mustache.github.io/mustache.5.html) with support for:
//
//   - Variables with HTML escaping: {{name}}
//   - Unescaped variables: {{{name}}} and {{& name}}
//   - Sections and inverted sections: {{#section}}...{{/section}}
//   - Partials: {{> user}}
//   - Set delimiters: {{=<% %>=}}
//   - Lambda functions (optional spec feature)
//   - Numeric indexing in dotted names
//
// Example:
//
//	template := "Hello {{name}}!"
//	data := map[string]any{"name": "World"}
//	result, err := mustachio.Render(template, data, nil)
package mustachio

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"reflect"
	"strings"
)

// ValueProvider provides values for keys (including dotted names) during rendering.
// Implementations may support lambdas by returning a func() any or func(string) any.
// For simple use, a plain map[string]any can be wrapped via MapProvider.

type ValueProvider interface {
	// Lookup returns a value for a given dotted name within the current stack of contexts.
	// If not found, returns (nil, false).
	Lookup(name string) (any, bool)
	// Push returns a new provider with an additional context pushed on top of the stack.
	Push(ctx any) ValueProvider
}

// MapProvider implements ValueProvider on a stack of contexts, each being map[string]any or struct via map-like interface.
// We keep it simple and support map[string]any and struct via reflect tags later if needed.

type MapProvider struct {
	stack []any
}

func NewMapProvider(root any) *MapProvider {
	return &MapProvider{stack: []any{root}}
}

func (p *MapProvider) Push(ctx any) ValueProvider {
	cp := &MapProvider{stack: make([]any, len(p.stack)+1)}
	copy(cp.stack, p.stack)
	cp.stack[len(p.stack)] = ctx
	return cp
}

func (p *MapProvider) Lookup(name string) (any, bool) {
	// Implicit iterator
	if name == "." {
		if len(p.stack) == 0 {
			return nil, false
		}
		return p.stack[len(p.stack)-1], true
	}
	if strings.Contains(name, ".") {
		segments := strings.Split(name, ".")
		first := segments[0]
		rest := segments[1:]
		for i := len(p.stack) - 1; i >= 0; i-- {
			base, ok := lookupInContext(p.stack[i], []string{first})
			if !ok {
				continue
			}
			val, ok := lookupInContext(base, rest)
			if ok {
				return val, true
			}
			// If chain fails from this base, do not fall back to lower stack frames
			return nil, false
		}
		return nil, false
	}
	// simple name
	segments := []string{name}
	for i := len(p.stack) - 1; i >= 0; i-- {
		val, ok := lookupInContext(p.stack[i], segments)
		if ok {
			return val, true
		}
	}
	return nil, false
}

func lookupInContext(ctx any, segments []string) (any, bool) {
	current := ctx
	for _, s := range segments {
		// Try map lookup
		if mm, ok := current.(map[string]any); ok {
			v, exists := mm[s]
			if !exists {
				return nil, false
			}
			current = v
			continue
		}
		// Try slice/array numeric index
		if arr, ok := current.([]any); ok {
			idx := -1
			if len(s) > 0 {
				idx = 0
				for _, ch := range s {
					if ch < '0' || ch > '9' {
						idx = -1
						break
					}
					idx = idx*10 + int(ch-'0')
				}
			}
			if idx < 0 || idx >= len(arr) {
				return nil, false
			}
			current = arr[idx]
			continue
		}
		return nil, false
	}
	return current, true
}

// Node types

type node interface {
	render(w io.Writer, provider ValueProvider, partials PartialLoader, delimiters delimiters) error
}

type textNode struct{ text string }

func (t *textNode) render(w io.Writer, _ ValueProvider, _ PartialLoader, _ delimiters) error {
	_, err := io.WriteString(w, t.text)
	return err
}

type varNode struct {
	name      string
	unescaped bool
}

func (v *varNode) render(w io.Writer, p ValueProvider, _ PartialLoader, _ delimiters) error {
	val, ok := p.Lookup(v.name)
	if !ok || val == nil {
		return nil
	}
	// Variable lambda: if callable zero-arg returns string, render as mustache against current context
	if str, called, err := tryCallZeroArgLambda(val); called {
		if err != nil {
			return err
		}
		ast, err := Parse(str, delimiters{otag: "{{", ctag: "}}"})
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := ast.render(&buf, p, nil, delimiters{otag: "{{", ctag: "}}"}); err != nil {
			return err
		}
		out := buf.String()
		if v.unescaped {
			_, err := io.WriteString(w, out)
			return err
		}
		_, err = io.WriteString(w, escapeHTMLSpec(out))
		return err
	}
	s := toString(val)
	if v.unescaped {
		_, err := io.WriteString(w, s)
		return err
	}
	_, err := io.WriteString(w, escapeHTMLSpec(s))
	return err
}

func escapeHTMLSpec(s string) string {
	// Spec expects &quot; for double quotes; Go's html.EscapeString outputs &#34;
	// We can use html.EscapeString then replace numeric entity with &quot;
	esc := html.EscapeString(s)
	esc = strings.ReplaceAll(esc, "&#34;", "&quot;")
	return esc
}

type sectionNode struct {
	name     string
	inverted bool
	children []node
	raw      string
}

func isFalsey(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	}
	return false
}

func toString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case fmt.Stringer:
		return s.String()
	case []byte:
		return string(s)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", s)
	default:
		return fmt.Sprintf("%v", s)
	}
}

func (s *sectionNode) render(w io.Writer, p ValueProvider, partials PartialLoader, delims delimiters) error {
	val, _ := p.Lookup(s.name)
	if s.inverted {
		if isFalsey(val) {
			return renderChildren(w, p, partials, delims, s.children)
		}
		return nil
	}
	// Section lambda
	if rendered, called, err := tryCallSectionLambda(val, s.raw, p, partials, delims); called {
		if err != nil {
			return err
		}
		_, err := io.WriteString(w, rendered)
		return err
	}
	// normal section
	switch v := val.(type) {
	case nil:
		return nil
	case bool:
		if v {
			return renderChildren(w, p, partials, delims, s.children)
		}
		return nil
	case []any:
		for _, item := range v {
			if err := renderChildren(w, p.Push(item), partials, delims, s.children); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		return renderChildren(w, p.Push(v), partials, delims, s.children)
	default:
		// truthy
		return renderChildren(w, p.Push(v), partials, delims, s.children)
	}
}

type partialNode struct {
	name   string
	indent string
}

func (pn *partialNode) render(w io.Writer, p ValueProvider, partials PartialLoader, delims delimiters) error {
	if partials == nil {
		return nil
	}
	tpl, ok := partials.LoadPartial(pn.name)
	if !ok || tpl == "" {
		return nil
	}
	if pn.indent != "" {
		tpl = applyIndent(tpl, pn.indent)
	}
	ast, err := Parse(tpl, delims)
	if err != nil {
		return err
	}
	return ast.render(w, p, partials, delims)
}

func applyIndent(tpl string, indent string) string {
	if indent == "" || tpl == "" {
		return tpl
	}
	parts := strings.Split(tpl, "\n")
	trailing := strings.HasSuffix(tpl, "\n")
	var b strings.Builder
	for i, part := range parts {
		// skip the synthetic last empty part produced by trailing newline
		if i == len(parts)-1 && trailing && part == "" {
			break
		}
		b.WriteString(indent)
		b.WriteString(part)
		if i < len(parts)-1 || trailing {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type rootNode struct{ children []node }

func (r *rootNode) render(w io.Writer, p ValueProvider, partials PartialLoader, delims delimiters) error {
	return renderChildren(w, p, partials, delims, r.children)
}

func renderChildren(w io.Writer, p ValueProvider, partials PartialLoader, delims delimiters, nodes []node) error {
	for _, n := range nodes {
		if err := n.render(w, p, partials, delims); err != nil {
			return err
		}
	}
	return nil
}

// PartialLoader loads partial templates by name.

type PartialLoader interface {
	LoadPartial(name string) (string, bool)
}

type MapPartials map[string]string

func (m MapPartials) LoadPartial(name string) (string, bool) { v, ok := m[name]; return v, ok }

// delimiters represents the current opening and closing tag delimiters

type delimiters struct {
	otag string
	ctag string
}

// Parse parses a mustache template into an AST using the provided delimiters (or default `{{`, `}}` if zero value).

func Parse(template string, delims delimiters) (*rootNode, error) {
	if delims.otag == "" && delims.ctag == "" {
		delims = delimiters{otag: "{{", ctag: "}}"}
	}
	tokens, err := lex(template, delims)
	if err != nil {
		return nil, err
	}
	return parseTokens(template, tokens)
}

// Render renders a template with the provided data context and partials.

func Render(template string, data any, partials PartialLoader) (string, error) {
	ast, err := Parse(template, delimiters{otag: "{{", ctag: "}}"})
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	prov := NewMapProvider(toAnyMap(data))
	if err := ast.render(&buf, prov, partials, delimiters{otag: "{{", ctag: "}}"}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func toAnyMap(d any) any {
	switch v := d.(type) {
	case map[string]any:
		return v
	default:
		return d
	}
}

// Lexer and parser

type tokenType int

const (
	tText tokenType = iota
	tVar
	tUVar
	tSectionStart
	tInvertedStart
	tSectionEnd
	tPartial
	tComment
	tSetDelims
)

type token struct {
	typ   tokenType
	val   string
	start int
	end   int
}

func lex(input string, delims delimiters) ([]token, error) {
	var tokens []token
	otag := delims.otag
	ctag := delims.ctag
	i := 0
	for i < len(input) {
		idx := strings.Index(input[i:], otag)
		if idx < 0 {
			if i < len(input) {
				tokens = append(tokens, token{typ: tText, val: input[i:], start: i, end: len(input)})
			}
			break
		}
		idx += i
		if idx > i {
			tokens = append(tokens, token{typ: tText, val: input[i:idx], start: i, end: idx})
			i = idx
		}
		// Triple mustache: {{{name}}}
		if otag == "{{" && strings.HasPrefix(input[i:], "{{{") {
			end := strings.Index(input[i+3:], "}}}")
			if end < 0 {
				return nil, fmt.Errorf("unclosed triple mustache")
			}
			end += i + 3
			name := strings.TrimSpace(input[i+3 : end])
			tokens = append(tokens, token{typ: tUVar, val: name, start: i, end: end + 3})
			i = end + 3
			continue
		}
		end := strings.Index(input[i+len(otag):], ctag)
		if end < 0 {
			return nil, fmt.Errorf("unclosed tag")
		}
		end += i + len(otag)
		tagContent := strings.TrimSpace(input[i+len(otag) : end])
		tagEnd := end + len(ctag)
		if tagContent == "" {
			i = tagEnd
			continue
		}
		switch {
		case strings.HasPrefix(tagContent, "!"):
			// comment
			tokens = append(tokens, token{typ: tComment, start: i, end: tagEnd})
		case strings.HasPrefix(tagContent, "=") && strings.HasSuffix(tagContent, "="):
			// set delimiters: =<% %>=
			inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(tagContent, "="), "="))
			parts := strings.Fields(inner)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid set delimiters")
			}
			tokens = append(tokens, token{typ: tSetDelims, val: parts[0] + " " + parts[1], start: i, end: tagEnd})
			// update runtime delimiters for subsequent lexing
			otag = parts[0]
			ctag = parts[1]
		case strings.HasPrefix(tagContent, "#"):
			tokens = append(tokens, token{typ: tSectionStart, val: strings.TrimSpace(tagContent[1:]), start: i, end: tagEnd})
		case strings.HasPrefix(tagContent, "^"):
			tokens = append(tokens, token{typ: tInvertedStart, val: strings.TrimSpace(tagContent[1:]), start: i, end: tagEnd})
		case strings.HasPrefix(tagContent, "/"):
			tokens = append(tokens, token{typ: tSectionEnd, val: strings.TrimSpace(tagContent[1:]), start: i, end: tagEnd})
		case strings.HasPrefix(tagContent, ">"):
			tokens = append(tokens, token{typ: tPartial, val: strings.TrimSpace(tagContent[1:]), start: i, end: tagEnd})
		case strings.HasPrefix(tagContent, "{") && strings.HasSuffix(tagContent, "}"):
			name := strings.TrimSpace(tagContent[1 : len(tagContent)-1])
			tokens = append(tokens, token{typ: tUVar, val: name, start: i, end: tagEnd})
		case strings.HasPrefix(tagContent, "&"):
			name := strings.TrimSpace(tagContent[1:])
			tokens = append(tokens, token{typ: tUVar, val: name, start: i, end: tagEnd})
		default:
			tokens = append(tokens, token{typ: tVar, val: strings.TrimSpace(tagContent), start: i, end: tagEnd})
		}
		i = tagEnd
	}
	return tokens, nil
}

// standalone utilities

func isWhitespaceOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' && s[i] != '\r' {
			return false
		}
	}
	return true
}

func detectStandalone(template string, t token) (standalone bool, indent string, removeTo int) {
	// find start of line
	ls := t.start
	for ls > 0 {
		if template[ls-1] == '\n' {
			break
		}
		ls--
	}
	indent = template[ls:t.start]
	if !isWhitespaceOnly(indent) {
		return false, "", 0
	}
	// find end of line
	le := t.end
	for le < len(template) && template[le] != '\n' {
		le++
	}
	trail := template[t.end:le]
	if !isWhitespaceOnly(trail) {
		return false, "", 0
	}
	// remove through newline if present
	if le < len(template) && template[le] == '\n' {
		removeTo = le + 1
	} else {
		removeTo = le
	}
	return true, indent, removeTo
}

func parseTokens(template string, tokens []token) (*rootNode, error) {
	root := &rootNode{}
	type openSec struct {
		node  *sectionNode
		start int
	}
	stack := []openSec{}
	appendNode := func(n node) {
		if len(stack) == 0 {
			root.children = append(root.children, n)
		} else {
			stack[len(stack)-1].node.children = append(stack[len(stack)-1].node.children, n)
		}
	}
	// helper to truncate last text node to before line start
	truncateIndent := func(indentStart int) {
		var list *[]node
		if len(stack) == 0 {
			list = &root.children
		} else {
			list = &stack[len(stack)-1].node.children
		}
		if len(*list) == 0 {
			return
		}
		if tn, ok := (*list)[len(*list)-1].(*textNode); ok {
			text := tn.text
			if idx := strings.LastIndexByte(text, '\n'); idx >= 0 {
				tn.text = text[:idx+1]
			} else {
				tn.text = ""
			}
		}
	}
	skipUntil := -1
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.typ == tText {
			if skipUntil >= 0 {
				if t.end <= skipUntil {
					continue
				}
				if t.start < skipUntil {
					t.val = t.val[skipUntil-t.start:]
					t.start = skipUntil
				}
			}
			appendNode(&textNode{text: t.val})
			continue
		}
		switch t.typ {
		case tVar:
			appendNode(&varNode{name: t.val, unescaped: false})
		case tUVar:
			appendNode(&varNode{name: t.val, unescaped: true})
		case tPartial, tComment, tSetDelims, tSectionStart, tInvertedStart, tSectionEnd:
			standalone, indent, removeTo := detectStandalone(template, t)
			if standalone {
				truncateIndent(t.start)
				skipUntil = removeTo
			}
			switch t.typ {
			case tPartial:
				pn := &partialNode{name: t.val}
				if standalone {
					pn.indent = indent
				}
				appendNode(pn)
			case tComment:
				// no AST node
			case tSetDelims:
				// ignore; delimiters already applied during lex
			case tSectionStart:
				stack = append(stack, openSec{node: &sectionNode{name: t.val}, start: t.end})
			case tInvertedStart:
				stack = append(stack, openSec{node: &sectionNode{name: t.val, inverted: true}, start: t.end})
			case tSectionEnd:
				if len(stack) == 0 {
					return nil, fmt.Errorf("unmatched section end for %s", t.val)
				}
				sec := stack[len(stack)-1]
				if sec.node.name != t.val {
					return nil, fmt.Errorf("section mismatch: %s vs %s", sec.node.name, t.val)
				}
				sec.node.raw = template[sec.start:t.start]
				stack = stack[:len(stack)-1]
				appendNode(sec.node)
			}
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("unclosed section %s", stack[len(stack)-1].node.name)
	}
	return root, nil
}

// Lambda helpers

func tryCallZeroArgLambda(v any) (string, bool, error) {
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.Func && rv.Type().NumIn() == 0 && rv.Type().NumOut() == 1 && rv.Type().Out(0).Kind() == reflect.String {
		res := rv.Call(nil)
		return res[0].String(), true, nil
	}
	return "", false, nil
}

func tryCallSectionLambda(v any, raw string, p ValueProvider, partials PartialLoader, delims delimiters) (string, bool, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Func {
		return "", false, nil
	}
	// func(string) string
	if rv.Type().NumIn() == 1 && rv.Type().In(0).Kind() == reflect.String && rv.Type().NumOut() == 1 && rv.Type().Out(0).Kind() == reflect.String {
		res := rv.Call([]reflect.Value{reflect.ValueOf(raw)})
		str := res[0].String()
		ast, err := Parse(str, delims)
		if err != nil {
			return "", true, err
		}
		var buf bytes.Buffer
		if err := ast.render(&buf, p, partials, delims); err != nil {
			return "", true, err
		}
		return buf.String(), true, nil
	}
	// func(string, func(string) string) string
	if rv.Type().NumIn() == 2 && rv.Type().In(0).Kind() == reflect.String && rv.Type().In(1).Kind() == reflect.Func && rv.Type().NumOut() == 1 && rv.Type().Out(0).Kind() == reflect.String {
		renderFn := func(s string) string {
			ast, err := Parse(s, delims)
			if err != nil {
				return ""
			}
			var buf bytes.Buffer
			if err := ast.render(&buf, p, partials, delims); err != nil {
				return ""
			}
			return buf.String()
		}
		res := rv.Call([]reflect.Value{reflect.ValueOf(raw), reflect.ValueOf(renderFn)})
		return res[0].String(), true, nil
	}
	return "", false, nil
}

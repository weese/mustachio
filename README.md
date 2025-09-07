<div align="center">
  <h1 align="center">mustachio (Go Mustache renderer)</h1>
  <h3 align="center">A fast, readable Mustache template engine for Go</h3>
</div>

<div align="center">
  <a href="https://github.com/weese/mustachio/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/weese/mustachio/go.yml?branch=main" />
  </a>
  <a href="https://pkg.go.dev/github.com/weese/mustachio">
    <img src="https://img.shields.io/badge/go-reference-blue.svg" />
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" />
  </a>
  <br/>
  <a align="center" href="https://github.com/weese?tab=followers">
    <img src="https://img.shields.io/github/followers/weese?label=Follow%20%40weese&style=social" />
  </a>
  <br/>
</div>

## Features

- Core Mustache
  - Variables with HTML escaping: `{{name}}`
  - Unescaped variables: `{{{name}}}` and `{{& name}}`
  - Dotted names and context precedence: `{{a.b.c}}`
  - Implicit iterator: `{{.}}`
  - Sections and inverted sections: `{{#section}}...{{/section}}`, `{{^section}}...{{/section}}`
  - Lists and nested contexts
  - Partials: `{{> user}}` with indentation handling for standalone usage
  - Set delimiters: `{{=<% %>=}} ... <%={{ }}=%}`
  - Standalone trimming (sections, inverted, partials, comments, set-delims)
  - CR and CRLF line ending handling in standalone detection
- Extensions implemented
  - Lambdas (optional per spec)
    - Variable lambdas: `func() string`
    - Section lambdas: `func(string) string` and `func(string, func(string) string) string` (render callback)
  - Numeric indexing in dotted names (e.g., `track.0.artist.#text`)
- Testing
  - Unit tests for core features and lambdas
  - Spec runner executes JSON fixtures from `spec/specs/*.json`

#### Not (yet) implemented

- Inheritance (Blocks/Parents): `{{$block}}` / `{{<parent}}` (optional module)
- Dynamic names (optional module)

## Install

```bash
go get github.com/weese/mustachio@latest
```

## Usage

Render a simple template with data:

```go
package main

import (
	"fmt"
	"github.com/weese/mustachio"
)

func main() {
	tpl := "Hello {{name}}!"
	out, err := mustachio.Render(tpl, map[string]any{"name": "World"}, nil)
	if err != nil { panic(err) }
	fmt.Println(out) // Hello World!
}
```

Render with partials, sections, and lambdas:

```go
partials := mustachio.MapPartials{
	"user": "<strong>{{name}}</strong>",
}

data := map[string]any{
	"name": "Chris",
	"wrapped": func(text string, render func(string) string) string {
		return "<b>" + render(text) + "</b>"
	},
}

tpl := "{{#wrapped}}Hi {{> user}}{{/wrapped}}"

out, err := mustachio.Render(tpl, data, partials)
// out => <b>Hi <strong>Chris</strong></b>
```

## Running tests

The repo includes unit tests and spec tests. Spec tests test against the official spec fixtures. They require the `spec` submodule to be present:

```bash
git submodule update --init --recursive
go test ./...
```

The spec runner automatically loads all `spec/specs/*.json` files (excluding optional modules and inheritance by default).

## API

- `Render(template string, data any, partials PartialLoader) (string, error)`
  - `data`: typically `map[string]any`, but any Go value is accepted and used as the root context
  - `partials`: implement `PartialLoader` or use `mustachio.MapPartials`

## License

MIT

## References

- Spec reference: [mustache(5) manual](https://mustache.github.io/mustache.5.html)
- Official spec tests: [mustache/spec](https://github.com/mustache/spec)
- Original Ruby implementation: [mustache/mustache](https://github.com/mustache/mustache)

package mustachio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type specFile struct {
	Overview string     `json:"overview"`
	Tests    []specCase `json:"tests"`
}

type specCase struct {
	Name        string            `json:"name"`
	Description string            `json:"desc"`
	Data        any               `json:"data"`
	Template    string            `json:"template"`
	Partials    map[string]string `json:"partials"`
	Expected    string            `json:"expected"`
}

func TestMustacheSpecJSON(t *testing.T) {
	specsDir := "/Users/weese/Development/mustachio/spec/specs"
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		t.Fatalf("read specs dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		if !strings.HasSuffix(name, ".json") { continue }
		// Skip optional (~*) and inheritance module for now
		if strings.HasPrefix(name, "~") { continue }
		if strings.Contains(strings.ToLower(name), "inheritance") { continue }
		path := filepath.Join(specsDir, name)
		content, err := os.ReadFile(path)
		if err != nil { t.Fatalf("read %s: %v", name, err) }
		var file specFile
		if err := json.Unmarshal(content, &file); err != nil {
			t.Fatalf("unmarshal %s: %v", name, err)
		}
		for _, tc := range file.Tests {
			t.Run(name+"/"+tc.Name, func(t *testing.T) {
				partials := MapPartials{}
				for k, v := range tc.Partials { partials[k] = v }
				out, err := Render(tc.Template, tc.Data, partials)
				if err != nil { t.Fatalf("render error: %v", err) }
				if out != tc.Expected {
					t.Fatalf("%s expected %q got %q", tc.Name, tc.Expected, out)
				}
			})
		}
	}
}

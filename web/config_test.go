package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// The page reads demo.textInput; other demos omit it and keep help-map keys.
func TestDemoTextInputJSON(t *testing.T) {
	data, err := json.Marshal(Config{Demos: []Demo{{ID: "term", TextInput: true}, {ID: "menu"}}})
	if err != nil {
		t.Fatal(err)
	}
	if s := string(data); !strings.Contains(s, `"id":"term","title":"","description":"","source":"","textInput":true`) ||
		strings.Count(s, "textInput") != 1 {
		t.Fatalf("config JSON %s", s)
	}
}

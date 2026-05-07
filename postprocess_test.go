package main

import "testing"

func TestProcess(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"lowercase single sentence strips period", "Hello World.", "hello world"},
		{"multi sentence keeps periods", "This is a test. Another sentence.", "this is a test. another sentence."},
		{"preserve acronym lowercases rest", "Use the API to fetch data.", "use the API to fetch data"},
		{"multiple acronyms", "HTTP and JSON are protocols.", "HTTP and JSON are protocols"},
		{"single letter I gets lowercased", "I went to NASA.", "i went to NASA"},
		{"trailing exclamation untouched", "Hello!", "hello!"},
		{"trailing question untouched", "Wait, what?", "wait, what?"},
		{"empty string", "", ""},
		{"single word with period", "Hi.", "hi"},
		{"acronym with possessive", "The API's response was slow.", "the API's response was slow"},
		{"trailing whitespace preserved", "Hello.\n", "hello\n"},
		{"period mid-sentence keeps trailing", "Visit example.com.", "visit example.com."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Process(c.in, nil)
			if got != c.want {
				t.Errorf("Process(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestProcessWithReplacements(t *testing.T) {
	cfg := &Config{
		Replacements: map[string]string{
			"javascript": "JavaScript",
			"vs code":    "VS Code",
			"i":          "I",
		},
	}
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"single word replacement", "I love javascript.", "I love JavaScript"},
		{"phrase replacement", "use vs code daily.", "use VS Code daily"},
		{"replacement of i preserves capitalization", "i went home.", "I went home"},
		{"case insensitive match", "JavaScript is fine.", "JavaScript is fine"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Process(c.in, cfg)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestProcessWithSurrounds(t *testing.T) {
	cfg := &Config{
		Surrounds: []Surround{
			{Start: "parent", End: "unparent", Open: "(", Close: ")"},
			{Start: "quote", End: "end quote", Open: "\"", Close: "\""},
		},
	}
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"parens", "the value parent x plus y unparent works.", "the value (x plus y) works"},
		{"quotes phrase end", "she said quote hello there end quote loudly.", "she said \"hello there\" loudly"},
		{"nested-different surrounds", "wrap parent quote hi end quote unparent.", "wrap (\"hi\")"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Process(c.in, cfg)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestProcessSurroundStrip(t *testing.T) {
	cfg := &Config{
		Surrounds: []Surround{
			{Start: "parent", End: "unparent", Open: "(", Close: ")", Strip: true},
		},
	}
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"strips leading and trailing punct", "the value parent, like a lot, unparent works.", "the value (like a lot) works"},
		{"strips leading period and trailing space", "wrap parent. so cool unparent.", "wrap (so cool)"},
		{"strips repeated punctuation both ends", "x parent , . , hi , . unparent.", "x (hi)"},
		{"no surround punct unchanged", "x parent hi unparent.", "x (hi)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Process(c.in, cfg)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

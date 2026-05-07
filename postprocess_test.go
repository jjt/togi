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
			got := Process(c.in)
			if got != c.want {
				t.Errorf("Process(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

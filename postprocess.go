package main

import (
	"regexp"
	"strings"
	"unicode"
)

var wordRe = regexp.MustCompile(`\p{L}+(?:'\p{L}+)?`)

func Process(text string, cfg *Config) string {
	text = lowercasePreserveAcronyms(text)
	if cfg != nil {
		text = applySurrounds(text, cfg.Surrounds)
		text = applyReplacements(text, cfg.Replacements)
	}
	text = stripTrailingPeriodIfSingleSentence(text)
	return text
}

func lowercasePreserveAcronyms(s string) string {
	return wordRe.ReplaceAllStringFunc(s, func(w string) string {
		if isAcronym(w) {
			return w
		}
		return strings.ToLower(w)
	})
}

func isAcronym(w string) bool {
	core := w
	if i := strings.Index(w, "'"); i >= 0 {
		core = w[:i]
	}
	if len(core) < 2 {
		return false
	}
	for _, r := range core {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func stripTrailingPeriodIfSingleSentence(s string) string {
	trimmed := strings.TrimRight(s, " \t\n\r")
	if !strings.HasSuffix(trimmed, ".") {
		return s
	}
	body := trimmed[:len(trimmed)-1]
	if strings.ContainsAny(body, ".!?") {
		return s
	}
	tail := s[len(trimmed):]
	return body + tail
}

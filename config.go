package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Replacements map[string]string `toml:"replacements"`
	Surrounds    []Surround        `toml:"surrounds"`
}

type Surround struct {
	Start string `toml:"start"`
	End   string `toml:"end"`
	Open  string `toml:"open"`
	Close string `toml:"close"`
	Strip bool   `toml:"strip"`
}

func ConfigPath() string {
	if p := os.Getenv("LOWR_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "handy-lowr", "config.toml")
}

type ConfigWatcher struct {
	mu     sync.Mutex
	path   string
	mtime  time.Time
	cfg    *Config
	err    error
	loaded bool
	logger interface {
		Info(msg string, args ...any)
		Error(msg string, args ...any)
	}
}

func NewConfigWatcher(path string, logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}) *ConfigWatcher {
	return &ConfigWatcher{path: path, logger: logger}
}

func (w *ConfigWatcher) Get() (*Config, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var newMtime time.Time
	info, err := os.Stat(w.path)
	if err == nil {
		newMtime = info.ModTime()
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if w.loaded && newMtime.Equal(w.mtime) {
		return w.cfg, w.err
	}

	cfg, found, loadErr := LoadConfig(w.path)
	w.loaded = true
	w.mtime = newMtime
	w.cfg = cfg
	w.err = loadErr
	if loadErr != nil {
		w.logger.Error("config load failed", "path", w.path, "err", loadErr)
	} else {
		w.logger.Info("config loaded",
			"path", w.path,
			"found", found,
			"replacements", len(cfg.Replacements),
			"surrounds", len(cfg.Surrounds),
		)
		for _, k := range sortedKeys(cfg.Replacements) {
			w.logger.Info("config.replacement", "from", k, "to", cfg.Replacements[k])
		}
		for _, s := range cfg.Surrounds {
			w.logger.Info("config.surround",
				"start", s.Start,
				"end", s.End,
				"open", s.Open,
				"close", s.Close,
				"strip", s.Strip,
			)
		}
	}
	return cfg, loadErr
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func LoadConfig(path string) (*Config, bool, error) {
	if path == "" {
		return &Config{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, false, nil
		}
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, true, nil
}

func phraseBody(phrase string) string {
	parts := strings.Fields(phrase)
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return strings.Join(parts, `\s+`)
}

func applySurrounds(s string, surrounds []Surround) string {
	for _, sr := range surrounds {
		if sr.Start == "" || sr.End == "" {
			continue
		}
		pat := `(?is)\b` + phraseBody(sr.Start) + `\b\s*(.*?)\s*\b` + phraseBody(sr.End) + `\b`
		re := regexp.MustCompile(pat)
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			sub := re.FindStringSubmatch(m)
			inner := ""
			if len(sub) > 1 {
				inner = sub[1]
			}
			if sr.Strip {
				inner = strings.Trim(inner, ", .\t")
			}
			return sr.Open + inner + sr.Close
		})
	}
	return s
}

func applyReplacements(s string, m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if k == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		v := m[k]
		re := regexp.MustCompile(`(?i)\b` + phraseBody(k) + `\b`)
		s = re.ReplaceAllStringFunc(s, func(string) string { return v })
	}
	return s
}

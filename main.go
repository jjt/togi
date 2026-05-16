package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
}

type modelEntry struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type modelsResponse struct {
	Object string       `json:"object"`
	Data   []modelEntry `json:"data"`
}

const modelID = "handy-deterministic"

var (
	logger    *slog.Logger
	configMgr *ConfigWatcher
)

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("read body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logger.Debug("chat.completions request", "remote", r.RemoteAddr, "body", string(body))

	var req chatRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		logger.Error("decode request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var raw string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			raw = req.Messages[i].Content
			break
		}
	}
	logger.Info("input", "text", raw)
	cfg, cerr := configMgr.Get()
	var out string
	if cerr != nil {
		out = "error in config file"
	} else {
		out = Process(raw, cfg)
	}
	logger.Info("output", "text", out, "changed", raw != out, "config_err", cerr != nil)

	model := req.Model
	if model == "" {
		model = modelID
	}
	resp := chatResponse{
		ID:      "handy-pp",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []chatChoice{{
			Index:        0,
			Message:      chatMessage{Role: "assistant", Content: out},
			FinishReason: "stop",
		}},
	}
	respBody, _ := json.Marshal(resp)
	logger.Debug("chat.completions response", "body", string(respBody))

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(respBody)
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	logger.Debug("models request", "remote", r.RemoteAddr)
	resp := modelsResponse{
		Object: "list",
		Data: []modelEntry{{
			ID:      modelID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "lowr",
		}},
	}
	respBody, _ := json.Marshal(resp)
	logger.Debug("models response", "body", string(respBody))
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(respBody)
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q (debug|info|warn|error)", s)
	}
}

func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		h.ServeHTTP(sw, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"remote", r.RemoteAddr,
			"dur_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func main() {
	addr := flag.String("addr", ":8089", "listen address")
	logLevel := flag.String("log-level", "debug", "log level: debug|info|warn|error")
	configFlag := flag.String("config", "", "path to config.toml (overrides TOGI_CONFIG and XDG default)")
	flag.Parse()

	level, err := parseLevel(*logLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	cfgPath := *configFlag
	if cfgPath == "" {
		cfgPath = ConfigPath()
	}
	configMgr = NewConfigWatcher(cfgPath, logger)
	if _, cerr := configMgr.Get(); cerr != nil {
		logger.Error("initial config load (will keep retrying per-request)", "err", cerr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handleChatCompletions)
	mux.HandleFunc("/chat/completions", handleChatCompletions)
	mux.HandleFunc("/v1/models", handleModels)
	mux.HandleFunc("/models", handleModels)

	logger.Info("listening", "addr", *addr, "log_level", level.String())
	if err := http.ListenAndServe(*addr, withLogging(mux)); err != nil {
		logger.Error("server", "err", err)
		os.Exit(1)
	}
}

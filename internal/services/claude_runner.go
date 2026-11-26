package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// DefaultRunnerModel is the Claude Code model alias used with CLAUDE_RUN_URL
// unless CLAUDE_RUN_MODEL says otherwise.
const DefaultRunnerModel = "opus"

// ClaudeRunner writes insights through a Claude Code container that exposes
// POST /run, so no API key is needed here.
type ClaudeRunner struct {
	url    string
	model  string
	effort string
	client *http.Client
}

// NewClaudeRunnerFromEnv returns a runner, or nil when CLAUDE_RUN_URL is unset.
func NewClaudeRunnerFromEnv() *ClaudeRunner {
	url := os.Getenv("CLAUDE_RUN_URL")
	if url == "" {
		return nil
	}
	model := os.Getenv("CLAUDE_RUN_MODEL")
	if model == "" {
		model = DefaultRunnerModel
	}
	// The request context carries the insight timeout.
	return &ClaudeRunner{url: url, model: model, effort: "medium", client: &http.Client{}}
}

type runRequest struct {
	Model        string `json:"model"`
	Effort       string `json:"effort,omitempty"`
	SystemPrompt string `json:"system_prompt"`
	Prompt       string `json:"prompt"`
}

// runResponse is the part of Claude Code's JSON output we use.
type runResponse struct {
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
	Subtype string `json:"subtype"`
}

func (c *ClaudeRunner) Write(ctx context.Context, system, user string) (string, string, error) {
	body, err := json.Marshal(runRequest{Model: c.model, Effort: c.effort, SystemPrompt: system, Prompt: user})
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("claude runner: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out runResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", fmt.Errorf("claude runner: decode response: %w", err)
	}
	if out.IsError {
		return "", "", fmt.Errorf("claude runner: %s: %s", out.Subtype, truncate(out.Result, 300))
	}
	if strings.TrimSpace(out.Result) == "" {
		return "", "", fmt.Errorf("claude runner: empty result")
	}
	return out.Result, "claude-code/" + c.model, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// NewInsightWriterFromEnv picks the insight writer: the Claude Code runner when
// CLAUDE_RUN_URL is set, else the Messages API when ANTHROPIC_API_KEY is set,
// else nil (insights off). It never returns a typed nil inside the interface.
func NewInsightWriterFromEnv() InsightWriter {
	if r := NewClaudeRunnerFromEnv(); r != nil {
		return r
	}
	if c := NewClaudeFromEnv(); c != nil {
		return c
	}
	return nil
}

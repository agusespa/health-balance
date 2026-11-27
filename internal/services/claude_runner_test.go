package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClaudeRunnerWrite(t *testing.T) {
	var got runRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"type":"result","subtype":"success","is_error":false,"result":"**Sleep** first."}`))
	}))
	defer srv.Close()

	t.Setenv("CLAUDE_RUN_URL", srv.URL)
	t.Setenv("CLAUDE_RUN_MODEL", "")
	r := NewClaudeRunnerFromEnv()
	text, model, err := r.Write(context.Background(), "sys", "usr")
	if err != nil {
		t.Fatal(err)
	}
	if text != "**Sleep** first." || model != "claude-code/sonnet" {
		t.Fatalf("got %q, %q", text, model)
	}
	if got.Model != "sonnet" || got.Effort != "medium" || got.SystemPrompt != "sys" || got.Prompt != "usr" {
		t.Fatalf("request = %+v", got)
	}
}

func TestClaudeRunnerErrors(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		want   string
	}{
		"http error": {500, "boom", "HTTP 500"},
		"is_error":   {200, `{"is_error":true,"subtype":"error_during_execution","result":"nope"}`, "error_during_execution"},
		"empty":      {200, `{"is_error":false,"result":"  "}`, "empty result"},
		"bad json":   {200, `not json`, "decode"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			r := &ClaudeRunner{url: srv.URL, model: "opus", client: srv.Client()}
			_, _, err := r.Write(context.Background(), "s", "u")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNewInsightWriterFromEnv(t *testing.T) {
	t.Setenv("CLAUDE_RUN_URL", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	if w := NewInsightWriterFromEnv(); w != nil {
		t.Fatalf("want nil, got %T", w)
	}
	t.Setenv("ANTHROPIC_API_KEY", "k")
	if _, ok := NewInsightWriterFromEnv().(*Claude); !ok {
		t.Fatal("want *Claude with only an API key")
	}
	t.Setenv("CLAUDE_RUN_URL", "http://claude:8080/run")
	if _, ok := NewInsightWriterFromEnv().(*ClaudeRunner); !ok {
		t.Fatal("want *ClaudeRunner when CLAUDE_RUN_URL is set")
	}
}

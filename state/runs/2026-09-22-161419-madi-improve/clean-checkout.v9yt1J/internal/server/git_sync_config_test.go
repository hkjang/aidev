package server

import (
	"context"
	"testing"
)

func TestGitSyncConfigurationAndNetworkBoundary(t *testing.T) {
	base := map[string]any{"url": "https://git.example.test/team/knowledge.git", "branch": "main", "prefix": "madi/workspace"}
	if e := gitSyncValidateConfig(base); e != nil {
		t.Fatal(e)
	}
	for _, test := range []struct {
		key   string
		value any
	}{
		{"url", "file:///tmp/other-repo"}, {"url", "git@host:repo.git"}, {"url", "https://user:password@git.example.test/a.git"}, {"url", "https://git.example.test/a.git?secret=value"}, {"url", "https://git.example.test:70000/a.git"}, {"url", "ssh://git@host/repo.git"}, {"url", "https://git.example.test/a/../b"}, {"prefix", "../../etc"}, {"prefix", ".git/hooks"}, {"prefix", "/absolute"}, {"branch", "-u origin"}, {"branch", "refs/heads/main"}, {"branch", "topic..bad"}, {"branch", "a.lock"}, {"password", true}, {"hooks_path", "anything"},
	} {
		cfg := map[string]any{}
		for k, v := range base {
			cfg[k] = v
		}
		cfg[test.key] = test.value
		if e := gitSyncValidateConfig(cfg); e == nil {
			t.Fatalf("unsafe config accepted %s=%v", test.key, test.value)
		}
	}
	ctx := context.Background()
	if _, _, e := gitSyncResolve(ctx, "https://127.0.0.1/repo", []string{"127.0.0.1"}, false); e == nil {
		t.Fatal("private network was not opted in")
	}
	if _, _, e := gitSyncResolve(ctx, "https://127.0.0.1/repo", []string{}, true); e == nil {
		t.Fatal("host allowlist bypass")
	}
	if _, _, e := gitSyncResolve(ctx, "https://169.254.169.254/repo", []string{"169.254.169.254"}, true); e == nil {
		t.Fatal("metadata IP accepted")
	}
	if _, ips, e := gitSyncResolve(ctx, "https://127.0.0.1/repo", []string{"127.0.0.1"}, true); e != nil || len(ips) == 0 {
		t.Fatal("explicit private fixture denied", e)
	}
}

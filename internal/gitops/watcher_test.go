package gitops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observeall/observeall/internal/config"
)

// ---- ConfigWatcher tests ----

func TestConfigWatcher_DetectsChange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "observeall.yaml")
	collectorPath := filepath.Join(dir, "collector-config.yaml")

	// Write initial config.
	initialContent := "name: test\nenvironment: dev\nlanguage: go\nbackends: []\nsignals: []\nsampling: \"\"\n"
	if err := os.WriteFile(cfgPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := NewConfigWatcher(cfgPath, collectorPath)
	watcher.pollInterval = 50 * time.Millisecond // speed up for tests

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	changed := make(chan struct{}, 1)
	go func() {
		_ = watcher.Watch(ctx, func(_ config.Project) {
			select {
			case changed <- struct{}{}:
			default:
			}
		})
	}()

	// Give the watcher one poll cycle to record the initial hash.
	time.Sleep(100 * time.Millisecond)

	// Modify the file.
	updatedContent := "name: test\nenvironment: prod\nlanguage: go\nbackends: []\nsignals: []\nsampling: \"\"\n"
	if err := os.WriteFile(cfgPath, []byte(updatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
		// Good – callback fired.
	case <-ctx.Done():
		t.Fatal("timeout: ConfigWatcher did not detect file change")
	}
}

func TestConfigWatcher_NoSpuriousFires(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "observeall.yaml")
	collectorPath := filepath.Join(dir, "collector-config.yaml")

	content := "name: test\nenvironment: dev\nlanguage: go\nbackends: []\nsignals: []\nsampling: \"\"\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := NewConfigWatcher(cfgPath, collectorPath)
	watcher.pollInterval = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	calls := 0
	_ = watcher.Watch(ctx, func(_ config.Project) { calls++ })

	if calls != 0 {
		t.Fatalf("expected 0 onChange calls for unchanged file, got %d", calls)
	}
}

// ---- GitOpsPublisher tests ----

// TestGitOpsPublisher_CommitAndPush uses a fake git binary on a temp PATH
// to verify that CommitAndPush calls git add, commit, and push in order.
func TestGitOpsPublisher_CommitAndPush(t *testing.T) {

	// Build a fake git binary that records its invocations.
	scriptDir := t.TempDir()
	logFile := filepath.Join(scriptDir, "git.log")

	fakeGit := "#!/bin/sh\necho \"$@\" >> " + logFile + "\n"
	fakeGitPath := filepath.Join(scriptDir, "git")
	if err := os.WriteFile(fakeGitPath, []byte(fakeGit), 0755); err != nil {
		t.Fatal(err)
	}

	// Prepend scriptDir to PATH so our fake git takes precedence.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", scriptDir+string(os.PathListSeparator)+origPath)

	pub := NewGitOpsPublisher("")
	if err := pub.CommitAndPush("test: auto commit"); err != nil {
		t.Fatalf("CommitAndPush returned unexpected error: %v", err)
	}

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("could not read git log: %v", err)
	}
	log := string(logBytes)

	for _, want := range []string{"add collector-config.yaml", "commit -m test: auto commit", "push"} {
		found := false
		for _, line := range splitLines(log) {
			if line == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected git invocation %q not found in log:\n%s", want, log)
		}
	}
}

// TestGitOpsPublisher_MissingGit verifies a descriptive error when git is absent.
func TestGitOpsPublisher_MissingGit(t *testing.T) {

	// Use an empty temp dir as PATH so git is not found.
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	pub := NewGitOpsPublisher("")
	err := pub.CommitAndPush("irrelevant")
	if err == nil {
		t.Fatal("expected error when git is not in PATH, got nil")
	}
	if !containsStr(err.Error(), "git not found") {
		t.Errorf("expected 'git not found' in error, got: %v", err)
	}
}

// splitLines splits s into non-empty lines.
func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		if line := s[start:]; line != "" {
			out = append(out, line)
		}
	}
	return out
}

func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

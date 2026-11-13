package gitops

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/observeall/observeall/internal/config"
	"gopkg.in/yaml.v3"
)

// ConfigWatcher polls configPath for changes using SHA-256 file hash comparison.
type ConfigWatcher struct {
	configPath          string
	collectorConfigPath string
	pollInterval        time.Duration
}

// NewConfigWatcher creates a new ConfigWatcher.
func NewConfigWatcher(configPath, collectorConfigPath string) *ConfigWatcher {
	return &ConfigWatcher{
		configPath:          configPath,
		collectorConfigPath: collectorConfigPath,
		pollInterval:        2 * time.Second,
	}
}

// Watch polls the config file for changes and calls onChange when a change is detected.
// It blocks until ctx is cancelled, then returns ctx.Err().
func (w *ConfigWatcher) Watch(ctx context.Context, onChange func(proj config.Project)) error {
	lastHash, _ := fileHash(w.configPath)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentHash, err := fileHash(w.configPath)
			if err != nil {
				// File temporarily unavailable – ignore and try again next tick.
				continue
			}
			if currentHash == lastHash {
				continue
			}
			lastHash = currentHash

			proj, err := loadConfig(w.configPath)
			if err != nil {
				// Malformed YAML mid-write – skip this tick.
				continue
			}
			onChange(proj)
		}
	}
}

// fileHash returns the hex-encoded SHA-256 hash of the named file's contents.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// loadConfig reads and parses a project YAML file.
func loadConfig(path string) (config.Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.Project{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var proj config.Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		return config.Project{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return proj, nil
}

// GitOpsPublisher stages, commits, and pushes changes using the system git binary.
type GitOpsPublisher struct {
	// WorkDir is the git repository root. Defaults to the current working directory.
	WorkDir string
}

// NewGitOpsPublisher creates a GitOpsPublisher rooted at workDir.
// Pass an empty string to use the current working directory.
func NewGitOpsPublisher(workDir string) *GitOpsPublisher {
	return &GitOpsPublisher{WorkDir: workDir}
}

// CommitAndPush runs:
//
//	git add collector-config.yaml
//	git commit -m <message>
//	git push
//
// It returns a descriptive error when git is not available, not configured, or
// when any step fails.
func (g *GitOpsPublisher) CommitAndPush(message string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: install git and ensure it is executable")
	}

	steps := []struct {
		args []string
		desc string
	}{
		{[]string{"add", "collector-config.yaml"}, "git add"},
		{[]string{"commit", "-m", message}, "git commit"},
		{[]string{"push"}, "git push"},
	}

	for _, step := range steps {
		if err := g.runGit(step.args...); err != nil {
			return fmt.Errorf("%s failed: %w", step.desc, err)
		}
	}
	return nil
}

// runGit executes git with the supplied arguments inside WorkDir.
func (g *GitOpsPublisher) runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	if g.WorkDir != "" {
		cmd.Dir = g.WorkDir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return fmt.Errorf("%w\n%s", err, detail)
		}
		return err
	}
	return nil
}

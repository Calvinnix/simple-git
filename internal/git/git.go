package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	repoRoot     string
	repoRootOnce sync.Once
)

// getRepoRoot returns the git repository root directory.
// The result is cached for efficiency.
func getRepoRoot() string {
	repoRootOnce.Do(func() {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		output, err := cmd.Output()
		if err != nil {
			repoRoot = ""
			return
		}
		repoRoot = strings.TrimSpace(string(output))
	})
	return repoRoot
}

// ResetRepoRoot clears the cached repository root.
// This is primarily for testing purposes.
func ResetRepoRoot() {
	repoRootOnce = sync.Once{}
	repoRoot = ""
}

// ToDisplayPath converts a repo-root-relative path to a path relative
// to the current working directory (for display purposes).
func ToDisplayPath(repoRelativePath string) string {
	root := getRepoRoot()
	if root == "" {
		return repoRelativePath
	}

	cwd, err := os.Getwd()
	if err != nil {
		return repoRelativePath
	}

	// Convert repo-relative path to absolute
	absPath := filepath.Join(root, repoRelativePath)

	// Get path relative to cwd
	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return repoRelativePath
	}

	return relPath
}

// Run executes a git command and returns the output
func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = getRepoRoot()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

// RunAllowFailure executes a git command and returns output even if the command fails
// (useful for commands like diff --no-index which exit with 1 when there are differences)
func RunAllowFailure(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = getRepoRoot()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// Return stdout even if there's an error (diff returns 1 when there are differences)
	return stdout.String(), err
}

// StageFile stages a file
func StageFile(path string) error {
	_, err := Run("add", "--", path)
	return err
}

// StageAll stages all changes (tracked and untracked)
func StageAll() error {
	_, err := Run("add", "-A")
	return err
}

// UnstageFile unstages a file
func UnstageFile(path string) error {
	_, err := Run("restore", "--staged", "--", path)
	return err
}

// UnstageAll unstages all staged changes
func UnstageAll() error {
	_, err := Run("reset", "HEAD")
	return err
}

// StashAll stashes all changes with an optional message
func StashAll(message string) error {
	if message == "" {
		_, err := Run("stash", "push")
		return err
	}
	_, err := Run("stash", "push", "-m", message)
	return err
}

// StashFiles stashes specific files with an optional message
func StashFiles(paths []string, message string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	args = append(args, "--")
	args = append(args, paths...)
	_, err := Run(args...)
	return err
}

// DiscardFile discards changes to a tracked file
func DiscardFile(path string) error {
	_, err := Run("restore", "--", path)
	return err
}

// DiscardUntracked removes an untracked file
func DiscardUntracked(path string) error {
	_, err := Run("clean", "-f", "--", path)
	return err
}

// StageHunk stages a specific hunk using patch mode
func StageHunk(patch string) error {
	cmd := exec.Command("git", "apply", "--cached")
	cmd.Dir = getRepoRoot()
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("git apply --cached: %w: %s", err, stderr.String())
	}
	return nil
}

// UnstageHunk unstages a specific hunk
func UnstageHunk(patch string) error {
	cmd := exec.Command("git", "apply", "--cached", "--reverse")
	cmd.Dir = getRepoRoot()
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("git apply --cached --reverse: %w: %s", err, stderr.String())
	}
	return nil
}

// DiscardHunk discards a specific hunk from the working tree
func DiscardHunk(patch string) error {
	cmd := exec.Command("git", "apply", "--reverse")
	cmd.Dir = getRepoRoot()
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("git apply --reverse: %w: %s", err, stderr.String())
	}
	return nil
}

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo() bool {
	_, err := Run("rev-parse", "--git-dir")
	return err == nil
}

// GetBranch returns the current branch name
func GetBranch() string {
	output, err := Run("branch", "--show-current")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(output)
}

// BranchStatus contains tracking information for the current branch
type BranchStatus struct {
	Name    string
	Remote  string // e.g., "origin/master"
	Ahead   int
	Behind  int
}

// Push pushes to the remote
func Push() error {
	_, err := Run("push")
	return err
}

// Commit creates a commit with the given message
func Commit(message string) error {
	_, err := Run("commit", "-m", message)
	return err
}

// GetLog returns the raw git log output
func GetLog(limit int) (string, error) {
	return Run("log", fmt.Sprintf("-%d", limit))
}

// GetBranchStatus returns the current branch and its tracking status
func GetBranchStatus() BranchStatus {
	status := BranchStatus{
		Name: GetBranch(),
	}

	// Get the upstream tracking branch
	upstream, err := Run("rev-parse", "--abbrev-ref", "@{upstream}")
	if err != nil {
		return status // No upstream configured
	}
	status.Remote = strings.TrimSpace(upstream)

	// Get ahead/behind counts
	output, err := Run("rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return status
	}

	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &status.Ahead)
		fmt.Sscanf(parts[1], "%d", &status.Behind)
	}

	return status
}

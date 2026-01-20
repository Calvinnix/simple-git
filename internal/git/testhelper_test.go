package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestRepo provides a temporary git repository for testing
type TestRepo struct {
	Dir     string
	T       *testing.T
	origDir string
}

// NewTestRepo creates a new temporary git repository
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	// Get the original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Create a temp directory
	dir, err := os.MkdirTemp("", "go-on-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo := &TestRepo{
		Dir:     dir,
		T:       t,
		origDir: origDir,
	}

	// Change to the temp directory
	if err := os.Chdir(dir); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	// Reset cached repo root for the new test directory
	ResetRepoRoot()

	// Initialize git repo
	repo.Git("init")
	repo.Git("config", "user.email", "test@example.com")
	repo.Git("config", "user.name", "Test User")

	return repo
}

// Cleanup removes the temp directory and restores the original working directory
func (r *TestRepo) Cleanup() {
	r.T.Helper()
	os.Chdir(r.origDir)
	os.RemoveAll(r.Dir)
	// Reset cached repo root when switching back
	ResetRepoRoot()
}

// Git runs a git command in the test repo
func (r *TestRepo) Git(args ...string) string {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.T.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}

// GitAllowFailure runs a git command that may fail
func (r *TestRepo) GitAllowFailure(args ...string) (string, error) {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// WriteFile creates or overwrites a file with the given content
func (r *TestRepo) WriteFile(name, content string) {
	r.T.Helper()
	path := filepath.Join(r.Dir, name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		r.T.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		r.T.Fatalf("failed to write file %s: %v", name, err)
	}
}

// ReadFile reads the content of a file
func (r *TestRepo) ReadFile(name string) string {
	r.T.Helper()
	path := filepath.Join(r.Dir, name)
	content, err := os.ReadFile(path)
	if err != nil {
		r.T.Fatalf("failed to read file %s: %v", name, err)
	}
	return string(content)
}

// DeleteFile removes a file
func (r *TestRepo) DeleteFile(name string) {
	r.T.Helper()
	path := filepath.Join(r.Dir, name)
	if err := os.Remove(path); err != nil {
		r.T.Fatalf("failed to delete file %s: %v", name, err)
	}
}

// FileExists checks if a file exists
func (r *TestRepo) FileExists(name string) bool {
	path := filepath.Join(r.Dir, name)
	_, err := os.Stat(path)
	return err == nil
}

// CommitFile adds and commits a file with the given message
func (r *TestRepo) CommitFile(name, content, message string) {
	r.T.Helper()
	r.WriteFile(name, content)
	r.Git("add", name)
	r.Git("commit", "-m", message)
}

// InitialCommit creates an initial commit in the repository
func (r *TestRepo) InitialCommit() {
	r.T.Helper()
	r.CommitFile("README.md", "# Test Repository\n", "Initial commit")
}

// CreateBranch creates a new branch and optionally switches to it
func (r *TestRepo) CreateBranch(name string, checkout bool) {
	r.T.Helper()
	if checkout {
		r.Git("checkout", "-b", name)
	} else {
		r.Git("branch", name)
	}
}

// SetupRemote creates a bare remote repository and sets it as origin
func (r *TestRepo) SetupRemote() string {
	r.T.Helper()

	// Create a bare repo as remote
	remoteDir, err := os.MkdirTemp("", "go-on-git-remote-*")
	if err != nil {
		r.T.Fatalf("failed to create remote dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if output, err := cmd.CombinedOutput(); err != nil {
		r.T.Fatalf("failed to init bare repo: %v\n%s", err, output)
	}

	r.Git("remote", "add", "origin", remoteDir)
	return remoteDir
}

// PushToRemote pushes the current branch to origin
func (r *TestRepo) PushToRemote() {
	r.T.Helper()
	r.Git("push", "-u", "origin", "HEAD")
}

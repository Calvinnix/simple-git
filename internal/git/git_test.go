package git

import (
	"os"
	"strings"
	"testing"
)

func TestIsGitRepo(t *testing.T) {
	t.Run("valid git repo", func(t *testing.T) {
		repo := NewTestRepo(t)
		defer repo.Cleanup()

		if !IsGitRepo() {
			t.Error("expected IsGitRepo to return true for a valid git repo")
		}
	})

	t.Run("non-git directory", func(t *testing.T) {
		// Create a regular temp directory (not a git repo)
		dir, err := os.MkdirTemp("", "not-a-repo-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(dir)

		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)

		os.Chdir(dir)

		if IsGitRepo() {
			t.Error("expected IsGitRepo to return false for a non-git directory")
		}
	})
}

func TestGetBranch(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()

	branch := GetBranch()
	// Default branch could be "master" or "main" depending on git config
	if branch != "master" && branch != "main" {
		t.Errorf("expected branch to be 'master' or 'main', got %q", branch)
	}
}

func TestRun(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	t.Run("successful command", func(t *testing.T) {
		output, err := Run("status")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("failed command", func(t *testing.T) {
		_, err := Run("invalid-command-that-does-not-exist")
		if err == nil {
			t.Error("expected error for invalid command")
		}
	})
}

func TestRunAllowFailure(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create a file to test diff --no-index (which exits with 1 when there are differences)
	repo.WriteFile("test.txt", "content")

	output, err := RunAllowFailure("diff", "--no-index", "--", "/dev/null", "test.txt")
	// diff --no-index returns exit code 1 when there are differences
	if err == nil {
		t.Log("diff returned no error (no differences or error suppressed)")
	}
	// Output should still be captured even if command "fails"
	if output == "" {
		t.Error("expected output to be captured even with non-zero exit")
	}
}

func TestStageFile(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()
	repo.WriteFile("new-file.txt", "new content")

	err := StageFile("new-file.txt")
	if err != nil {
		t.Fatalf("StageFile failed: %v", err)
	}

	// Verify file is staged
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, "A  new-file.txt") {
		t.Errorf("expected file to be staged as added, got: %s", output)
	}
}

func TestStageAll(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()
	repo.WriteFile("file1.txt", "content1")
	repo.WriteFile("file2.txt", "content2")
	repo.WriteFile("subdir/file3.txt", "content3")

	err := StageAll()
	if err != nil {
		t.Fatalf("StageAll failed: %v", err)
	}

	// Verify all files are staged
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, "A  file1.txt") {
		t.Errorf("expected file1.txt to be staged, got: %s", output)
	}
	if !strings.Contains(output, "A  file2.txt") {
		t.Errorf("expected file2.txt to be staged, got: %s", output)
	}
	if !strings.Contains(output, "A  subdir/file3.txt") {
		t.Errorf("expected subdir/file3.txt to be staged, got: %s", output)
	}
}

func TestUnstageFile(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()
	repo.WriteFile("test.txt", "content")
	repo.Git("add", "test.txt")

	// Verify file is initially staged
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, "A  test.txt") {
		t.Fatalf("file should be staged initially, got: %s", output)
	}

	err := UnstageFile("test.txt")
	if err != nil {
		t.Fatalf("UnstageFile failed: %v", err)
	}

	// Verify file is unstaged
	output = repo.Git("status", "--porcelain")
	if !strings.Contains(output, "?? test.txt") {
		t.Errorf("expected file to be untracked after unstaging, got: %s", output)
	}
}

func TestUnstageAll(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()
	repo.WriteFile("file1.txt", "content1")
	repo.WriteFile("file2.txt", "content2")
	repo.Git("add", "-A")

	// Verify files are staged
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, "A  file1.txt") || !strings.Contains(output, "A  file2.txt") {
		t.Fatalf("files should be staged initially, got: %s", output)
	}

	err := UnstageAll()
	if err != nil {
		t.Fatalf("UnstageAll failed: %v", err)
	}

	// Verify files are unstaged
	output = repo.Git("status", "--porcelain")
	if !strings.Contains(output, "?? file1.txt") {
		t.Errorf("expected file1.txt to be untracked, got: %s", output)
	}
	if !strings.Contains(output, "?? file2.txt") {
		t.Errorf("expected file2.txt to be untracked, got: %s", output)
	}
}

func TestUnstageFileNoCommits(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Don't create any commits - this is a fresh repo
	repo.WriteFile("test.txt", "content")
	repo.Git("add", "test.txt")

	// Verify file is initially staged
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, "A  test.txt") {
		t.Fatalf("file should be staged initially, got: %s", output)
	}

	err := UnstageFile("test.txt")
	if err != nil {
		t.Fatalf("UnstageFile failed in repo with no commits: %v", err)
	}

	// Verify file is unstaged
	output = repo.Git("status", "--porcelain")
	if !strings.Contains(output, "?? test.txt") {
		t.Errorf("expected file to be untracked after unstaging, got: %s", output)
	}
}

func TestUnstageAllNoCommits(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Don't create any commits - this is a fresh repo
	repo.WriteFile("file1.txt", "content1")
	repo.WriteFile("file2.txt", "content2")
	repo.Git("add", "-A")

	// Verify files are staged
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, "A  file1.txt") || !strings.Contains(output, "A  file2.txt") {
		t.Fatalf("files should be staged initially, got: %s", output)
	}

	err := UnstageAll()
	if err != nil {
		t.Fatalf("UnstageAll failed in repo with no commits: %v", err)
	}

	// Verify files are unstaged
	output = repo.Git("status", "--porcelain")
	if !strings.Contains(output, "?? file1.txt") {
		t.Errorf("expected file1.txt to be untracked, got: %s", output)
	}
	if !strings.Contains(output, "?? file2.txt") {
		t.Errorf("expected file2.txt to be untracked, got: %s", output)
	}
}

func TestDiscardFile(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.CommitFile("test.txt", "original content", "initial")

	// Modify the file
	repo.WriteFile("test.txt", "modified content")

	// Verify file is modified
	output := repo.Git("status", "--porcelain")
	if !strings.Contains(output, " M test.txt") {
		t.Fatalf("file should be modified, got: %s", output)
	}

	err := DiscardFile("test.txt")
	if err != nil {
		t.Fatalf("DiscardFile failed: %v", err)
	}

	// Verify content is restored
	content := repo.ReadFile("test.txt")
	if content != "original content" {
		t.Errorf("expected original content, got: %s", content)
	}

	// Verify working tree is clean
	output = repo.Git("status", "--porcelain")
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected clean working tree, got: %s", output)
	}
}

func TestDiscardUntracked(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()
	repo.WriteFile("untracked.txt", "untracked content")

	// Verify file exists
	if !repo.FileExists("untracked.txt") {
		t.Fatal("untracked file should exist")
	}

	err := DiscardUntracked("untracked.txt")
	if err != nil {
		t.Fatalf("DiscardUntracked failed: %v", err)
	}

	// Verify file is removed
	if repo.FileExists("untracked.txt") {
		t.Error("untracked file should have been removed")
	}
}

func TestStashAll(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.CommitFile("test.txt", "original", "initial")
	repo.WriteFile("test.txt", "modified")

	err := StashAll("test stash message")
	if err != nil {
		t.Fatalf("StashAll failed: %v", err)
	}

	// Verify working tree is clean
	output := repo.Git("status", "--porcelain")
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected clean working tree after stash, got: %s", output)
	}

	// Verify stash was created
	stashOutput := repo.Git("stash", "list")
	if !strings.Contains(stashOutput, "test stash message") {
		t.Errorf("expected stash with message, got: %s", stashOutput)
	}
}

func TestStashFiles(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.CommitFile("file1.txt", "original1", "initial1")
	repo.CommitFile("file2.txt", "original2", "initial2")

	repo.WriteFile("file1.txt", "modified1")
	repo.WriteFile("file2.txt", "modified2")

	// Stash only file1.txt
	err := StashFiles([]string{"file1.txt"}, "partial stash")
	if err != nil {
		t.Fatalf("StashFiles failed: %v", err)
	}

	// Verify file1 is restored, file2 still modified
	content1 := repo.ReadFile("file1.txt")
	if content1 != "original1" {
		t.Errorf("expected file1 to be restored, got: %s", content1)
	}

	content2 := repo.ReadFile("file2.txt")
	if content2 != "modified2" {
		t.Errorf("expected file2 to still be modified, got: %s", content2)
	}
}

func TestGetBranchStatus(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()

	status := GetBranchStatus()
	if status.Name != "master" && status.Name != "main" {
		t.Errorf("expected branch name to be 'master' or 'main', got %q", status.Name)
	}

	// Without a remote, should have empty Remote
	if status.Remote != "" {
		t.Errorf("expected empty remote without upstream, got %q", status.Remote)
	}
}

func TestGetBranchStatusWithRemote(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.InitialCommit()
	remoteDir := repo.SetupRemote()
	defer os.RemoveAll(remoteDir)

	repo.PushToRemote()

	// Make local commits ahead
	repo.CommitFile("ahead.txt", "ahead content", "ahead commit")

	status := GetBranchStatus()
	if status.Ahead != 1 {
		t.Errorf("expected 1 commit ahead, got %d", status.Ahead)
	}
}

func TestGetLog(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	repo.CommitFile("file1.txt", "content1", "First commit")
	repo.CommitFile("file2.txt", "content2", "Second commit")
	repo.CommitFile("file3.txt", "content3", "Third commit")

	log, err := GetLog(2)
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if !strings.Contains(log, "Third commit") {
		t.Errorf("expected log to contain 'Third commit', got: %s", log)
	}
	if !strings.Contains(log, "Second commit") {
		t.Errorf("expected log to contain 'Second commit', got: %s", log)
	}
	// Should not contain first commit (limit 2)
	lines := strings.Split(log, "\n")
	commitCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "commit ") {
			commitCount++
		}
	}
	if commitCount != 2 {
		t.Errorf("expected 2 commits in log, got %d", commitCount)
	}
}

func TestStageHunk(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create a file with multiple lines
	repo.CommitFile("test.txt", "line1\nline2\nline3\nline4\nline5\n", "initial")

	// Modify the file to have changes in multiple hunks
	repo.WriteFile("test.txt", "line1\nmodified2\nline3\nmodified4\nline5\n")

	// Get the unstaged diff
	diff, err := GetDiff()
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}

	if len(diff.Files) == 0 {
		t.Fatal("expected at least one file in diff")
	}

	if len(diff.Files[0].Hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}

	// Generate and apply patch for first hunk
	patch := diff.Files[0].Hunks[0].GeneratePatch(&diff.Files[0])

	err = StageHunk(patch)
	if err != nil {
		t.Fatalf("StageHunk failed: %v", err)
	}

	// Verify something is staged
	stagedDiff, err := GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff failed: %v", err)
	}

	if stagedDiff.IsEmpty() {
		t.Error("expected staged diff to not be empty after staging hunk")
	}
}

func TestUnstageHunk(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create a file and stage changes
	repo.CommitFile("test.txt", "line1\nline2\nline3\n", "initial")
	repo.WriteFile("test.txt", "line1\nmodified\nline3\n")
	repo.Git("add", "test.txt")

	// Get the staged diff
	stagedDiff, err := GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff failed: %v", err)
	}

	if len(stagedDiff.Files) == 0 || len(stagedDiff.Files[0].Hunks) == 0 {
		t.Fatal("expected staged hunks")
	}

	// Generate patch and unstage
	patch := stagedDiff.Files[0].Hunks[0].GeneratePatch(&stagedDiff.Files[0])

	err = UnstageHunk(patch)
	if err != nil {
		t.Fatalf("UnstageHunk failed: %v", err)
	}

	// Verify nothing is staged
	stagedDiff, err = GetStagedDiff()
	if err != nil {
		t.Fatalf("GetStagedDiff failed: %v", err)
	}

	if !stagedDiff.IsEmpty() {
		t.Error("expected staged diff to be empty after unstaging hunk")
	}
}

func TestDiscardHunk(t *testing.T) {
	repo := NewTestRepo(t)
	defer repo.Cleanup()

	// Create a file with initial content
	repo.CommitFile("test.txt", "line1\nline2\nline3\n", "initial")

	// Modify the file
	repo.WriteFile("test.txt", "line1\nmodified\nline3\n")

	// Get the unstaged diff
	diff, err := GetDiff()
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}

	if len(diff.Files) == 0 || len(diff.Files[0].Hunks) == 0 {
		t.Fatal("expected hunks in diff")
	}

	// Generate patch and discard
	patch := diff.Files[0].Hunks[0].GeneratePatch(&diff.Files[0])

	err = DiscardHunk(patch)
	if err != nil {
		t.Fatalf("DiscardHunk failed: %v", err)
	}

	// Verify file is restored
	content := repo.ReadFile("test.txt")
	if content != "line1\nline2\nline3\n" {
		t.Errorf("expected original content, got: %s", content)
	}
}

package git

import (
	"strings"
)

// FileStatus represents the status of a single file
type FileStatus struct {
	Path                string // Path relative to repo root (for git commands)
	DisplayPath         string // Path relative to cwd (for display)
	IndexStatus         byte   // Status in the index (staged)
	WorkStatus          byte   // Status in the working tree
	OriginalPath        string // For renamed files (repo-relative)
	OriginalDisplayPath string // For renamed files (cwd-relative)
}

// IsStaged returns true if the file has staged changes
func (f FileStatus) IsStaged() bool {
	return f.IndexStatus != ' ' && f.IndexStatus != '?'
}

// IsUnstaged returns true if the file has unstaged changes
func (f FileStatus) IsUnstaged() bool {
	return f.WorkStatus != ' ' && f.WorkStatus != '?'
}

// IsUntracked returns true if the file is untracked
func (f FileStatus) IsUntracked() bool {
	return f.IndexStatus == '?' && f.WorkStatus == '?'
}

// StatusDescription returns a human-readable status
func (f FileStatus) StatusDescription() string {
	if f.IsUntracked() {
		return "untracked"
	}

	var parts []string

	// Index status
	switch f.IndexStatus {
	case 'M':
		parts = append(parts, "staged: modified")
	case 'A':
		parts = append(parts, "staged: added")
	case 'D':
		parts = append(parts, "staged: deleted")
	case 'R':
		parts = append(parts, "staged: renamed")
	case 'C':
		parts = append(parts, "staged: copied")
	}

	// Work tree status
	switch f.WorkStatus {
	case 'M':
		parts = append(parts, "modified")
	case 'D':
		parts = append(parts, "deleted")
	}

	return strings.Join(parts, ", ")
}

// StatusResult holds all file statuses grouped by type
type StatusResult struct {
	Staged    []FileStatus
	Unstaged  []FileStatus
	Untracked []FileStatus
}

// GetStatus returns the current git status
func GetStatus() (*StatusResult, error) {
	output, err := Run("status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}

	result := &StatusResult{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		indexStatus := line[0]
		workStatus := line[1]
		path := line[3:]

		// Handle renamed files (format: "R  old -> new")
		var origPath string
		if indexStatus == 'R' || indexStatus == 'C' {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				origPath = parts[0]
				path = parts[1]
			}
		}

		var origDisplayPath string
		if origPath != "" {
			origDisplayPath = ToDisplayPath(origPath)
		}

		fs := FileStatus{
			Path:                path,
			DisplayPath:         ToDisplayPath(path),
			IndexStatus:         indexStatus,
			WorkStatus:          workStatus,
			OriginalPath:        origPath,
			OriginalDisplayPath: origDisplayPath,
		}

		// Categorize the file
		if fs.IsUntracked() {
			result.Untracked = append(result.Untracked, fs)
		} else {
			if fs.IsStaged() {
				result.Staged = append(result.Staged, fs)
			}
			if fs.IsUnstaged() {
				result.Unstaged = append(result.Unstaged, fs)
			}
		}
	}

	return result, nil
}

// TotalFiles returns the total number of files with changes
func (s *StatusResult) TotalFiles() int {
	seen := make(map[string]bool)
	for _, f := range s.Staged {
		seen[f.Path] = true
	}
	for _, f := range s.Unstaged {
		seen[f.Path] = true
	}
	for _, f := range s.Untracked {
		seen[f.Path] = true
	}
	return len(seen)
}

// IsEmpty returns true if there are no changes
func (s *StatusResult) IsEmpty() bool {
	return len(s.Staged) == 0 && len(s.Unstaged) == 0 && len(s.Untracked) == 0
}

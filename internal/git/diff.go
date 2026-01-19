package git

import (
	"regexp"
	"strings"
)

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    LineType
	Content string
}

// LineType represents the type of a diff line
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
	LineHeader
)

// Hunk represents a single hunk in a diff
type Hunk struct {
	Header    string     // The @@ line
	Lines     []DiffLine // The actual diff lines
	StartOld  int        // Starting line in old file
	CountOld  int        // Number of lines from old file
	StartNew  int        // Starting line in new file
	CountNew  int        // Number of lines in new file
	FilePath  string     // Path to the file this hunk belongs to
	FileIndex int        // Index of the file in the diff
	HunkIndex int        // Index of this hunk within the file
	Staged    bool       // Whether this hunk is staged (true) or unstaged (false)
}

// FileDiff represents the diff for a single file
type FileDiff struct {
	Path   string
	Hunks  []Hunk
	Header []string // File header lines (diff --git, index, ---, +++)
}

// DiffResult holds all file diffs
type DiffResult struct {
	Files []FileDiff
}

// GetDiff returns the unstaged diff
func GetDiff() (*DiffResult, error) {
	output, err := Run("diff")
	if err != nil {
		return nil, err
	}
	return parseDiff(output), nil
}

// GetStagedDiff returns the staged diff
func GetStagedDiff() (*DiffResult, error) {
	output, err := Run("diff", "--cached")
	if err != nil {
		return nil, err
	}
	return parseDiff(output), nil
}

var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)

func parseDiff(output string) *DiffResult {
	result := &DiffResult{}
	if output == "" {
		return result
	}

	lines := strings.Split(output, "\n")
	var currentFile *FileDiff
	var currentHunk *Hunk
	fileIndex := -1

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// New file diff
		if strings.HasPrefix(line, "diff --git") {
			// Save previous file
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				result.Files = append(result.Files, *currentFile)
			}

			fileIndex++
			currentFile = &FileDiff{
				Header: []string{line},
			}
			currentHunk = nil

			// Extract file path (format: "diff --git a/path b/path" or "diff --git i/path w/path")
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				path := parts[3]
				// Remove any single-char prefix like a/, b/, i/, w/
				if len(path) > 2 && path[1] == '/' {
					path = path[2:]
				}
				currentFile.Path = path
			}
			continue
		}

		// File header lines
		if currentFile != nil && currentHunk == nil {
			if strings.HasPrefix(line, "index ") ||
				strings.HasPrefix(line, "---") ||
				strings.HasPrefix(line, "+++") ||
				strings.HasPrefix(line, "new file") ||
				strings.HasPrefix(line, "deleted file") ||
				strings.HasPrefix(line, "old mode") ||
				strings.HasPrefix(line, "new mode") {
				currentFile.Header = append(currentFile.Header, line)
				continue
			}
		}

		// Hunk header
		if strings.HasPrefix(line, "@@") {
			// Save previous hunk
			if currentHunk != nil && currentFile != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			currentHunk = &Hunk{
				Header:    line,
				FilePath:  currentFile.Path,
				FileIndex: fileIndex,
				HunkIndex: len(currentFile.Hunks),
			}

			// Parse hunk header for line numbers
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 4 {
				currentHunk.StartOld = parseInt(matches[1])
				currentHunk.CountOld = parseInt(matches[2])
				if currentHunk.CountOld == 0 && matches[2] == "" {
					currentHunk.CountOld = 1
				}
				currentHunk.StartNew = parseInt(matches[3])
				currentHunk.CountNew = parseInt(matches[4])
				if currentHunk.CountNew == 0 && matches[4] == "" {
					currentHunk.CountNew = 1
				}
			}
			continue
		}

		// Diff content
		if currentHunk != nil {
			var lineType LineType
			switch {
			case strings.HasPrefix(line, "+"):
				lineType = LineAdded
			case strings.HasPrefix(line, "-"):
				lineType = LineRemoved
			default:
				lineType = LineContext
			}

			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Type:    lineType,
				Content: line,
			})
		}
	}

	// Save last file and hunk
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		result.Files = append(result.Files, *currentFile)
	}

	return result
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// TotalHunks returns the total number of hunks across all files
func (d *DiffResult) TotalHunks() int {
	total := 0
	for _, f := range d.Files {
		total += len(f.Hunks)
	}
	return total
}

// IsEmpty returns true if there are no diffs
func (d *DiffResult) IsEmpty() bool {
	return len(d.Files) == 0
}

// GetAllHunks returns all hunks flattened into a single slice
func (d *DiffResult) GetAllHunks() []Hunk {
	var hunks []Hunk
	for _, f := range d.Files {
		hunks = append(hunks, f.Hunks...)
	}
	return hunks
}

// CombinedDiffResult holds both staged and unstaged diffs together
type CombinedDiffResult struct {
	StagedDiff   *DiffResult
	UnstagedDiff *DiffResult
}

// GetCombinedDiff returns both staged and unstaged diffs
func GetCombinedDiff() (*CombinedDiffResult, error) {
	staged, err := GetStagedDiff()
	if err != nil {
		return nil, err
	}
	unstaged, err := GetDiff()
	if err != nil {
		return nil, err
	}
	return &CombinedDiffResult{
		StagedDiff:   staged,
		UnstagedDiff: unstaged,
	}, nil
}

// GetAllHunksCombined returns all hunks from both staged and unstaged, marked with their state
func (c *CombinedDiffResult) GetAllHunksCombined() []Hunk {
	var hunks []Hunk

	// Add staged hunks
	if c.StagedDiff != nil {
		for i := range c.StagedDiff.Files {
			for j := range c.StagedDiff.Files[i].Hunks {
				hunk := c.StagedDiff.Files[i].Hunks[j]
				hunk.Staged = true
				hunks = append(hunks, hunk)
			}
		}
	}

	// Add unstaged hunks
	if c.UnstagedDiff != nil {
		for i := range c.UnstagedDiff.Files {
			for j := range c.UnstagedDiff.Files[i].Hunks {
				hunk := c.UnstagedDiff.Files[i].Hunks[j]
				hunk.Staged = false
				hunks = append(hunks, hunk)
			}
		}
	}

	return hunks
}

// GetFileDiff returns the FileDiff for a given hunk
func (c *CombinedDiffResult) GetFileDiff(h *Hunk) *FileDiff {
	if h.Staged {
		if c.StagedDiff != nil && h.FileIndex < len(c.StagedDiff.Files) {
			return &c.StagedDiff.Files[h.FileIndex]
		}
	} else {
		if c.UnstagedDiff != nil && h.FileIndex < len(c.UnstagedDiff.Files) {
			return &c.UnstagedDiff.Files[h.FileIndex]
		}
	}
	return nil
}

// IsEmpty returns true if there are no diffs in either staged or unstaged
func (c *CombinedDiffResult) IsEmpty() bool {
	stagedEmpty := c.StagedDiff == nil || c.StagedDiff.IsEmpty()
	unstagedEmpty := c.UnstagedDiff == nil || c.UnstagedDiff.IsEmpty()
	return stagedEmpty && unstagedEmpty
}

// GeneratePatch generates a patch string for a single hunk
func (h *Hunk) GeneratePatch(fileDiff *FileDiff) string {
	var sb strings.Builder

	// Write file header
	for _, headerLine := range fileDiff.Header {
		sb.WriteString(headerLine)
		sb.WriteString("\n")
	}

	// Write hunk header
	sb.WriteString(h.Header)
	sb.WriteString("\n")

	// Write hunk content
	for _, line := range h.Lines {
		sb.WriteString(line.Content)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetUntrackedFileDiff returns a diff for an untracked file (showing all content as additions)
func GetUntrackedFileDiff(path string) *FileDiff {
	// Use git diff --no-index to compare /dev/null with the file
	// This command exits with code 1 when there are differences, so we ignore the error
	output, _ := RunAllowFailure("diff", "--no-index", "--", "/dev/null", path)
	if output == "" {
		return nil
	}

	result := parseDiff(output)
	if len(result.Files) > 0 {
		// Fix the file path (--no-index uses full paths)
		result.Files[0].Path = path
		for i := range result.Files[0].Hunks {
			result.Files[0].Hunks[i].FilePath = path
		}
		return &result.Files[0]
	}
	return nil
}

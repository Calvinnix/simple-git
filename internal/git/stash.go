package git

import (
	"fmt"
	"strings"
)

// Stash represents a git stash entry
type Stash struct {
	Index   int
	Message string
	Branch  string
	Date    string
}

// GetStashes returns all stash entries
func GetStashes() ([]Stash, error) {
	// Format: stash@{0}: On branch_name: message
	// or: stash@{0}: WIP on branch_name: hash message
	output, err := Run("stash", "list", "--format=%gd|%s")
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(output) == "" {
		return nil, nil
	}

	var stashes []Stash
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}

		// Parse index from stash@{N}
		var index int
		fmt.Sscanf(parts[0], "stash@{%d}", &index)

		// Parse message - format is usually "On branch: message" or "WIP on branch: hash message"
		message := parts[1]
		branch := ""

		if strings.HasPrefix(message, "On ") {
			// Format: "On branch_name: message"
			colonIdx := strings.Index(message, ": ")
			if colonIdx > 3 {
				branch = message[3:colonIdx]
				message = message[colonIdx+2:]
			}
		} else if strings.HasPrefix(message, "WIP on ") {
			// Format: "WIP on branch_name: hash message"
			colonIdx := strings.Index(message, ": ")
			if colonIdx > 7 {
				branch = message[7:colonIdx]
				// Skip the hash (first word after colon)
				rest := message[colonIdx+2:]
				spaceIdx := strings.Index(rest, " ")
				if spaceIdx > 0 {
					message = rest[spaceIdx+1:]
				} else {
					message = rest
				}
			}
		}

		stashes = append(stashes, Stash{
			Index:   index,
			Message: message,
			Branch:  branch,
		})
	}

	return stashes, nil
}

// GetStashDiff returns the diff for a specific stash
func GetStashDiff(index int) (*CombinedDiffResult, error) {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	output, err := Run("stash", "show", "-p", stashRef)
	if err != nil {
		return nil, err
	}

	// Parse as unstaged diff (stash shows what would be applied)
	diff := parseDiff(output)

	return &CombinedDiffResult{
		StagedDiff:   &DiffResult{},
		UnstagedDiff: diff,
	}, nil
}

// ApplyStash applies a stash without removing it
func ApplyStash(index int) error {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	_, err := Run("stash", "apply", stashRef)
	return err
}

// PopStash applies and removes a stash
func PopStash(index int) error {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	_, err := Run("stash", "pop", stashRef)
	return err
}

// DropStash removes a stash without applying
func DropStash(index int) error {
	stashRef := fmt.Sprintf("stash@{%d}", index)
	_, err := Run("stash", "drop", stashRef)
	return err
}

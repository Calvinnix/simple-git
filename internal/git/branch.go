package git

import (
	"fmt"
	"strings"
)

// Branch represents a git branch
type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	Upstream  string // tracking branch
	Ahead     int
	Behind    int
	LastCommit string // short commit message
}

// GetBranches returns all local branches with their status
func GetBranches() ([]Branch, error) {
	// Get branch list with upstream tracking info
	// Format: %(refname:short)|%(upstream:short)|%(upstream:track)|%(HEAD)|%(subject)
	output, err := Run("for-each-ref", "--format=%(refname:short)|%(upstream:short)|%(upstream:track)|%(HEAD)|%(subject)", "refs/heads/")
	if err != nil {
		return nil, err
	}

	var branches []Branch
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		branch := Branch{
			Name:       parts[0],
			Upstream:   parts[1],
			IsCurrent:  parts[3] == "*",
			LastCommit: parts[4],
		}

		// Parse ahead/behind from track info (e.g., "[ahead 2, behind 1]" or "[ahead 2]")
		track := parts[2]
		if strings.Contains(track, "ahead") {
			fmt.Sscanf(track, "[ahead %d", &branch.Ahead)
		}
		if strings.Contains(track, "behind") {
			if strings.Contains(track, "ahead") {
				// Format: [ahead X, behind Y]
				idx := strings.Index(track, "behind")
				if idx > 0 {
					fmt.Sscanf(track[idx:], "behind %d", &branch.Behind)
				}
			} else {
				// Format: [behind Y]
				fmt.Sscanf(track, "[behind %d", &branch.Behind)
			}
		}

		branches = append(branches, branch)
	}

	return branches, nil
}

// CheckoutBranch switches to the specified branch
func CheckoutBranch(name string) error {
	_, err := Run("checkout", name)
	return err
}

// CreateBranch creates a new branch from HEAD
func CreateBranch(name string) error {
	_, err := Run("checkout", "-b", name)
	return err
}

// DeleteBranch deletes a local branch
func DeleteBranch(name string) error {
	_, err := Run("branch", "-d", name)
	return err
}

// ForceDeleteBranch deletes a local branch even if not fully merged
func ForceDeleteBranch(name string) error {
	_, err := Run("branch", "-D", name)
	return err
}

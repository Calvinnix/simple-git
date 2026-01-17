package ui

import (
	"fmt"
	"strings"

	"simple-git/internal/git"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// BranchesModel is the bubbletea model for the branches tab
type BranchesModel struct {
	branches    []git.Branch
	cursor      int
	showHelp    bool
	confirmMode bool
	inputMode   bool
	branchInput textinput.Model
	lastKey     string
	err         error
	width       int
	height      int
}

// NewBranchesModel creates a new branches model
func NewBranchesModel() BranchesModel {
	ti := textinput.New()
	ti.Placeholder = "New branch name"
	ti.CharLimit = 100
	ti.Width = 40
	return BranchesModel{
		branchInput: ti,
	}
}

// Init initializes the model
func (m BranchesModel) Init() tea.Cmd {
	return refreshBranches
}

func refreshBranches() tea.Msg {
	branches, err := git.GetBranches()
	if err != nil {
		return errMsg{err}
	}
	return branchesMsg{branches}
}

// Update handles messages
func (m BranchesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle help mode
		if m.showHelp {
			if key == "?" || key == "esc" || key == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		// Handle confirm mode (delete branch)
		if m.confirmMode {
			switch key {
			case "y", "Y":
				m.confirmMode = false
				return m, m.doDeleteBranch()
			case "n", "N", "esc":
				m.confirmMode = false
				return m, nil
			}
			return m, nil
		}

		// Handle input mode (new branch)
		if m.inputMode {
			switch key {
			case "enter":
				name := m.branchInput.Value()
				m.inputMode = false
				m.branchInput.Reset()
				m.branchInput.Blur()
				if name != "" {
					return m, m.doCreateBranch(name)
				}
				return m, nil
			case "esc":
				m.inputMode = false
				m.branchInput.Reset()
				m.branchInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.branchInput, cmd = m.branchInput.Update(msg)
				return m, cmd
			}
		}

		// Check for gg sequence
		if m.lastKey == "g" && key == "g" {
			m.lastKey = ""
			m.cursor = 0
			return m, nil
		}

		if key == "g" {
			m.lastKey = "g"
			return m, nil
		}
		m.lastKey = ""

		switch key {
		case "?":
			m.showHelp = true
			return m, nil
		case "j", "down":
			if len(m.branches) > 0 {
				m.cursor = min(m.cursor+1, len(m.branches)-1)
			}
			return m, nil
		case "k", "up":
			if len(m.branches) > 0 {
				m.cursor = max(m.cursor-1, 0)
			}
			return m, nil
		case "G":
			if len(m.branches) > 0 {
				m.cursor = len(m.branches) - 1
			}
			return m, nil
		case "enter", "l":
			// Checkout selected branch
			if len(m.branches) > 0 && m.cursor < len(m.branches) {
				branch := m.branches[m.cursor]
				if !branch.IsCurrent {
					return m, m.doCheckoutBranch(branch.Name)
				}
			}
			return m, nil
		case "n":
			// Create new branch
			m.inputMode = true
			m.branchInput.Focus()
			return m, textinput.Blink
		case "d":
			// Delete branch (with confirmation)
			if len(m.branches) > 0 && m.cursor < len(m.branches) {
				branch := m.branches[m.cursor]
				if !branch.IsCurrent {
					m.confirmMode = true
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case branchesMsg:
		m.branches = msg.branches
		if m.cursor >= len(m.branches) {
			m.cursor = max(0, len(m.branches)-1)
		}
		// Find and position cursor on current branch
		for i, b := range m.branches {
			if b.IsCurrent {
				m.cursor = i
				break
			}
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m BranchesModel) doCheckoutBranch(name string) tea.Cmd {
	return func() tea.Msg {
		err := git.CheckoutBranch(name)
		if err != nil {
			return errMsg{err}
		}
		return refreshBranches()
	}
}

func (m BranchesModel) doCreateBranch(name string) tea.Cmd {
	return func() tea.Msg {
		err := git.CreateBranch(name)
		if err != nil {
			return errMsg{err}
		}
		return refreshBranches()
	}
}

func (m BranchesModel) doDeleteBranch() tea.Cmd {
	if m.cursor >= len(m.branches) {
		return nil
	}
	branch := m.branches[m.cursor]
	return func() tea.Msg {
		err := git.DeleteBranch(branch.Name)
		if err != nil {
			return errMsg{err}
		}
		return refreshBranches()
	}
}

// View renders the model
func (m BranchesModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	var sb strings.Builder

	if m.err != nil {
		sb.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
		sb.WriteString("\n\n")
	}

	if len(m.branches) == 0 {
		sb.WriteString(StyleMuted.Render("No branches found"))
		sb.WriteString("\n")
		return sb.String()
	}

	sb.WriteString(StyleSectionHeader.Render("Branches"))
	sb.WriteString("\n\n")

	for i, branch := range m.branches {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		// Branch name with current indicator
		name := branch.Name
		if branch.IsCurrent {
			name = "* " + name
		} else {
			name = "  " + name
		}

		// Style based on selection
		var line string
		if i == m.cursor {
			line = StyleSelected.Render(prefix + name)
		} else if branch.IsCurrent {
			line = prefix + StyleStaged.Render(name)
		} else {
			line = prefix + name
		}

		sb.WriteString(line)

		// Show tracking info
		if branch.Upstream != "" {
			trackInfo := ""
			if branch.Ahead > 0 && branch.Behind > 0 {
				trackInfo = fmt.Sprintf(" [+%d/-%d]", branch.Ahead, branch.Behind)
			} else if branch.Ahead > 0 {
				trackInfo = fmt.Sprintf(" [+%d]", branch.Ahead)
			} else if branch.Behind > 0 {
				trackInfo = fmt.Sprintf(" [-%d]", branch.Behind)
			}
			if trackInfo != "" {
				sb.WriteString(StyleMuted.Render(trackInfo))
			}
		}

		// Show last commit message (truncated)
		if branch.LastCommit != "" {
			msg := branch.LastCommit
			maxLen := 50
			if len(msg) > maxLen {
				msg = msg[:maxLen-3] + "..."
			}
			sb.WriteString(StyleMuted.Render(" - " + msg))
		}

		sb.WriteString("\n")
	}

	// Confirm prompt
	if m.confirmMode && m.cursor < len(m.branches) {
		sb.WriteString("\n")
		branch := m.branches[m.cursor]
		sb.WriteString(StyleConfirm.Render(fmt.Sprintf("Delete branch '%s'? (y/n) ", branch.Name)))
	}

	// Input mode
	if m.inputMode {
		sb.WriteString("\n")
		sb.WriteString("New branch name: ")
		sb.WriteString(m.branchInput.View())
		sb.WriteString(StyleMuted.Render("  (enter to create, esc to cancel)"))
	}

	return sb.String()
}

func (m BranchesModel) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(StyleHelpTitle.Render("Branches Shortcuts"))
	sb.WriteString("\n\n")

	help := []struct {
		key  string
		desc string
	}{
		{"j/k/↑/↓", "Move down/up"},
		{"gg", "Go to top"},
		{"G", "Go to bottom"},
		{"Enter/l", "Checkout branch"},
		{"n", "Create new branch"},
		{"d", "Delete branch"},
		{"?", "Toggle help"},
		{"h/←/ESC", "Go back"},
	}

	for _, h := range help {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			StyleHelpKey.Render(fmt.Sprintf("%-8s", h.key)),
			StyleHelpDesc.Render(h.desc)))
	}

	return sb.String()
}

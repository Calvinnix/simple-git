package ui

import (
	"fmt"
	"strings"

	"go-on-git/internal/git"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// BranchesModel is the bubbletea model for the branches tab
type BranchesModel struct {
	branches            []git.Branch
	cursor              int
	showHelp            bool
	inputMode           bool
	deleteConfirmMode   bool
	forceDeleteMode     bool
	pendingDeleteBranch string
	branchInput         textinput.Model
	deleteInput         textinput.Model
	lastKey             string
	err                 error
	width               int
	height              int
}

// NewBranchesModel creates a new branches model
func NewBranchesModel() BranchesModel {
	ti := textinput.New()
	ti.Placeholder = "New branch name"
	ti.CharLimit = 100
	ti.Width = 40

	di := textinput.New()
	di.Placeholder = "Type branch name to confirm"
	di.CharLimit = 100
	di.Width = 40

	return BranchesModel{
		branchInput: ti,
		deleteInput: di,
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
			if key == Keys.Help || key == "esc" || key == Keys.Quit {
				m.showHelp = false
			}
			return m, nil
		}

		// Handle delete confirm mode (type branch name)
		if m.deleteConfirmMode {
			switch key {
			case "enter":
				typedName := m.deleteInput.Value()
				m.deleteConfirmMode = false
				m.deleteInput.Reset()
				m.deleteInput.Blur()
				if m.cursor < len(m.branches) && typedName == m.branches[m.cursor].Name {
					return m, m.doDeleteBranch()
				}
				return m, nil
			case "esc":
				m.deleteConfirmMode = false
				m.deleteInput.Reset()
				m.deleteInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.deleteInput, cmd = m.deleteInput.Update(msg)
				return m, cmd
			}
		}

		// Handle force delete confirm mode
		if m.forceDeleteMode {
			switch key {
			case "y", "Y":
				m.forceDeleteMode = false
				m.err = nil
				return m, m.doForceDeleteBranch()
			case "n", "N", "esc":
				m.forceDeleteMode = false
				m.pendingDeleteBranch = ""
				m.err = nil
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
		if m.lastKey == Keys.Top && key == Keys.Top {
			m.lastKey = ""
			m.cursor = 0
			return m, nil
		}

		if key == Keys.Top {
			m.lastKey = Keys.Top
			return m, nil
		}
		m.lastKey = ""

		switch key {
		case Keys.Help:
			m.showHelp = true
			return m, nil
		case Keys.Down, "down":
			if len(m.branches) > 0 {
				m.cursor = min(m.cursor+1, len(m.branches)-1)
			}
			return m, nil
		case Keys.Up, "up":
			if len(m.branches) > 0 {
				m.cursor = max(m.cursor-1, 0)
			}
			return m, nil
		case Keys.Bottom:
			if len(m.branches) > 0 {
				m.cursor = len(m.branches) - 1
			}
			return m, nil
		case Keys.Right, "right", "enter":
			// Checkout selected branch
			if len(m.branches) > 0 && m.cursor < len(m.branches) {
				branch := m.branches[m.cursor]
				if !branch.IsCurrent {
					return m, m.doCheckoutBranch(branch.Name)
				}
			}
			return m, nil
		case Keys.NewBranch:
			// Create new branch
			m.inputMode = true
			m.branchInput.Focus()
			return m, textinput.Blink
		case Keys.Delete:
			// Delete branch (with confirmation)
			if len(m.branches) > 0 && m.cursor < len(m.branches) {
				branch := m.branches[m.cursor]
				if !branch.IsCurrent {
					m.deleteConfirmMode = true
					m.deleteInput.Focus()
					return m, textinput.Blink
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

	case branchDeleteFailedMsg:
		m.err = msg.err
		m.pendingDeleteBranch = msg.branchName
		m.forceDeleteMode = true
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
			// Check if the error is about unmerged branch
			if strings.Contains(err.Error(), "not fully merged") {
				return branchDeleteFailedMsg{branchName: branch.Name, err: err}
			}
			return errMsg{err}
		}
		return refreshBranches()
	}
}

func (m BranchesModel) doForceDeleteBranch() tea.Cmd {
	if m.pendingDeleteBranch == "" {
		return nil
	}
	branchName := m.pendingDeleteBranch
	return func() tea.Msg {
		err := git.ForceDeleteBranch(branchName)
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

	if m.err != nil && !m.forceDeleteMode {
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

	// Delete confirm prompt
	if m.deleteConfirmMode && m.cursor < len(m.branches) {
		sb.WriteString("\n")
		branch := m.branches[m.cursor]
		sb.WriteString(fmt.Sprintf("Type '%s' to delete: ", branch.Name))
		sb.WriteString(m.deleteInput.View())
		sb.WriteString(StyleMuted.Render("  (esc to cancel)"))
	}

	// Force delete prompt
	if m.forceDeleteMode {
		sb.WriteString("\n")
		if m.err != nil {
			sb.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
			sb.WriteString("\n")
		}
		sb.WriteString(StyleConfirm.Render(fmt.Sprintf("Force delete '%s'? (y/n) ", m.pendingDeleteBranch)))
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

	moveKeys := formatKeyList(Keys.Down, Keys.Up, "↓", "↑")
	topKey := formatDoubleKey(Keys.Top)
	checkoutKeys := formatKeyList(Keys.Right, "Enter", "→")
	backKeys := formatKeyList(Keys.Left, "←", "ESC")

	help := []struct {
		key  string
		desc string
	}{
		{moveKeys, "Move down/up"},
		{topKey, "Go to top"},
		{Keys.Bottom, "Go to bottom"},
		{checkoutKeys, "Checkout branch"},
		{Keys.NewBranch, "Create new branch"},
		{Keys.Delete, "Delete branch"},
		{Keys.Help, "Toggle help"},
		{backKeys, "Go back"},
	}

	for _, h := range help {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			StyleHelpKey.Render(fmt.Sprintf("%-8s", h.key)),
			StyleHelpDesc.Render(h.desc)))
	}

	return sb.String()
}

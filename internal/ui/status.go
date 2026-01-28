package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"go-on-git/internal/git"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusItem represents a selectable item in the status view
type StatusItem struct {
	File    git.FileStatus
	Section string // "staged", "unstaged", "untracked"
}

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDiscard
	confirmPush
	confirmPushNew
	confirmStash
)

type stashMode int

const (
	stashNone stashMode = iota
	stashFiles
	stashAll
)

// StatusModel is the bubbletea model for the status view
type StatusModel struct {
	items           []StatusItem
	cursor          int
	scrollOffset    int
	selected        map[int]bool
	visualMode      bool
	visualStart     int
	status          *git.StatusResult
	branchStatus    git.BranchStatus
	showHelp        bool
	showVerboseHelp bool
	confirmMode     confirmAction
	confirmInput    string
	pendingPushRemote   string
	stashMode           stashMode
	stashInput          textinput.Model
	pendingStashMode    stashMode
	pendingStashMessage string
	commitMode          bool
	commitInput     textinput.Model
	quitting        bool
	lastKey         string
	err             error
	width           int
	height          int
}

// NewStatusModel creates a new status model
func NewStatusModel() StatusModel {
	return NewStatusModelWithHelp(false)
}

// NewStatusModelWithHelp creates a new status model with optional help mode
func NewStatusModelWithHelp(showHelp bool) StatusModel {
	ti := textinput.New()
	ti.Placeholder = "Stash message (optional)"
	ti.CharLimit = 200
	ti.Width = 40

	ci := textinput.New()
	ci.Placeholder = "Commit message"
	ci.CharLimit = 200
	ci.Width = 50

	return StatusModel{
		selected:        make(map[int]bool),
		stashInput:      ti,
		commitInput:     ci,
		showVerboseHelp: showHelp,
	}
}

// isBlocking returns true if the model is in a mode that shouldn't be interrupted by auto-refresh
func (m StatusModel) isBlocking() bool {
	return m.confirmMode != confirmNone || m.commitMode || m.stashMode != stashNone || m.showHelp || m.visualMode || len(m.selected) > 0
}

// Init initializes the model
func (m StatusModel) Init() tea.Cmd {
	return refreshStatus
}

// Update handles messages
func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle compact help overlay
		if m.showHelp {
			if key == Keys.Help || key == "esc" || key == Keys.Quit {
				m.showHelp = false
			}
			return m, nil
		}

		// Handle confirm mode
		if m.confirmMode != confirmNone {
			// Discard and stash require typing 'yes'
			if m.confirmMode == confirmDiscard || m.confirmMode == confirmStash {
				switch key {
				case "backspace":
					if len(m.confirmInput) > 0 {
						m.confirmInput = m.confirmInput[:len(m.confirmInput)-1]
					}
					return m, nil
				case "enter":
					if m.confirmInput == "yes" {
						action := m.confirmMode
						m.confirmMode = confirmNone
						m.confirmInput = ""
						switch action {
						case confirmDiscard:
							return m, m.doDiscard()
						case confirmStash:
							mode := m.pendingStashMode
							message := m.pendingStashMessage
							m.pendingStashMode = stashNone
							m.pendingStashMessage = ""
							return m, m.doStash(mode, message)
						}
					}
					return m, nil
				case "esc":
					m.confirmMode = confirmNone
					m.confirmInput = ""
					m.pendingStashMode = stashNone
					m.pendingStashMessage = ""
					return m, nil
				default:
					// Only accept lowercase letters for typing "yes"
					if len(key) == 1 && key[0] >= 'a' && key[0] <= 'z' {
						m.confirmInput += key
					}
					return m, nil
				}
			}
			// Simple y/n confirmation for push
			switch key {
			case "y", "Y":
				action := m.confirmMode
				remote := m.pendingPushRemote
				m.confirmMode = confirmNone
				m.pendingPushRemote = ""
				if action == confirmPushNew {
					return m, m.doPushSetUpstream(remote)
				}
				return m, m.doPush()
			case "n", "N", "esc":
				m.confirmMode = confirmNone
				m.pendingPushRemote = ""
				return m, nil
			}
			return m, nil
		}

		// Handle stash input mode
		if m.stashMode != stashNone {
			switch key {
			case "enter":
				// Store pending stash and enter confirm mode
				m.pendingStashMode = m.stashMode
				m.pendingStashMessage = m.stashInput.Value()
				m.stashMode = stashNone
				m.stashInput.Reset()
				m.stashInput.Blur()
				m.confirmMode = confirmStash
				return m, nil
			case "esc":
				m.stashMode = stashNone
				m.stashInput.Reset()
				m.stashInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.stashInput, cmd = m.stashInput.Update(msg)
				return m, cmd
			}
		}

		// Handle commit input mode
		if m.commitMode {
			switch key {
			case "enter":
				message := m.commitInput.Value()
				m.commitMode = false
				m.commitInput.Reset()
				m.commitInput.Blur()
				if message != "" {
					return m, m.doCommit(message)
				}
				return m, nil
			case "esc":
				m.commitMode = false
				m.commitInput.Reset()
				m.commitInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.commitInput, cmd = m.commitInput.Update(msg)
				return m, cmd
			}
		}

		// Check for gg sequence (go to top)
		if m.lastKey == Keys.Top && key == Keys.Top {
			m.lastKey = ""
			m.cursor = 0
			m.ensureCursorVisible()
			if m.visualMode {
				m.updateVisualSelection()
			}
			return m, nil
		}

		if key == Keys.Top {
			m.lastKey = Keys.Top
			return m, nil
		}
		m.lastKey = ""

		switch {
		case key == Keys.Quit:
			if m.visualMode {
				m.visualMode = false
				m.selected = make(map[int]bool)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case key == "esc":
			if m.visualMode || len(m.selected) > 0 {
				m.visualMode = false
				m.selected = make(map[int]bool)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case key == Keys.Help:
			m.showHelp = true
			return m, nil
		case key == Keys.VerboseHelp:
			m.showVerboseHelp = !m.showVerboseHelp
			return m, nil
		case key == Keys.Visual || key == "V":
			if m.visualMode && key == Keys.Visual {
				m.visualMode = false
				m.selected = make(map[int]bool)
			} else if !m.visualMode {
				m.visualMode = true
				m.visualStart = m.cursor
				m.selected = make(map[int]bool)
				m.selected[m.cursor] = true
			}
			return m, nil
		case key == Keys.Down || key == "down":
			if len(m.items) > 0 {
				m.cursor = min(m.cursor+1, len(m.items)-1)
				m.ensureCursorVisible()
				if m.visualMode {
					m.updateVisualSelection()
				}
			}
			return m, nil
		case key == Keys.Up || key == "up":
			if len(m.items) > 0 {
				m.cursor = max(m.cursor-1, 0)
				m.ensureCursorVisible()
				if m.visualMode {
					m.updateVisualSelection()
				}
			}
			return m, nil
		case key == Keys.Bottom:
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				m.ensureCursorVisible()
				if m.visualMode {
					m.updateVisualSelection()
				}
			}
			return m, nil
		case key == Keys.Select || key == "left":
			// Toggle selection of current item (non-contiguous multi-select)
			if len(m.items) > 0 && !m.visualMode {
				if m.selected[m.cursor] {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = true
				}
			}
			return m, nil
		case key == " ":
			return m, m.toggleStage()
		case key == Keys.Stage:
			return m, m.stageFiles()
		case key == Keys.StageAll:
			return m, m.stageAll()
		case key == Keys.Unstage:
			return m, m.unstageFiles()
		case key == Keys.UnstageAll:
			return m, m.unstageAll()
		case key == Keys.Discard:
			if len(m.items) > 0 && (len(m.selected) > 0 || !m.visualMode) {
				m.confirmMode = confirmDiscard
			}
			return m, nil
		case key == Keys.Push:
			if m.branchStatus.Remote != "" && m.branchStatus.Ahead > 0 {
				m.confirmMode = confirmPush
				return m, nil
			}
			if m.branchStatus.Remote == "" {
				// No upstream - detect remotes and offer to push with -u
				remotes, err := git.GetRemotes()
				if err != nil {
					m.err = err
					return m, nil
				}
				if len(remotes) == 0 {
					m.err = fmt.Errorf("no remotes configured")
					return m, nil
				}
				// Use first remote (typically "origin")
				m.pendingPushRemote = remotes[0]
				m.confirmMode = confirmPushNew
				return m, nil
			}
			return m, m.doPush()
		case key == Keys.Commit:
			// Inline commit with message
			if m.status != nil && len(m.status.Staged) > 0 {
				m.commitMode = true
				m.commitInput.Focus()
				return m, textinput.Blink
			}
			return m, nil
		case key == Keys.CommitEdit:
			// Run git commit with editor
			m.quitting = true
			return m, runGitCommit()
		case key == Keys.Stash:
			// Stash selected file(s)
			if len(m.items) > 0 {
				m.stashMode = stashFiles
				m.stashInput.Focus()
				return m, textinput.Blink
			}
			return m, nil
		case key == Keys.StashAll:
			// Stash all changes
			if len(m.items) > 0 {
				m.stashMode = stashAll
				m.stashInput.Focus()
				return m, textinput.Blink
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statusMsg:
		m.status = msg.status
		m.branchStatus = msg.branchStatus
		m.items = buildItems(msg.status)
		if m.cursor >= len(m.items) {
			m.cursor = max(0, len(m.items)-1)
		}
		m.ensureCursorVisible()
		m.selected = make(map[int]bool)
		m.visualMode = false
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m *StatusModel) updateVisualSelection() {
	m.selected = make(map[int]bool)
	start, end := m.visualStart, m.cursor
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		m.selected[i] = true
	}
}

// visibleLines returns the number of item lines that can be displayed
func (m StatusModel) visibleLines() int {
	// Reserve lines for: branch info (~3), section headers (~3), blank lines (~4),
	// help bar (3 if shown), and some buffer
	reserved := 12
	if m.showVerboseHelp {
		reserved += 3
	}
	if m.height <= reserved {
		return 10 // fallback minimum
	}
	return m.height - reserved
}

// ensureCursorVisible adjusts scrollOffset to keep cursor in view
func (m *StatusModel) ensureCursorVisible() {
	visible := m.visibleLines()
	if visible <= 0 {
		return
	}

	// If cursor is above visible area, scroll up
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}

	// If cursor is below visible area, scroll down
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}

	// Clamp scrollOffset to valid range
	maxOffset := len(m.items) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m StatusModel) getSelectedItems() []StatusItem {
	if len(m.selected) > 0 {
		var items []StatusItem
		cursorIncluded := false
		for i := 0; i < len(m.items); i++ {
			if m.selected[i] {
				items = append(items, m.items[i])
				if i == m.cursor {
					cursorIncluded = true
				}
			}
		}
		// Always include cursor item if not already selected
		if !cursorIncluded && m.cursor < len(m.items) {
			items = append(items, m.items[m.cursor])
		}
		return items
	}
	if m.cursor < len(m.items) {
		return []StatusItem{m.items[m.cursor]}
	}
	return nil
}

func buildItems(status *git.StatusResult) []StatusItem {
	if status == nil {
		return nil
	}

	var items []StatusItem

	for _, f := range status.Staged {
		items = append(items, StatusItem{File: f, Section: "staged"})
	}
	for _, f := range status.Unstaged {
		items = append(items, StatusItem{File: f, Section: "unstaged"})
	}
	for _, f := range status.Untracked {
		items = append(items, StatusItem{File: f, Section: "untracked"})
	}

	return items
}

func (m StatusModel) toggleStage() tea.Cmd {
	items := m.getSelectedItems()
	if len(items) == 0 {
		return nil
	}

	return func() tea.Msg {
		for _, item := range items {
			var err error
			switch item.Section {
			case "staged":
				err = git.UnstageFile(item.File.Path)
			case "unstaged", "untracked":
				err = git.StageFile(item.File.Path)
			}
			if err != nil {
				return errMsg{err}
			}
		}
		return refreshStatus()
	}
}

func (m StatusModel) stageFiles() tea.Cmd {
	items := m.getSelectedItems()
	if len(items) == 0 {
		return nil
	}

	return func() tea.Msg {
		for _, item := range items {
			// Only stage unstaged/untracked files
			if item.Section == "unstaged" || item.Section == "untracked" {
				if err := git.StageFile(item.File.Path); err != nil {
					return errMsg{err}
				}
			}
		}
		return refreshStatus()
	}
}

func (m StatusModel) unstageFiles() tea.Cmd {
	items := m.getSelectedItems()
	if len(items) == 0 {
		return nil
	}

	return func() tea.Msg {
		for _, item := range items {
			// Only unstage staged files
			if item.Section == "staged" {
				if err := git.UnstageFile(item.File.Path); err != nil {
					return errMsg{err}
				}
			}
		}
		return refreshStatus()
	}
}

func (m StatusModel) stageAll() tea.Cmd {
	return func() tea.Msg {
		if err := git.StageAll(); err != nil {
			return errMsg{err}
		}
		return refreshStatus()
	}
}

func (m StatusModel) unstageAll() tea.Cmd {
	return func() tea.Msg {
		if err := git.UnstageAll(); err != nil {
			return errMsg{err}
		}
		return refreshStatus()
	}
}

func (m StatusModel) doDiscard() tea.Cmd {
	items := m.getSelectedItems()
	if len(items) == 0 {
		return nil
	}

	return func() tea.Msg {
		for _, item := range items {
			var err error
			switch item.Section {
			case "staged":
				// Only unstage, preserving working tree changes (like git restore --staged)
				err = git.UnstageFile(item.File.Path)
			case "unstaged":
				err = git.DiscardFile(item.File.Path)
			case "untracked":
				err = git.DiscardUntracked(item.File.Path)
			}
			if err != nil {
				return errMsg{err}
			}
		}
		return refreshStatus()
	}
}

func (m StatusModel) doPush() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		if err != nil {
			return errMsg{err}
		}
		return refreshStatus()
	}
}

func (m StatusModel) doPushSetUpstream(remote string) tea.Cmd {
	branch := m.branchStatus.Name
	return func() tea.Msg {
		err := git.PushSetUpstream(remote, branch)
		if err != nil {
			return errMsg{err}
		}
		return refreshStatus()
	}
}

func (m StatusModel) doCommit(message string) tea.Cmd {
	return func() tea.Msg {
		err := git.Commit(message)
		if err != nil {
			return errMsg{err}
		}
		return refreshStatus()
	}
}

func (m StatusModel) doStash(mode stashMode, message string) tea.Cmd {
	if mode == stashAll {
		return func() tea.Msg {
			if err := git.StashAll(message); err != nil {
				return errMsg{err}
			}
			return refreshStatus()
		}
	}

	// Stash specific files
	items := m.getSelectedItems()
	if len(items) == 0 {
		return nil
	}

	paths := make([]string, len(items))
	for i, item := range items {
		paths[i] = item.File.Path
	}

	return func() tea.Msg {
		if err := git.StashFiles(paths, message); err != nil {
			return errMsg{err}
		}
		return refreshStatus()
	}
}

func (m StatusModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	var content strings.Builder

	if m.err != nil {
		content.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
		content.WriteString("\n")
	}

	if m.status == nil {
		content.WriteString(StyleMuted.Render("Loading..."))
		content.WriteString("\n")
		return content.String()
	}

	if m.status.IsEmpty() {
		// Branch status info
		content.WriteString(fmt.Sprintf("On branch %s", m.branchStatus.Name))
		content.WriteString("\n")
		if m.branchStatus.Remote != "" {
			if m.branchStatus.Ahead > 0 && m.branchStatus.Behind > 0 {
				content.WriteString(fmt.Sprintf("Your branch and '%s' have diverged,", m.branchStatus.Remote))
				content.WriteString("\n")
				content.WriteString(fmt.Sprintf("and have %d and %d different commits each, respectively.", m.branchStatus.Ahead, m.branchStatus.Behind))
			} else if m.branchStatus.Ahead > 0 {
				if m.branchStatus.Ahead == 1 {
					content.WriteString(fmt.Sprintf("Your branch is ahead of '%s' by 1 commit.", m.branchStatus.Remote))
				} else {
					content.WriteString(fmt.Sprintf("Your branch is ahead of '%s' by %d commits.", m.branchStatus.Remote, m.branchStatus.Ahead))
				}
			} else if m.branchStatus.Behind > 0 {
				if m.branchStatus.Behind == 1 {
					content.WriteString(fmt.Sprintf("Your branch is behind '%s' by 1 commit.", m.branchStatus.Remote))
				} else {
					content.WriteString(fmt.Sprintf("Your branch is behind '%s' by %d commits.", m.branchStatus.Remote, m.branchStatus.Behind))
				}
			} else {
				content.WriteString(fmt.Sprintf("Your branch is up to date with '%s'.", m.branchStatus.Remote))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
		content.WriteString(StyleEmpty.Render("Nothing to commit, working tree clean"))
		content.WriteString("\n")

		// Confirm prompt for push (when working tree is clean but have unpushed commits)
		if m.confirmMode == confirmPush {
			content.WriteString("\n")
			if m.branchStatus.Ahead == 1 {
				content.WriteString(fmt.Sprintf("Push 1 commit to '%s'? (y/n) ", m.branchStatus.Remote))
			} else {
				content.WriteString(fmt.Sprintf("Push %d commits to '%s'? (y/n) ", m.branchStatus.Ahead, m.branchStatus.Remote))
			}
		} else if m.confirmMode == confirmPushNew {
			content.WriteString("\n")
			content.WriteString(fmt.Sprintf("Push branch '%s' to '%s'? (y/n) ", m.branchStatus.Name, m.pendingPushRemote))
		}

		if m.showVerboseHelp {
			content.WriteString("\n")
			content.WriteString(m.renderHelpBar())
		}
		return content.String()
	}

	// Branch status info
	content.WriteString(fmt.Sprintf("On branch %s", m.branchStatus.Name))
	content.WriteString("\n")
	if m.branchStatus.Remote != "" {
		if m.branchStatus.Ahead > 0 && m.branchStatus.Behind > 0 {
			content.WriteString(fmt.Sprintf("Your branch and '%s' have diverged,", m.branchStatus.Remote))
			content.WriteString("\n")
			content.WriteString(fmt.Sprintf("and have %d and %d different commits each, respectively.", m.branchStatus.Ahead, m.branchStatus.Behind))
		} else if m.branchStatus.Ahead > 0 {
			if m.branchStatus.Ahead == 1 {
				content.WriteString(fmt.Sprintf("Your branch is ahead of '%s' by 1 commit.", m.branchStatus.Remote))
			} else {
				content.WriteString(fmt.Sprintf("Your branch is ahead of '%s' by %d commits.", m.branchStatus.Remote, m.branchStatus.Ahead))
			}
		} else if m.branchStatus.Behind > 0 {
			if m.branchStatus.Behind == 1 {
				content.WriteString(fmt.Sprintf("Your branch is behind '%s' by 1 commit.", m.branchStatus.Remote))
			} else {
				content.WriteString(fmt.Sprintf("Your branch is behind '%s' by %d commits.", m.branchStatus.Remote, m.branchStatus.Behind))
			}
		} else {
			content.WriteString(fmt.Sprintf("Your branch is up to date with '%s'.", m.branchStatus.Remote))
		}
		content.WriteString("\n")
	}
	if m.visualMode && !m.quitting {
		content.WriteString(StyleVisual.Render("-- VISUAL --"))
	}
	content.WriteString("\n")

	// Calculate visible range
	visibleStart := m.scrollOffset
	visibleEnd := m.scrollOffset + m.visibleLines()
	if visibleEnd > len(m.items) {
		visibleEnd = len(m.items)
	}

	// Show scroll indicator at top if scrolled down
	if m.scrollOffset > 0 {
		content.WriteString(StyleMuted.Render(fmt.Sprintf("  ↑ %d more above", m.scrollOffset)))
		content.WriteString("\n")
	}

	itemIndex := 0

	if len(m.status.Staged) > 0 {
		stagedStart := itemIndex
		stagedEnd := itemIndex + len(m.status.Staged)
		// Show section header if any staged items are visible
		if stagedEnd > visibleStart && stagedStart < visibleEnd {
			content.WriteString("Changes to be committed:\n")
			for i, f := range m.status.Staged {
				if itemIndex >= visibleStart && itemIndex < visibleEnd {
					content.WriteString(m.renderItem(itemIndex, f, "staged"))
					content.WriteString("\n")
				}
				itemIndex++
				// Show trailing indicator if more staged items below visible area
				if i == len(m.status.Staged)-1 && itemIndex <= visibleEnd {
					content.WriteString("\n")
				}
			}
		} else {
			itemIndex += len(m.status.Staged)
		}
	}

	if len(m.status.Unstaged) > 0 {
		unstagedStart := itemIndex
		unstagedEnd := itemIndex + len(m.status.Unstaged)
		// Show section header if any unstaged items are visible
		if unstagedEnd > visibleStart && unstagedStart < visibleEnd {
			content.WriteString("Changes not staged for commit:\n")
			for i, f := range m.status.Unstaged {
				if itemIndex >= visibleStart && itemIndex < visibleEnd {
					content.WriteString(m.renderItem(itemIndex, f, "unstaged"))
					content.WriteString("\n")
				}
				itemIndex++
				if i == len(m.status.Unstaged)-1 && itemIndex <= visibleEnd {
					content.WriteString("\n")
				}
			}
		} else {
			itemIndex += len(m.status.Unstaged)
		}
	}

	if len(m.status.Untracked) > 0 {
		untrackedStart := itemIndex
		untrackedEnd := itemIndex + len(m.status.Untracked)
		// Show section header if any untracked items are visible
		if untrackedEnd > visibleStart && untrackedStart < visibleEnd {
			content.WriteString("Untracked files:\n")
			for i, f := range m.status.Untracked {
				if itemIndex >= visibleStart && itemIndex < visibleEnd {
					content.WriteString(m.renderItem(itemIndex, f, "untracked"))
					content.WriteString("\n")
				}
				itemIndex++
				if i == len(m.status.Untracked)-1 && itemIndex <= visibleEnd {
					content.WriteString("\n")
				}
			}
		} else {
			itemIndex += len(m.status.Untracked)
		}
	}

	// Show scroll indicator at bottom if more items below
	if visibleEnd < len(m.items) {
		content.WriteString(StyleMuted.Render(fmt.Sprintf("  ↓ %d more below", len(m.items)-visibleEnd)))
		content.WriteString("\n")
	}

	// Confirm prompt (only shown when confirming)
	if m.confirmMode == confirmDiscard {
		items := m.getSelectedItems()
		if len(items) == 1 {
			content.WriteString(StyleConfirm.Render(fmt.Sprintf("Discard '%s'? Type 'yes' to confirm: %s", items[0].File.DisplayPath, m.confirmInput)))
		} else {
			content.WriteString(StyleConfirm.Render(fmt.Sprintf("Discard %d files? Type 'yes' to confirm: %s", len(items), m.confirmInput)))
		}
	} else if m.confirmMode == confirmPush {
		if m.branchStatus.Ahead == 1 {
			content.WriteString(fmt.Sprintf("Push 1 commit to '%s'? (y/n) ", m.branchStatus.Remote))
		} else {
			content.WriteString(fmt.Sprintf("Push %d commits to '%s'? (y/n) ", m.branchStatus.Ahead, m.branchStatus.Remote))
		}
	} else if m.confirmMode == confirmPushNew {
		content.WriteString(fmt.Sprintf("Push branch '%s' to '%s'? (y/n) ", m.branchStatus.Name, m.pendingPushRemote))
	} else if m.confirmMode == confirmStash {
		if m.pendingStashMode == stashAll {
			content.WriteString(StyleConfirm.Render(fmt.Sprintf("Stash all changes? Type 'yes' to confirm: %s", m.confirmInput)))
		} else {
			items := m.getSelectedItems()
			if len(items) == 1 {
				content.WriteString(StyleConfirm.Render(fmt.Sprintf("Stash '%s'? Type 'yes' to confirm: %s", items[0].File.DisplayPath, m.confirmInput)))
			} else {
				content.WriteString(StyleConfirm.Render(fmt.Sprintf("Stash %d files? Type 'yes' to confirm: %s", len(items), m.confirmInput)))
			}
		}
	} else if m.stashMode != stashNone {
		if m.stashMode == stashAll {
			content.WriteString("Stash all changes\n")
		} else {
			items := m.getSelectedItems()
			if len(items) == 1 {
				content.WriteString(fmt.Sprintf("Stash '%s'\n", items[0].File.DisplayPath))
			} else {
				content.WriteString(fmt.Sprintf("Stash %d files\n", len(items)))
			}
		}
		content.WriteString(m.stashInput.View())
		content.WriteString(StyleMuted.Render("  (enter to confirm, esc to cancel)"))
	} else if m.commitMode {
		content.WriteString("Commit message: ")
		content.WriteString(m.commitInput.View())
		content.WriteString(StyleMuted.Render("  (enter to commit, esc to cancel)"))
	}

	// Show persistent help bar when in help mode
	if m.showVerboseHelp {
		content.WriteString("\n")
		content.WriteString(m.renderHelpBar())
	}

	return content.String()
}

func (m StatusModel) renderItem(index int, f git.FileStatus, section string) string {
	path := f.DisplayPath
	if f.OriginalDisplayPath != "" {
		path = fmt.Sprintf("%s → %s", f.OriginalDisplayPath, f.DisplayPath)
	}

	var pathStyle lipgloss.Style
	switch section {
	case "staged":
		pathStyle = StyleStaged
	case "unstaged":
		pathStyle = StyleUnstaged
	case "untracked":
		pathStyle = StyleUntracked
	}

	// When quitting, render without any cursor or selection highlighting
	if m.quitting {
		statusChar := StatusChar(f.IndexStatus, f.WorkStatus, section)
		return fmt.Sprintf("        %s%s", statusChar, pathStyle.Render(path))
	}

	isSelected := m.selected[index]
	isCursor := index == m.cursor

	prefix := "        "
	if isCursor {
		prefix = ">       "
	}

	// Apply visual mode highlight for selected items
	if isSelected {
		statusChar := StatusCharStyled(f.IndexStatus, f.WorkStatus, section, StyleVisual)
		return StyleVisual.Render(prefix) + statusChar + pathStyle.Inherit(StyleVisual).Render(path)
	}

	statusChar := StatusChar(f.IndexStatus, f.WorkStatus, section)
	return fmt.Sprintf("%s%s%s", prefix, statusChar, pathStyle.Render(path))
}

func (m StatusModel) renderHelp() string {
	var sb strings.Builder

	type column struct {
		title string
		items []struct{ key, desc string }
	}

	navKeys := formatKeyList(Keys.Down, Keys.Up)
	topBottomKeys := formatKeyList(formatDoubleKey(Keys.Top), Keys.Bottom)
	stageKeys := formatKeyList(Keys.Stage, Keys.StageAll)
	unstageKeys := formatKeyList(Keys.Unstage, Keys.UnstageAll)
	commitKeys := formatKeyList(Keys.Commit, Keys.CommitEdit)
	stashKeys := formatKeyList(Keys.Stash, Keys.StashAll)
	fileDiffKeys := formatKeyList(Keys.FileDiff, Keys.Right)
	quitKeys := formatKeyList(Keys.Quit, "ESC")

	columns := []column{
		{
			title: "Navigation",
			items: []struct{ key, desc string }{
				{navKeys, "up/down"},
				{topBottomKeys, "top/bottom"},
				{Keys.Select, "select"},
				{Keys.Visual, "visual"},
			},
		},
		{
			title: "Staging",
			items: []struct{ key, desc string }{
				{"SPACE", "toggle"},
				{stageKeys, "stage"},
				{unstageKeys, "unstage"},
				{Keys.Discard, "discard"},
			},
		},
		{
			title: "Actions",
			items: []struct{ key, desc string }{
				{commitKeys, "commit"},
				{Keys.Push, "push"},
				{stashKeys, "stash"},
			},
		},
		{
			title: "Views",
			items: []struct{ key, desc string }{
				{fileDiffKeys, "file diff"},
				{Keys.AllDiffs, "all diffs"},
				{Keys.Branches, "branches"},
				{Keys.Stashes, "stashes"},
				{Keys.Log, "log"},
			},
		},
		{
			title: "General",
			items: []struct{ key, desc string }{
				{Keys.Help, "help"},
				{Keys.VerboseHelp, "help mode"},
				{quitKeys, "quit"},
			},
		},
	}

	colWidth := 18

	// Find max rows
	maxRows := 0
	for _, col := range columns {
		if len(col.items) > maxRows {
			maxRows = len(col.items)
		}
	}

	// Render header row
	for _, col := range columns {
		title := col.title
		padding := colWidth - len(title)
		sb.WriteString(StyleSectionHeader.Render(title))
		sb.WriteString(strings.Repeat(" ", padding))
	}
	sb.WriteString("\n")

	// Render item rows
	for i := 0; i < maxRows; i++ {
		for _, col := range columns {
			if i < len(col.items) {
				item := col.items[i]
				cell := fmt.Sprintf("%-6s%s", item.key, item.desc)
				padding := colWidth - len(cell)
				sb.WriteString(StyleHelpKey.Render(fmt.Sprintf("%-6s", item.key)))
				sb.WriteString(StyleHelpDesc.Render(item.desc))
				if padding > 0 {
					sb.WriteString(strings.Repeat(" ", padding))
				}
			} else {
				sb.WriteString(strings.Repeat(" ", colWidth))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m StatusModel) renderHelpBar() string {
	var sb strings.Builder

	sb.WriteString(StyleMuted.Render("─────────────────────────────────────────────────────────────────────────────────"))
	sb.WriteString("\n")

	line1 := []struct{ key, desc string }{
		{formatKeyList(Keys.Down, Keys.Up), "navigate"},
		{"SPACE", "stage/unstage"},
		{formatKeyList(Keys.Stage, Keys.StageAll), "stage"},
		{formatKeyList(Keys.Unstage, Keys.UnstageAll), "unstage"},
		{Keys.Discard, "discard"},
		{formatKeyList(Keys.Commit, Keys.CommitEdit), "commit"},
		{Keys.Push, "push"},
	}

	line2 := []struct{ key, desc string }{
		{Keys.Select, "select"},
		{Keys.Visual, "visual"},
		{Keys.FileDiff, "diff"},
		{Keys.AllDiffs, "all diffs"},
		{Keys.Branches, "branches"},
		{Keys.Stashes, "stashes"},
		{Keys.Log, "log"},
		{Keys.VerboseHelp, "hide help"},
	}

	for _, item := range line1 {
		sb.WriteString(StyleHelpKey.Render(item.key))
		sb.WriteString(" ")
		sb.WriteString(StyleHelpDesc.Render(item.desc))
		sb.WriteString("  ")
	}
	sb.WriteString("\n")

	for _, item := range line2 {
		sb.WriteString(StyleHelpKey.Render(item.key))
		sb.WriteString(" ")
		sb.WriteString(StyleHelpDesc.Render(item.desc))
		sb.WriteString("  ")
	}

	return sb.String()
}

func runGitCommit() tea.Cmd {
	c := exec.Command("git", "commit")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return tea.Quit()
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

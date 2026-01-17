package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"simple-git/internal/git"

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
	selected        map[int]bool
	visualMode      bool
	visualStart     int
	status          *git.StatusResult
	branchStatus    git.BranchStatus
	showHelp        bool
	showVerboseHelp bool
	confirmMode     confirmAction
	stashMode       stashMode
	stashInput      textinput.Model
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
	return StatusModel{
		selected:        make(map[int]bool),
		stashInput:      ti,
		showVerboseHelp: showHelp,
	}
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
			if key == "?" || key == "esc" || key == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		// Handle confirm mode
		if m.confirmMode != confirmNone {
			switch key {
			case "y", "Y":
				action := m.confirmMode
				m.confirmMode = confirmNone
				if action == confirmDiscard {
					return m, m.doDiscard()
				} else if action == confirmPush {
					return m, m.doPush()
				}
				return m, nil
			case "n", "N", "esc":
				m.confirmMode = confirmNone
				return m, nil
			}
			return m, nil
		}

		// Handle stash input mode
		if m.stashMode != stashNone {
			switch key {
			case "enter":
				mode := m.stashMode
				message := m.stashInput.Value()
				m.stashMode = stashNone
				m.stashInput.Reset()
				m.stashInput.Blur()
				return m, m.doStash(mode, message)
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

		// Check for gg sequence
		if m.lastKey == "g" && key == "g" {
			m.lastKey = ""
			m.cursor = 0
			if m.visualMode {
				m.updateVisualSelection()
			}
			return m, nil
		}

		if key == "g" {
			m.lastKey = "g"
			return m, nil
		}
		m.lastKey = ""

		switch key {
		case "q":
			if m.visualMode {
				m.visualMode = false
				m.selected = make(map[int]bool)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.visualMode || len(m.selected) > 0 {
				m.visualMode = false
				m.selected = make(map[int]bool)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = true
			return m, nil
		case "/":
			m.showVerboseHelp = !m.showVerboseHelp
			return m, nil
		case "v":
			if m.visualMode {
				m.visualMode = false
				m.selected = make(map[int]bool)
			} else {
				m.visualMode = true
				m.visualStart = m.cursor
				m.selected = make(map[int]bool)
				m.selected[m.cursor] = true
			}
			return m, nil
		case "V":
			if !m.visualMode {
				m.visualMode = true
				m.visualStart = m.cursor
				m.selected = make(map[int]bool)
				m.selected[m.cursor] = true
			}
			return m, nil
		case "j", "down":
			if len(m.items) > 0 {
				m.cursor = min(m.cursor+1, len(m.items)-1)
				if m.visualMode {
					m.updateVisualSelection()
				}
			}
			return m, nil
		case "k", "up":
			if len(m.items) > 0 {
				m.cursor = max(m.cursor-1, 0)
				if m.visualMode {
					m.updateVisualSelection()
				}
			}
			return m, nil
		case "G":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				if m.visualMode {
					m.updateVisualSelection()
				}
			}
			return m, nil
		case "h", "left":
			// Toggle selection of current item (non-contiguous multi-select)
			if len(m.items) > 0 && !m.visualMode {
				if m.selected[m.cursor] {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = true
				}
			}
			return m, nil
		case " ":
			return m, m.toggleStage()
		case "a":
			return m, m.stageFiles()
		case "A":
			return m, m.stageAll()
		case "u":
			return m, m.unstageFiles()
		case "U":
			return m, m.unstageAll()
		case "d":
			if len(m.items) > 0 && (len(m.selected) > 0 || !m.visualMode) {
				m.confirmMode = confirmDiscard
			}
			return m, nil
		case "p":
			// Push with confirmation
			if m.branchStatus.Remote != "" && m.branchStatus.Ahead > 0 {
				m.confirmMode = confirmPush
			}
			return m, nil
		case "c":
			// Run git commit and quit when done
			m.quitting = true
			return m, runGitCommit()
		case "s":
			// Stash selected file(s)
			if len(m.items) > 0 {
				m.stashMode = stashFiles
				m.stashInput.Focus()
				return m, textinput.Blink
			}
			return m, nil
		case "S":
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
				if err = git.UnstageFile(item.File.Path); err != nil {
					return errMsg{err}
				}
				if item.File.IndexStatus != 'A' {
					err = git.DiscardFile(item.File.Path)
				}
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
		content.WriteString(StyleEmpty.Render("Nothing to commit, working tree clean"))
		content.WriteString("\n")
		if m.showVerboseHelp {
			content.WriteString("\n")
			content.WriteString(m.renderHelpBar())
		}
		return content.String()
	}

	// Branch status info
	content.WriteString(StyleMuted.Render(fmt.Sprintf("On branch %s", m.branchStatus.Name)))
	content.WriteString("\n")
	if m.branchStatus.Remote != "" {
		if m.branchStatus.Ahead > 0 && m.branchStatus.Behind > 0 {
			content.WriteString(StyleMuted.Render(fmt.Sprintf("Your branch and '%s' have diverged,", m.branchStatus.Remote)))
			content.WriteString("\n")
			content.WriteString(StyleMuted.Render(fmt.Sprintf("and have %d and %d different commits each, respectively.", m.branchStatus.Ahead, m.branchStatus.Behind)))
		} else if m.branchStatus.Ahead > 0 {
			if m.branchStatus.Ahead == 1 {
				content.WriteString(StyleMuted.Render(fmt.Sprintf("Your branch is ahead of '%s' by 1 commit.", m.branchStatus.Remote)))
			} else {
				content.WriteString(StyleMuted.Render(fmt.Sprintf("Your branch is ahead of '%s' by %d commits.", m.branchStatus.Remote, m.branchStatus.Ahead)))
			}
		} else if m.branchStatus.Behind > 0 {
			if m.branchStatus.Behind == 1 {
				content.WriteString(StyleMuted.Render(fmt.Sprintf("Your branch is behind '%s' by 1 commit.", m.branchStatus.Remote)))
			} else {
				content.WriteString(StyleMuted.Render(fmt.Sprintf("Your branch is behind '%s' by %d commits.", m.branchStatus.Remote, m.branchStatus.Behind)))
			}
		} else {
			content.WriteString(StyleMuted.Render(fmt.Sprintf("Your branch is up to date with '%s'.", m.branchStatus.Remote)))
		}
		content.WriteString("\n")
	}
	content.WriteString("\n")

	if m.visualMode && !m.quitting {
		content.WriteString(StyleVisual.Render("-- VISUAL --"))
		content.WriteString("\n")
	}

	itemIndex := 0

	if len(m.status.Staged) > 0 {
		content.WriteString(StyleSectionHeader.Render("Staged Changes"))
		content.WriteString("\n")
		for _, f := range m.status.Staged {
			content.WriteString(m.renderItem(itemIndex, f, "staged"))
			content.WriteString("\n")
			itemIndex++
		}
	}

	if len(m.status.Unstaged) > 0 {
		content.WriteString(StyleSectionHeader.Render("Unstaged Changes"))
		content.WriteString("\n")
		for _, f := range m.status.Unstaged {
			content.WriteString(m.renderItem(itemIndex, f, "unstaged"))
			content.WriteString("\n")
			itemIndex++
		}
	}

	if len(m.status.Untracked) > 0 {
		content.WriteString(StyleSectionHeader.Render("Untracked Files"))
		content.WriteString("\n")
		for _, f := range m.status.Untracked {
			content.WriteString(m.renderItem(itemIndex, f, "untracked"))
			content.WriteString("\n")
			itemIndex++
		}
	}

	// Confirm prompt (only shown when confirming)
	if m.confirmMode == confirmDiscard {
		items := m.getSelectedItems()
		if len(items) == 1 {
			content.WriteString(StyleConfirm.Render(fmt.Sprintf("Discard '%s'? (y/n) ", items[0].File.Path)))
		} else {
			content.WriteString(StyleConfirm.Render(fmt.Sprintf("Discard %d files? (y/n) ", len(items))))
		}
	} else if m.confirmMode == confirmPush {
		if m.branchStatus.Ahead == 1 {
			content.WriteString(fmt.Sprintf("Push 1 commit to '%s'? (y/n) ", m.branchStatus.Remote))
		} else {
			content.WriteString(fmt.Sprintf("Push %d commits to '%s'? (y/n) ", m.branchStatus.Ahead, m.branchStatus.Remote))
		}
	} else if m.stashMode != stashNone {
		if m.stashMode == stashAll {
			content.WriteString("Stash all changes\n")
		} else {
			items := m.getSelectedItems()
			if len(items) == 1 {
				content.WriteString(fmt.Sprintf("Stash '%s'\n", items[0].File.Path))
			} else {
				content.WriteString(fmt.Sprintf("Stash %d files\n", len(items)))
			}
		}
		content.WriteString(m.stashInput.View())
		content.WriteString(StyleMuted.Render("  (enter to confirm, esc to cancel)"))
	}

	// Show persistent help bar when in help mode
	if m.showVerboseHelp {
		content.WriteString("\n")
		content.WriteString(m.renderHelpBar())
	}

	return content.String()
}

func (m StatusModel) renderItem(index int, f git.FileStatus, section string) string {
	path := f.Path
	if f.OriginalPath != "" {
		path = fmt.Sprintf("%s → %s", f.OriginalPath, f.Path)
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
		statusChar := StatusChar(f.IndexStatus, f.WorkStatus)
		return fmt.Sprintf("  %s %s", statusChar, pathStyle.Render(path))
	}

	isSelected := m.selected[index]
	isCursor := index == m.cursor

	prefix := "  "
	if isCursor {
		prefix = "> "
	}

	if !isSelected && !isCursor {
		statusChar := StatusChar(f.IndexStatus, f.WorkStatus)
		return fmt.Sprintf("%s%s %s", prefix, statusChar, pathStyle.Render(path))
	}

	highlight := StyleSelected
	if isSelected {
		highlight = StyleVisual
	}

	statusChar := StatusCharStyled(f.IndexStatus, f.WorkStatus, highlight)
	gap := highlight.Render(" ")

	return highlight.Render(prefix) + statusChar + gap + pathStyle.Inherit(highlight).Render(path)
}

func (m StatusModel) renderHelp() string {
	var sb strings.Builder

	type column struct {
		title string
		items []struct{ key, desc string }
	}

	columns := []column{
		{
			title: "Navigation",
			items: []struct{ key, desc string }{
				{"j/k", "up/down"},
				{"gg/G", "top/bottom"},
				{"h", "select"},
				{"v", "visual"},
			},
		},
		{
			title: "Staging",
			items: []struct{ key, desc string }{
				{"SPACE", "toggle"},
				{"a/A", "stage"},
				{"u/U", "unstage"},
				{"d", "discard"},
			},
		},
		{
			title: "Actions",
			items: []struct{ key, desc string }{
				{"c", "commit"},
				{"p", "push"},
				{"s/S", "stash"},
			},
		},
		{
			title: "Views",
			items: []struct{ key, desc string }{
				{"l", "file diff"},
				{"i", "all diffs"},
				{"b", "branches"},
				{"e", "stashes"},
				{"o", "log"},
			},
		},
		{
			title: "General",
			items: []struct{ key, desc string }{
				{"?", "help"},
				{"/", "help mode"},
				{"q/ESC", "quit"},
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
		{"j/k", "navigate"},
		{"SPACE", "stage/unstage"},
		{"a/A", "stage"},
		{"u/U", "unstage"},
		{"d", "discard"},
		{"c", "commit"},
		{"p", "push"},
	}

	line2 := []struct{ key, desc string }{
		{"h", "select"},
		{"v", "visual"},
		{"l", "diff"},
		{"i", "all diffs"},
		{"b", "branches"},
		{"e", "stashes"},
		{"o", "log"},
		{"/", "hide help"},
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

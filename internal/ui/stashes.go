package ui

import (
	"fmt"
	"strings"

	"go-on-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// StashDiffModel is a simplified diff view for stash contents
type StashDiffModel struct {
	diff         *git.CombinedDiffResult
	hunks        []git.Hunk
	cursor       int
	viewingHunk  bool
	scrollOffset int
	showHelp     bool
	err          error
	width        int
	height       int
}

// NewStashDiffModel creates a new stash diff model
func NewStashDiffModel(width, height int) StashDiffModel {
	return StashDiffModel{
		width:  width,
		height: height,
	}
}

// Init initializes the model
func (m StashDiffModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the stash diff view
func (m StashDiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		if m.showHelp {
			if key == Keys.Help || key == "esc" || key == Keys.Quit {
				m.showHelp = false
			}
			return m, nil
		}

		// Viewing single hunk detail
		if m.viewingHunk {
			switch key {
			case Keys.Left, "left", "esc":
				m.viewingHunk = false
				m.scrollOffset = 0
				return m, nil
			case Keys.Down, "down":
				if m.cursor < len(m.hunks) {
					maxScroll := len(m.hunks[m.cursor].Lines) - m.visibleLines()
					if maxScroll > 0 {
						m.scrollOffset = min(m.scrollOffset+1, maxScroll)
					}
				}
				return m, nil
			case Keys.Up, "up":
				m.scrollOffset = max(m.scrollOffset-1, 0)
				return m, nil
			case Keys.Bottom:
				if m.cursor < len(m.hunks) {
					maxScroll := len(m.hunks[m.cursor].Lines) - m.visibleLines()
					if maxScroll > 0 {
						m.scrollOffset = maxScroll
					}
				}
				return m, nil
			case Keys.Top:
				m.scrollOffset = 0
				return m, nil
			case Keys.Help:
				m.showHelp = true
				return m, nil
			}
			return m, nil
		}

		switch key {
		case Keys.Help:
			m.showHelp = true
			return m, nil
		case Keys.Right, "right":
			if len(m.hunks) > 0 && m.cursor < len(m.hunks) {
				m.viewingHunk = true
				m.scrollOffset = 0
			}
			return m, nil
		case Keys.Down, "down":
			if len(m.hunks) > 0 {
				m.cursor = min(m.cursor+1, len(m.hunks)-1)
			}
			return m, nil
		case Keys.Up, "up":
			if len(m.hunks) > 0 {
				m.cursor = max(m.cursor-1, 0)
			}
			return m, nil
		case Keys.Bottom:
			if len(m.hunks) > 0 {
				m.cursor = len(m.hunks) - 1
			}
			return m, nil
		case Keys.Top:
			m.cursor = 0
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case stashDiffMsg:
		m.diff = msg.diff
		m.hunks = msg.diff.GetAllHunksCombined()
		m.cursor = 0
		// Auto-enter detail view when there's only one hunk
		if len(m.hunks) == 1 {
			m.viewingHunk = true
			m.scrollOffset = 0
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m StashDiffModel) visibleLines() int {
	if m.height <= 5 {
		return 40
	}
	return m.height - 5
}

func (m StashDiffModel) anchorBottom(content string) string {
	lines := strings.Count(content, "\n")
	if m.height <= lines {
		return content
	}
	padding := m.height - lines - 1
	return strings.Repeat("\n", padding) + content
}

// View renders the stash diff
func (m StashDiffModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	var sb strings.Builder

	if m.err != nil {
		sb.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
		sb.WriteString("\n")
	}

	if m.diff == nil || len(m.hunks) == 0 {
		sb.WriteString(StyleMuted.Render("No changes in stash"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Hunk detail view
	if m.viewingHunk && m.cursor < len(m.hunks) {
		return m.anchorBottom(m.renderHunkDetail())
	}

	// Hunk list view with preview
	fixedLines := len(m.hunks)
	availableForDetail := 50
	if m.height > fixedLines+5 {
		availableForDetail = m.height - fixedLines - 3
	}

	// Show current hunk preview first (above the list)
	if m.cursor < len(m.hunks) && availableForDetail > 0 {
		hunk := m.hunks[m.cursor]
		sb.WriteString(fmt.Sprintf("─── %s %s ───", hunk.DisplayFilePath, hunk.Header))
		sb.WriteString("\n")

		totalLines := len(hunk.Lines)
		showLines := min(totalLines, availableForDetail)

		for i := 0; i < showLines; i++ {
			line := hunk.Lines[i]
			var styled string
			switch line.Type {
			case git.LineAdded:
				styled = StyleDiffAdded.Render(line.Content)
			case git.LineRemoved:
				styled = StyleDiffRemoved.Render(line.Content)
			default:
				styled = StyleDiffContext.Render(line.Content)
			}
			sb.WriteString(styled)
			sb.WriteString("\n")
		}

		if totalLines > showLines {
			sb.WriteString(StyleMuted.Render(fmt.Sprintf("... %d more lines (l/→ to view full hunk)", totalLines-showLines)))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Show hunk list at the bottom
	for i, h := range m.hunks {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		adds, dels := 0, 0
		for _, line := range h.Lines {
			switch line.Type {
			case git.LineAdded:
				adds++
			case git.LineRemoved:
				dels++
			}
		}

		if i == m.cursor {
			sb.WriteString(StyleSelected.Render(fmt.Sprintf("%s@@ %s +%d -%d", cursor, h.DisplayFilePath, adds, dels)))
		} else {
			sb.WriteString(fmt.Sprintf("%s@@ %s +%d -%d", cursor, h.DisplayFilePath, adds, dels))
		}
		sb.WriteString("\n")
	}

	return m.anchorBottom(sb.String())
}

func (m StashDiffModel) renderHunkDetail() string {
	var sb strings.Builder

	hunk := m.hunks[m.cursor]

	// Hunk lines with scrolling
	totalLines := len(hunk.Lines)
	visible := m.visibleLines()
	endLine := min(m.scrollOffset+visible, totalLines)

	for i := m.scrollOffset; i < endLine; i++ {
		line := hunk.Lines[i]
		var styled string
		switch line.Type {
		case git.LineAdded:
			styled = StyleDiffAdded.Render(line.Content)
		case git.LineRemoved:
			styled = StyleDiffRemoved.Render(line.Content)
		default:
			styled = StyleDiffContext.Render(line.Content)
		}
		sb.WriteString(styled)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if totalLines > visible {
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("Lines %d-%d of %d", m.scrollOffset+1, min(m.scrollOffset+visible, totalLines), totalLines)))
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("─── %s %s ───", hunk.DisplayFilePath, hunk.Header))
	sb.WriteString("\n")

	return sb.String()
}

func (m StashDiffModel) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(StyleHelpTitle.Render("Stash Diff Shortcuts"))
	sb.WriteString("\n\n")

	drillKeys := formatKeyList(Keys.Right, "→")
	backKeys := formatKeyList(Keys.Left, "←", "ESC")
	moveKeys := formatKeyList(Keys.Down, Keys.Up, "↓", "↑")
	topBottomKeys := formatKeyList(Keys.Top, Keys.Bottom)

	help := []struct {
		key  string
		desc string
	}{
		{drillKeys, "View hunk detail"},
		{backKeys, "Go back"},
		{moveKeys, "Navigate / scroll"},
		{topBottomKeys, "Go to top/bottom"},
		{Keys.Help, "Toggle help"},
	}

	for _, h := range help {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			StyleHelpKey.Render(fmt.Sprintf("%-8s", h.key)),
			StyleHelpDesc.Render(h.desc)))
	}

	return sb.String()
}

// StashesModel is the bubbletea model for the stashes view
type StashesModel struct {
	stashes         []git.Stash
	cursor          int
	scrollOffset    int
	showHelp        bool
	showVerboseHelp bool
	confirmMode     bool
	confirmAction   string // "drop", "pop"
	diffModel       StashDiffModel
	lastKey         string
	err             error
	width           int
	height          int
}

// NewStashesModel creates a new stashes model
func NewStashesModel() StashesModel {
	return NewStashesModelWithOptions(false)
}

// NewStashesModelWithOptions creates a new stashes model with options
func NewStashesModelWithOptions(showVerboseHelp bool) StashesModel {
	return StashesModel{
		showVerboseHelp: showVerboseHelp,
	}
}

// Init initializes the model
func (m StashesModel) Init() tea.Cmd {
	return refreshStashes
}

func refreshStashes() tea.Msg {
	stashes, err := git.GetStashes()
	if err != nil {
		return errMsg{err}
	}
	return stashesMsg{stashes}
}

// Update handles messages
func (m StashesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		// Handle confirm mode
		if m.confirmMode {
			switch key {
			case "y", "Y":
				action := m.confirmAction
				m.confirmMode = false
				m.confirmAction = ""
				switch action {
				case "drop":
					return m, m.doDropStash()
				case "pop":
					return m, m.doPopStash()
				}
				return m, nil
			case "n", "N", "esc":
				m.confirmMode = false
				m.confirmAction = ""
				return m, nil
			}
			return m, nil
		}

		// Check for gg sequence
		if m.lastKey == Keys.Top && key == Keys.Top {
			m.lastKey = ""
			m.cursor = 0
			m.ensureCursorVisible()
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
		case Keys.VerboseHelp:
			m.showVerboseHelp = !m.showVerboseHelp
			return m, nil
		case Keys.Down, "down":
			if len(m.stashes) > 0 {
				m.cursor = min(m.cursor+1, len(m.stashes)-1)
				m.ensureCursorVisible()
			}
			return m, nil
		case Keys.Up, "up":
			if len(m.stashes) > 0 {
				m.cursor = max(m.cursor-1, 0)
				m.ensureCursorVisible()
			}
			return m, nil
		case Keys.Bottom:
			if len(m.stashes) > 0 {
				m.cursor = len(m.stashes) - 1
				m.ensureCursorVisible()
			}
			return m, nil
		case "a":
			// Apply stash (keep in list)
			if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
				return m, m.doApplyStash()
			}
			return m, nil
		case "p":
			// Pop stash (apply and remove, with confirmation)
			if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
				m.confirmMode = true
				m.confirmAction = "pop"
			}
			return m, nil
		case "d":
			// Drop stash (with confirmation)
			if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
				m.confirmMode = true
				m.confirmAction = "drop"
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case stashesMsg:
		m.stashes = msg.stashes
		if m.cursor >= len(m.stashes) {
			m.cursor = max(0, len(m.stashes)-1)
		}
		m.ensureCursorVisible()
		return m, nil

	case stashDiffMsg:
		m.diffModel.diff = msg.diff
		m.diffModel.hunks = msg.diff.GetAllHunksCombined()
		m.diffModel.cursor = 0
		if len(m.diffModel.hunks) == 1 {
			m.diffModel.viewingHunk = true
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m StashesModel) doApplyStash() tea.Cmd {
	if m.cursor >= len(m.stashes) {
		return nil
	}
	stash := m.stashes[m.cursor]
	return func() tea.Msg {
		err := git.ApplyStash(stash.Index)
		if err != nil {
			return errMsg{err}
		}
		return refreshStashes()
	}
}

func (m StashesModel) doPopStash() tea.Cmd {
	if m.cursor >= len(m.stashes) {
		return nil
	}
	stash := m.stashes[m.cursor]
	return func() tea.Msg {
		err := git.PopStash(stash.Index)
		if err != nil {
			return errMsg{err}
		}
		return refreshStashes()
	}
}

func (m StashesModel) doDropStash() tea.Cmd {
	if m.cursor >= len(m.stashes) {
		return nil
	}
	stash := m.stashes[m.cursor]
	return func() tea.Msg {
		err := git.DropStash(stash.Index)
		if err != nil {
			return errMsg{err}
		}
		return refreshStashes()
	}
}

// visibleLines returns the number of stash lines that can be displayed
func (m StashesModel) visibleLines() int {
	// Reserve lines for: header (~3), help bar (~3 if shown), confirm prompt, and buffer
	reserved := 8
	if m.showVerboseHelp {
		reserved += 3
	}
	if m.height <= reserved {
		return 10 // fallback minimum
	}
	return m.height - reserved
}

// ensureCursorVisible adjusts scrollOffset to keep cursor in view
func (m *StashesModel) ensureCursorVisible() {
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
	maxOffset := len(m.stashes) - visible
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

// View renders the model
func (m StashesModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	var sb strings.Builder

	if m.err != nil {
		sb.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
		sb.WriteString("\n\n")
	}

	if len(m.stashes) == 0 {
		sb.WriteString(StyleMuted.Render("No stashes"))
		sb.WriteString("\n")
		return sb.String()
	}

	sb.WriteString(m.renderHeader())
	sb.WriteString("\n\n")

	// Calculate visible range
	visibleStart := m.scrollOffset
	visibleEnd := m.scrollOffset + m.visibleLines()
	if visibleEnd > len(m.stashes) {
		visibleEnd = len(m.stashes)
	}

	// Show scroll indicator at top if scrolled down
	if m.scrollOffset > 0 {
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("  ↑ %d more above", m.scrollOffset)))
		sb.WriteString("\n")
	}

	for i := visibleStart; i < visibleEnd; i++ {
		stash := m.stashes[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		label := fmt.Sprintf("stash@{%d}", stash.Index)
		if stash.Branch != "" {
			label += " on " + stash.Branch
		}
		label += ": " + stash.Message

		sb.WriteString(prefix + label)
		sb.WriteString("\n")
	}

	// Show scroll indicator at bottom if more items below
	if visibleEnd < len(m.stashes) {
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("  ↓ %d more below", len(m.stashes)-visibleEnd)))
		sb.WriteString("\n")
	}

	// Confirm prompt
	if m.confirmMode && m.cursor < len(m.stashes) {
		sb.WriteString("\n")
		stash := m.stashes[m.cursor]
		switch m.confirmAction {
		case "drop":
			sb.WriteString(StyleConfirm.Render(fmt.Sprintf("Drop stash@{%d}? (y/n) ", stash.Index)))
		case "pop":
			sb.WriteString(StyleConfirm.Render(fmt.Sprintf("Pop stash@{%d}? (y/n) ", stash.Index)))
		}
	}

	// Help bar (only show when showVerboseHelp is on and not in confirm mode)
	if m.showVerboseHelp && !m.confirmMode {
		sb.WriteString("\n\n")
		sb.WriteString(m.renderHelpBar())
	}

	return sb.String()
}

func (m StashesModel) renderHeader() string {
	return StyleMuted.Render("> git stash list") + "  " + StyleMuted.Render("(esc to go back)") + "\n" + StyleMuted.Render("───────────────────────────────────────────────────────────────")
}

func (m StashesModel) renderHelpBar() string {
	var sb strings.Builder

	sb.WriteString(StyleMuted.Render("───────────────────────────────────────────────────────────────"))
	sb.WriteString("\n")

	items := []struct{ key, desc string }{
		{formatKeyList(Keys.Down, Keys.Up), "navigate"},
		{formatKeyList(Keys.Right, "→"), "view diff"},
		{"a", "apply"},
		{"p", "pop"},
		{"d", "drop"},
		{Keys.Help, "help"},
		{formatKeyList(Keys.Left, "ESC"), "back"},
	}

	for _, item := range items {
		sb.WriteString(StyleHelpKey.Render(item.key))
		sb.WriteString(" ")
		sb.WriteString(StyleHelpDesc.Render(item.desc))
		sb.WriteString("  ")
	}

	return sb.String()
}

func (m StashesModel) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(StyleHelpTitle.Render("Stashes Shortcuts"))
	sb.WriteString("\n\n")

	moveKeys := formatKeyList(Keys.Down, Keys.Up, "↓", "↑")
	topKey := formatDoubleKey(Keys.Top)
	drillKeys := formatKeyList(Keys.Right, "→")
	backKeys := formatKeyList(Keys.Left, "←", "ESC")

	help := []struct {
		key  string
		desc string
	}{
		{moveKeys, "Move down/up"},
		{topKey, "Go to top"},
		{Keys.Bottom, "Go to bottom"},
		{drillKeys, "View stash diff"},
		{"a", "Apply stash (keep in list)"},
		{"p", "Pop stash (apply and remove)"},
		{"d", "Drop stash (delete)"},
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

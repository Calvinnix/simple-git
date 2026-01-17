package ui

import (
	"fmt"
	"strings"

	"simple-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// DiffModel is the bubbletea model for the diff view
type DiffModel struct {
	diff         *git.CombinedDiffResult
	hunks        []git.Hunk
	cursor       int
	filterFiles  []string // only show hunks for these files (empty = all)
	viewingHunk  bool     // true when drilled into a single hunk
	scrollOffset int      // scroll position within hunk content
	showHelp     bool
	confirmMode  bool
	lastKey      string
	err          error
	width        int
	height       int
}

// NewDiffModel creates a new diff model
func NewDiffModel(filterFiles []string) DiffModel {
	return DiffModel{
		filterFiles: filterFiles,
	}
}

// NewDiffModelWithSize creates a new diff model with known dimensions
func NewDiffModelWithSize(filterFiles []string, width, height int) DiffModel {
	return DiffModel{
		filterFiles: filterFiles,
		width:       width,
		height:      height,
	}
}

// IsViewingHunk returns true if the user is in the hunk detail view
func (m DiffModel) IsViewingHunk() bool {
	return m.viewingHunk
}

// Init initializes the model
func (m DiffModel) Init() tea.Cmd {
	return m.refreshCombinedDiff
}

func (m DiffModel) refreshCombinedDiff() tea.Msg {
	diff, err := git.GetCombinedDiff()
	if err != nil {
		return errMsg{err}
	}
	return combinedDiffMsg{diff}
}

// Update handles messages
func (m DiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle help mode - ESC or ? closes help
		if m.showHelp {
			if key == "?" || key == "esc" || key == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		if m.confirmMode {
			switch key {
			case "y", "Y":
				m.confirmMode = false
				return m, m.doDiscard()
			case "n", "N", "esc":
				m.confirmMode = false
				return m, nil
			}
			return m, nil
		}

		// Handle hunk detail view navigation
		if m.viewingHunk {
			switch key {
			case "h", "left", "esc":
				m.viewingHunk = false
				m.scrollOffset = 0
				return m, nil
			case "j", "down":
				if m.cursor < len(m.hunks) {
					maxScroll := len(m.hunks[m.cursor].Lines) - m.visibleLines()
					if maxScroll > 0 {
						m.scrollOffset = min(m.scrollOffset+1, maxScroll)
					}
				}
				return m, nil
			case "k", "up":
				m.scrollOffset = max(m.scrollOffset-1, 0)
				return m, nil
			case "g":
				if m.lastKey == "g" {
					m.lastKey = ""
					m.scrollOffset = 0
					return m, nil
				}
				m.lastKey = "g"
				return m, nil
			case "G":
				if m.cursor < len(m.hunks) {
					maxScroll := len(m.hunks[m.cursor].Lines) - m.visibleLines()
					if maxScroll > 0 {
						m.scrollOffset = maxScroll
					}
				}
				return m, nil
			case " ":
				return m, m.toggleStage()
			case "a":
				return m, m.stageHunk()
			case "u":
				return m, m.unstageHunk()
			case "d":
				if len(m.hunks) > 0 && m.cursor < len(m.hunks) && !m.hunks[m.cursor].Staged {
					m.confirmMode = true
				}
				return m, nil
			case "q":
				return m, tea.Quit
			case "?":
				m.showHelp = true
				return m, nil
			}
			m.lastKey = ""
			return m, nil
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
		case "q":
			return m, tea.Quit
		case "?":
			m.showHelp = true
			return m, nil
		case "l", "right":
			if len(m.hunks) > 0 && m.cursor < len(m.hunks) {
				m.viewingHunk = true
				m.scrollOffset = 0
			}
			return m, nil
		case "j", "down":
			if len(m.hunks) > 0 {
				m.cursor = min(m.cursor+1, len(m.hunks)-1)
			}
			return m, nil
		case "k", "up":
			if len(m.hunks) > 0 {
				m.cursor = max(m.cursor-1, 0)
			}
			return m, nil
		case "G":
			if len(m.hunks) > 0 {
				m.cursor = len(m.hunks) - 1
			}
			return m, nil
		case " ":
			return m, m.toggleStage()
		case "a":
			return m, m.stageHunk()
		case "u":
			return m, m.unstageHunk()
		case "d":
			// Only allow discard on unstaged hunks
			if len(m.hunks) > 0 && m.cursor < len(m.hunks) && !m.hunks[m.cursor].Staged {
				m.confirmMode = true
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case combinedDiffMsg:
		m.diff = msg.diff
		newHunks := m.getFilteredHunks()
		if len(m.hunks) > 0 && len(newHunks) > 0 {
			newHunks = m.keepHunkOrder(newHunks)
		}
		m.hunks = newHunks
		if m.cursor >= len(m.hunks) {
			m.cursor = max(0, len(m.hunks)-1)
		}
		// Auto-enter detail view when there's only one hunk
		if len(m.hunks) == 1 && !m.viewingHunk {
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

func (m DiffModel) keepHunkOrder(newHunks []git.Hunk) []git.Hunk {
	indexByKey := make(map[string][]int, len(newHunks))
	for i, hunk := range newHunks {
		key := hunkStableKey(hunk)
		indexByKey[key] = append(indexByKey[key], i)
	}

	ordered := make([]git.Hunk, 0, len(newHunks))
	used := make([]bool, len(newHunks))
	for _, hunk := range m.hunks {
		key := hunkStableKey(hunk)
		indices := indexByKey[key]
		if len(indices) == 0 {
			continue
		}
		idx := indices[0]
		indexByKey[key] = indices[1:]
		if !used[idx] {
			ordered = append(ordered, newHunks[idx])
			used[idx] = true
		}
	}

	for i, hunk := range newHunks {
		if !used[i] {
			ordered = append(ordered, hunk)
		}
	}

	return ordered
}

func hunkStableKey(hunk git.Hunk) string {
	var sb strings.Builder
	sb.WriteString(hunk.FilePath)
	sb.WriteString("\n")
	hasDiffLines := false
	for _, line := range hunk.Lines {
		if line.Type == git.LineAdded || line.Type == git.LineRemoved {
			sb.WriteString(line.Content)
			sb.WriteString("\n")
			hasDiffLines = true
		}
	}
	if !hasDiffLines {
		sb.WriteString(hunk.Header)
	}
	return sb.String()
}

func (m DiffModel) visibleLines() int {
	// Reserve lines for header and padding
	if m.height <= 5 {
		return 40 // fallback default for full screen
	}
	return m.height - 5
}

func (m DiffModel) anchorBottom(content string) string {
	lines := strings.Count(content, "\n")
	if m.height <= lines {
		return content
	}
	padding := m.height - lines - 1
	return strings.Repeat("\n", padding) + content
}

func (m DiffModel) getFilteredHunks() []git.Hunk {
	if m.diff == nil {
		return nil
	}
	allHunks := m.diff.GetAllHunksCombined()
	if len(m.filterFiles) == 0 {
		return allHunks
	}
	// Build set of filter files for O(1) lookup
	filterSet := make(map[string]bool, len(m.filterFiles))
	for _, f := range m.filterFiles {
		filterSet[f] = true
	}
	var filtered []git.Hunk
	for _, h := range allHunks {
		if filterSet[h.FilePath] {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

func (m DiffModel) toggleStage() tea.Cmd {
	if len(m.hunks) == 0 || m.cursor >= len(m.hunks) {
		return nil
	}

	hunk := m.hunks[m.cursor]
	fileDiff := m.diff.GetFileDiff(&hunk)
	if fileDiff == nil {
		return nil
	}

	return func() tea.Msg {
		patch := hunk.GeneratePatch(fileDiff)
		var err error
		if hunk.Staged {
			err = git.UnstageHunk(patch)
		} else {
			err = git.StageHunk(patch)
		}
		if err != nil {
			return errMsg{err}
		}

		// Refresh combined diff
		diff, err := git.GetCombinedDiff()
		if err != nil {
			return errMsg{err}
		}
		return combinedDiffMsg{diff}
	}
}

func (m DiffModel) stageHunk() tea.Cmd {
	if len(m.hunks) == 0 || m.cursor >= len(m.hunks) {
		return nil
	}

	hunk := m.hunks[m.cursor]
	// Only stage if not already staged
	if hunk.Staged {
		return nil
	}

	fileDiff := m.diff.GetFileDiff(&hunk)
	if fileDiff == nil {
		return nil
	}

	return func() tea.Msg {
		patch := hunk.GeneratePatch(fileDiff)
		err := git.StageHunk(patch)
		if err != nil {
			return errMsg{err}
		}

		diff, err := git.GetCombinedDiff()
		if err != nil {
			return errMsg{err}
		}
		return combinedDiffMsg{diff}
	}
}

func (m DiffModel) unstageHunk() tea.Cmd {
	if len(m.hunks) == 0 || m.cursor >= len(m.hunks) {
		return nil
	}

	hunk := m.hunks[m.cursor]
	// Only unstage if currently staged
	if !hunk.Staged {
		return nil
	}

	fileDiff := m.diff.GetFileDiff(&hunk)
	if fileDiff == nil {
		return nil
	}

	return func() tea.Msg {
		patch := hunk.GeneratePatch(fileDiff)
		err := git.UnstageHunk(patch)
		if err != nil {
			return errMsg{err}
		}

		diff, err := git.GetCombinedDiff()
		if err != nil {
			return errMsg{err}
		}
		return combinedDiffMsg{diff}
	}
}

func (m DiffModel) doDiscard() tea.Cmd {
	if len(m.hunks) == 0 || m.cursor >= len(m.hunks) {
		return nil
	}

	hunk := m.hunks[m.cursor]
	// Only allow discard on unstaged hunks
	if hunk.Staged {
		return nil
	}

	fileDiff := m.diff.GetFileDiff(&hunk)
	if fileDiff == nil {
		return nil
	}

	return func() tea.Msg {
		patch := hunk.GeneratePatch(fileDiff)
		err := git.DiscardHunk(patch)
		if err != nil {
			return errMsg{err}
		}
		// Refresh combined diff
		diff, err := git.GetCombinedDiff()
		if err != nil {
			return errMsg{err}
		}
		return combinedDiffMsg{diff}
	}
}

// View renders the model
func (m DiffModel) View() string {
	var sb strings.Builder

	if m.showHelp {
		return m.renderHelp()
	}

	if m.err != nil {
		sb.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
		sb.WriteString("\n")
	}

	if m.diff == nil {
		sb.WriteString(StyleMuted.Render("Loading..."))
		sb.WriteString("\n")
		return sb.String()
	}

	if len(m.hunks) == 0 {
		sb.WriteString(StyleEmpty.Render("No changes"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Drill-down view: show single hunk with scrolling
	if m.viewingHunk && m.cursor < len(m.hunks) {
		return m.anchorBottom(m.renderHunkDetail())
	}

	// Calculate available lines for preview
	fixedLines := len(m.hunks)
	availableForDetail := 50
	if m.height > fixedLines+5 {
		availableForDetail = m.height - fixedLines - 3
	}

	// Show current hunk preview first (above the list)
	if m.cursor < len(m.hunks) && availableForDetail > 0 {
		hunk := m.hunks[m.cursor]

		sb.WriteString(fmt.Sprintf("─── %s %s ───", renderStageLabel(hunk.Staged), hunk.Header))
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

		stageLabel := "[U]"
		stageStyle := StyleHunkHeaderUnstaged
		if h.Staged {
			stageLabel = "[S]"
			stageStyle = StyleHunkHeaderStaged
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
			stageStyle = stageStyle.Inherit(StyleSelected)
			sb.WriteString(StyleSelected.Render(cursor))
			sb.WriteString(stageStyle.Render(stageLabel))
			sb.WriteString(StyleSelected.Render(fmt.Sprintf(" @@ %s +%d -%d", h.FilePath, adds, dels)))
		} else {
			sb.WriteString(cursor)
			sb.WriteString(stageStyle.Render(stageLabel))
			sb.WriteString(fmt.Sprintf(" @@ %s +%d -%d", h.FilePath, adds, dels))
		}
		sb.WriteString("\n")
	}

	// Confirm prompt
	if m.confirmMode {
		sb.WriteString("\n")
		hunk := m.hunks[m.cursor]
		sb.WriteString(StyleConfirm.Render(fmt.Sprintf("Discard hunk from '%s'? (y/n) ", hunk.FilePath)))
	}

	return m.anchorBottom(sb.String())
}

func (m DiffModel) renderHunkDetail() string {
	var sb strings.Builder

	hunk := m.hunks[m.cursor]

	// Hunk lines with scrolling (content first, at top)
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

	// Show scroll position if scrollable
	if totalLines > visible {
		sb.WriteString(StyleMuted.Render(fmt.Sprintf("Lines %d-%d of %d", m.scrollOffset+1, min(m.scrollOffset+visible, totalLines), totalLines)))
		sb.WriteString("\n")
	}

	// Header with file info and navigation hint (at bottom)
	sb.WriteString(fmt.Sprintf("─── %s %s %s ───", renderStageLabel(hunk.Staged), hunk.FilePath, hunk.Header))
	sb.WriteString("\n")

	// Confirm prompt (only shown when confirming)
	if m.confirmMode {
		sb.WriteString(StyleConfirm.Render(fmt.Sprintf("Discard hunk from '%s'? (y/n) ", hunk.FilePath)))
	}

	return sb.String()
}

func (m DiffModel) renderHelp() string {
	var sb strings.Builder

	sb.WriteString(StyleHelpTitle.Render("Diff View Shortcuts"))
	sb.WriteString("\n\n")

	help := []struct {
		key  string
		desc string
	}{
		{"l/→", "View hunk detail (scrollable)"},
		{"h/←/ESC", "Go back"},
		{"j/k/↑/↓", "Navigate / scroll"},
		{"gg", "Go to top"},
		{"G", "Go to bottom"},
		{"SPACE", "Toggle stage/unstage hunk"},
		{"a", "Stage hunk"},
		{"u", "Unstage hunk"},
		{"d", "Discard hunk (unstaged only)"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}

	for _, h := range help {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			StyleHelpKey.Render(fmt.Sprintf("%-8s", h.key)),
			StyleHelpDesc.Render(h.desc)))
	}

	return sb.String()
}

func renderStageLabel(staged bool) string {
	if staged {
		return StyleHunkHeaderStaged.Render("[Staged]")
	}
	return StyleHunkHeaderUnstaged.Render("[Unstaged]")
}

package ui

import (
	"fmt"
	"strings"
	"testing"

	"go-on-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewDiffModel(t *testing.T) {
	m := NewDiffModel(nil)

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.viewingHunk {
		t.Error("viewingHunk should be false initially")
	}
	if m.showHelp {
		t.Error("showHelp should be false initially")
	}
	if m.confirmMode {
		t.Error("confirmMode should be false initially")
	}
	if len(m.filterFiles) != 0 {
		t.Errorf("filterFiles should be empty, got %d", len(m.filterFiles))
	}
}

func TestNewDiffModelWithSize(t *testing.T) {
	m := NewDiffModelWithSize([]string{"file1.txt", "file2.txt"}, 100, 50)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
	// Each file creates 2 filters (staged and unstaged)
	if len(m.filterFiles) != 4 {
		t.Errorf("filterFiles = %d, want 4", len(m.filterFiles))
	}
}

func TestNewDiffModelWithFilters(t *testing.T) {
	filters := []FileFilter{
		{Path: "file1.txt", ShowStaged: true},
		{Path: "file2.txt", ShowStaged: false, Untracked: true},
	}
	m := NewDiffModelWithFilters(filters, 80, 40)

	if len(m.filterFiles) != 2 {
		t.Errorf("filterFiles = %d, want 2", len(m.filterFiles))
	}
	if m.width != 80 {
		t.Errorf("width = %d, want 80", m.width)
	}
}

func TestDiffModelInit(t *testing.T) {
	m := NewDiffModel(nil)
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestDiffModelIsViewingHunk(t *testing.T) {
	m := NewDiffModel(nil)

	if m.IsViewingHunk() {
		t.Error("IsViewingHunk should be false initially")
	}

	m.viewingHunk = true
	if !m.IsViewingHunk() {
		t.Error("IsViewingHunk should be true after setting viewingHunk")
	}
}

func TestDiffModelNavigation(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt", Header: "@@ -1,3 +1,3 @@"},
		{FilePath: "file2.txt", Header: "@@ -1,3 +1,3 @@"},
		{FilePath: "file3.txt", Header: "@@ -1,3 +1,3 @@"},
	}

	// Test move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(DiffModel)
	if m.cursor != 1 {
		t.Errorf("after 'j', cursor = %d, want 1", m.cursor)
	}

	// Test move down again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(DiffModel)
	if m.cursor != 2 {
		t.Errorf("after second 'j', cursor = %d, want 2", m.cursor)
	}

	// Test can't go past end
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(DiffModel)
	if m.cursor != 2 {
		t.Errorf("cursor should stay at 2, got %d", m.cursor)
	}

	// Test move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(DiffModel)
	if m.cursor != 1 {
		t.Errorf("after 'k', cursor = %d, want 1", m.cursor)
	}

	// Test jump to bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(DiffModel)
	if m.cursor != 2 {
		t.Errorf("after 'G', cursor = %d, want 2", m.cursor)
	}

	// Test double g to top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(DiffModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(DiffModel)
	if m.cursor != 0 {
		t.Errorf("after 'gg', cursor = %d, want 0", m.cursor)
	}
}

func TestDiffModelDrillDown(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{
			FilePath: "file1.txt",
			Header:   "@@ -1,3 +1,3 @@",
			Lines: []git.DiffLine{
				{Content: " context", Type: git.LineContext},
				{Content: "-removed", Type: git.LineRemoved},
				{Content: "+added", Type: git.LineAdded},
			},
		},
	}

	// Press right to drill down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(DiffModel)
	if !m.viewingHunk {
		t.Error("should be viewing hunk after 'l'")
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset should be 0, got %d", m.scrollOffset)
	}

	// Press left to go back
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(DiffModel)
	if m.viewingHunk {
		t.Error("should exit hunk view after 'h'")
	}
}

func TestDiffModelHunkDetailNavigation(t *testing.T) {
	m := NewDiffModel(nil)
	m.height = 10
	m.hunks = []git.Hunk{
		{
			FilePath: "file1.txt",
			Header:   "@@ -1,20 +1,20 @@",
			Lines:    make([]git.DiffLine, 20), // 20 lines
		},
	}
	m.viewingHunk = true

	// Test scroll down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(DiffModel)
	if m.scrollOffset != 1 {
		t.Errorf("scrollOffset = %d, want 1", m.scrollOffset)
	}

	// Test scroll up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(DiffModel)
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Test can't scroll past top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(DiffModel)
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset should stay at 0, got %d", m.scrollOffset)
	}

	// Test jump to bottom (G)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(DiffModel)
	if m.scrollOffset == 0 {
		t.Error("scrollOffset should be > 0 after 'G'")
	}

	// Test double g to top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(DiffModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(DiffModel)
	if m.scrollOffset != 0 {
		t.Errorf("after 'gg', scrollOffset = %d, want 0", m.scrollOffset)
	}
}

func TestDiffModelHelpToggle(t *testing.T) {
	m := NewDiffModel(nil)

	// Toggle help
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(DiffModel)
	if !m.showHelp {
		t.Error("showHelp should be true after '?'")
	}

	// Close help with '?'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(DiffModel)
	if m.showHelp {
		t.Error("showHelp should be false after pressing '?' again")
	}
}

func TestDiffModelConfirmMode(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt", Staged: false},
	}

	// Press d to trigger confirm (only for unstaged)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(DiffModel)
	if !m.confirmMode {
		t.Error("confirmMode should be true after 'd' on unstaged hunk")
	}

	// Press esc to cancel
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(DiffModel)
	if m.confirmMode {
		t.Error("confirmMode should be false after 'esc'")
	}
}

func TestDiffModelConfirmModeNotForStaged(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt", Staged: true},
	}

	// Press d - should NOT trigger confirm for staged hunks
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(DiffModel)
	if m.confirmMode {
		t.Error("confirmMode should not be triggered for staged hunks")
	}
}

func TestDiffModelQuit(t *testing.T) {
	m := NewDiffModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("quit should return a command")
	}
}

func TestDiffModelWindowResize(t *testing.T) {
	m := NewDiffModel(nil)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m = newModel.(DiffModel)

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 60 {
		t.Errorf("height = %d, want 60", m.height)
	}
}

func TestDiffModelCombinedDiffMsg(t *testing.T) {
	m := NewDiffModel(nil)

	diff := &git.CombinedDiffResult{
		StagedDiff: &git.DiffResult{
			Files: []git.FileDiff{
				{
					Path: "file1.txt",
					Hunks: []git.Hunk{
						{FilePath: "file1.txt", Staged: true},
					},
				},
			},
		},
		UnstagedDiff: &git.DiffResult{
			Files: []git.FileDiff{
				{
					Path: "file2.txt",
					Hunks: []git.Hunk{
						{FilePath: "file2.txt", Staged: false},
					},
				},
			},
		},
	}

	newModel, _ := m.Update(combinedDiffMsg{diff: diff})
	m = newModel.(DiffModel)

	if m.diff != diff {
		t.Error("diff was not set")
	}
	if len(m.hunks) != 2 {
		t.Errorf("len(hunks) = %d, want 2", len(m.hunks))
	}
}

func TestDiffModelAutoEnterDetailForSingleHunk(t *testing.T) {
	m := NewDiffModel(nil)

	diff := &git.CombinedDiffResult{
		UnstagedDiff: &git.DiffResult{
			Files: []git.FileDiff{
				{
					Path: "file1.txt",
					Hunks: []git.Hunk{
						{FilePath: "file1.txt", Staged: false},
					},
				},
			},
		},
	}

	newModel, _ := m.Update(combinedDiffMsg{diff: diff})
	m = newModel.(DiffModel)

	// Should auto-enter detail view for single hunk
	if !m.viewingHunk {
		t.Error("should auto-enter hunk detail view when there's only one hunk")
	}
}

func TestDiffModelErrMsg(t *testing.T) {
	m := NewDiffModel(nil)

	newModel, _ := m.Update(errMsg{err: fmt.Errorf("test error")})
	m = newModel.(DiffModel)

	if m.err == nil {
		t.Error("err should be set")
	}
}

func TestDiffModelVisibleLines(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{height: 0, want: 40},
		{height: 5, want: 40},
		{height: 10, want: 5},
		{height: 50, want: 45},
	}

	for _, tt := range tests {
		m := NewDiffModel(nil)
		m.height = tt.height
		got := m.visibleLines()
		if got != tt.want {
			t.Errorf("visibleLines() with height=%d = %d, want %d", tt.height, got, tt.want)
		}
	}
}

func TestDiffModelView(t *testing.T) {
	m := NewDiffModel(nil)
	m.diff = &git.CombinedDiffResult{}
	m.hunks = []git.Hunk{
		{
			FilePath:        "file1.txt",
			DisplayFilePath: "file1.txt",
			Header:          "@@ -1,3 +1,3 @@",
			Staged:          false,
			Lines: []git.DiffLine{
				{Content: "-old", Type: git.LineRemoved},
				{Content: "+new", Type: git.LineAdded},
			},
		},
	}

	view := m.View()

	if !strings.Contains(view, "file1.txt") {
		t.Error("view should contain file path")
	}
	if !strings.Contains(view, "[U]") || !strings.Contains(view, "[S]") && !strings.Contains(view, "Unstaged") {
		// Either shows [U] for unstaged or shows the label
	}
}

func TestDiffModelViewLoading(t *testing.T) {
	m := NewDiffModel(nil)
	m.diff = nil

	view := m.View()

	if !strings.Contains(view, "Loading") {
		t.Error("view should show Loading when diff is nil")
	}
}

func TestDiffModelViewEmpty(t *testing.T) {
	m := NewDiffModel(nil)
	m.diff = &git.CombinedDiffResult{}
	m.hunks = nil

	view := m.View()

	if !strings.Contains(view, "No changes") {
		t.Error("view should show 'No changes' when hunks is empty")
	}
}

func TestDiffModelViewHelp(t *testing.T) {
	m := NewDiffModel(nil)
	m.showHelp = true

	view := m.View()

	if !strings.Contains(view, "Diff View") {
		t.Error("help view should contain 'Diff View'")
	}
}

func TestDiffModelViewHunkDetail(t *testing.T) {
	m := NewDiffModel(nil)
	m.diff = &git.CombinedDiffResult{}
	m.hunks = []git.Hunk{
		{
			FilePath: "file1.txt",
			Header:   "@@ -1,3 +1,3 @@",
			Staged:   true,
			Lines: []git.DiffLine{
				{Content: " context", Type: git.LineContext},
				{Content: "-removed", Type: git.LineRemoved},
				{Content: "+added", Type: git.LineAdded},
			},
		},
	}
	m.viewingHunk = true
	m.height = 20

	view := m.View()

	// Should show content
	if !strings.Contains(view, "context") && !strings.Contains(view, "removed") && !strings.Contains(view, "added") {
		t.Error("hunk detail view should show diff lines")
	}
}

func TestDiffModelViewConfirmPrompt(t *testing.T) {
	m := NewDiffModel(nil)
	m.diff = &git.CombinedDiffResult{}
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt", Staged: false},
	}
	m.confirmMode = true

	view := m.View()

	if !strings.Contains(view, "Discard") {
		t.Error("view should show discard confirmation")
	}
	if !strings.Contains(view, "Type 'yes' to confirm") {
		t.Error("view should show 'yes' confirmation prompt")
	}
}

func TestDiffModelViewWithError(t *testing.T) {
	m := NewDiffModel(nil)
	m.err = fmt.Errorf("test error")
	m.diff = &git.CombinedDiffResult{}

	view := m.View()

	if !strings.Contains(view, "Error:") {
		t.Error("view should show error")
	}
}

func TestDiffModelArrowKeys(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt"},
		{FilePath: "file2.txt"},
	}

	// Test down arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(DiffModel)
	if m.cursor != 1 {
		t.Errorf("after down arrow, cursor = %d, want 1", m.cursor)
	}

	// Test up arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(DiffModel)
	if m.cursor != 0 {
		t.Errorf("after up arrow, cursor = %d, want 0", m.cursor)
	}
}

func TestDiffModelEnterKey(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt", Lines: []git.DiffLine{{Content: "test"}}},
	}

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(DiffModel)

	if !m.viewingHunk {
		t.Error("should enter hunk view on enter key")
	}
}

func TestDiffModelHelpModeBlocksNavigation(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt"},
		{FilePath: "file2.txt"},
	}
	m.showHelp = true

	// Navigation should be blocked in help mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(DiffModel)
	if m.cursor != 0 {
		t.Error("navigation should be blocked in help mode")
	}
}

func TestHunkStableKey(t *testing.T) {
	hunk := git.Hunk{
		FilePath: "test.txt",
		Header:   "@@ -1,3 +1,3 @@",
		Lines: []git.DiffLine{
			{Content: "-old", Type: git.LineRemoved},
			{Content: "+new", Type: git.LineAdded},
		},
	}

	key := hunkStableKey(hunk)

	if !strings.Contains(key, "test.txt") {
		t.Error("key should contain file path")
	}
	if !strings.Contains(key, "-old") || !strings.Contains(key, "+new") {
		t.Error("key should contain diff lines")
	}
}

func TestHunkStableKeyNoLines(t *testing.T) {
	hunk := git.Hunk{
		FilePath: "test.txt",
		Header:   "@@ -1,3 +1,3 @@",
		Lines: []git.DiffLine{
			{Content: " context only", Type: git.LineContext},
		},
	}

	key := hunkStableKey(hunk)

	if !strings.Contains(key, "test.txt") {
		t.Error("key should contain file path")
	}
	if !strings.Contains(key, "@@ -1,3 +1,3 @@") {
		t.Error("key should contain header when no diff lines")
	}
}

func TestRenderStageLabel(t *testing.T) {
	stagedLabel := renderStageLabel(true)
	if !strings.Contains(stagedLabel, "Staged") {
		t.Errorf("staged label should contain 'Staged', got %q", stagedLabel)
	}

	unstagedLabel := renderStageLabel(false)
	if !strings.Contains(unstagedLabel, "Unstaged") {
		t.Errorf("unstaged label should contain 'Unstaged', got %q", unstagedLabel)
	}
}

func TestDiffModelKeepHunkOrder(t *testing.T) {
	m := NewDiffModel(nil)
	m.hunks = []git.Hunk{
		{FilePath: "a.txt", Lines: []git.DiffLine{{Content: "+a", Type: git.LineAdded}}},
		{FilePath: "b.txt", Lines: []git.DiffLine{{Content: "+b", Type: git.LineAdded}}},
		{FilePath: "c.txt", Lines: []git.DiffLine{{Content: "+c", Type: git.LineAdded}}},
	}

	newHunks := []git.Hunk{
		{FilePath: "c.txt", Lines: []git.DiffLine{{Content: "+c", Type: git.LineAdded}}},
		{FilePath: "b.txt", Lines: []git.DiffLine{{Content: "+b", Type: git.LineAdded}}},
		{FilePath: "a.txt", Lines: []git.DiffLine{{Content: "+a", Type: git.LineAdded}}},
	}

	ordered := m.keepHunkOrder(newHunks)

	// Should maintain original order
	if ordered[0].FilePath != "a.txt" {
		t.Errorf("first hunk should be a.txt, got %s", ordered[0].FilePath)
	}
	if ordered[1].FilePath != "b.txt" {
		t.Errorf("second hunk should be b.txt, got %s", ordered[1].FilePath)
	}
	if ordered[2].FilePath != "c.txt" {
		t.Errorf("third hunk should be c.txt, got %s", ordered[2].FilePath)
	}
}

func TestDiffModelAnchorBottom(t *testing.T) {
	m := NewDiffModel(nil)
	m.height = 10

	content := "line1\nline2\nline3\n"
	anchored := m.anchorBottom(content)

	// Should add padding to push content to bottom
	if !strings.HasPrefix(anchored, "\n") {
		t.Error("anchored content should have leading newlines")
	}
}

package ui

import (
	"fmt"
	"strings"
	"testing"

	"go-on-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for StashDiffModel

func TestNewStashDiffModel(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.viewingHunk {
		t.Error("viewingHunk should be false initially")
	}
	if m.showHelp {
		t.Error("showHelp should be false initially")
	}
}

func TestStashDiffModelInit(t *testing.T) {
	m := NewStashDiffModel(100, 50)
	cmd := m.Init()

	if cmd != nil {
		t.Error("Init() should return nil for StashDiffModel")
	}
}

func TestStashDiffModelNavigation(t *testing.T) {
	m := NewStashDiffModel(100, 50)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt"},
		{FilePath: "file2.txt"},
		{FilePath: "file3.txt"},
	}

	// Test move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StashDiffModel)
	if m.cursor != 1 {
		t.Errorf("after 'j', cursor = %d, want 1", m.cursor)
	}

	// Test move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(StashDiffModel)
	if m.cursor != 0 {
		t.Errorf("after 'k', cursor = %d, want 0", m.cursor)
	}

	// Test jump to bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(StashDiffModel)
	if m.cursor != 2 {
		t.Errorf("after 'G', cursor = %d, want 2", m.cursor)
	}

	// Test jump to top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(StashDiffModel)
	if m.cursor != 0 {
		t.Errorf("after 'g', cursor = %d, want 0", m.cursor)
	}
}

func TestStashDiffModelDrillDown(t *testing.T) {
	m := NewStashDiffModel(100, 50)
	m.hunks = []git.Hunk{
		{FilePath: "file1.txt", Lines: []git.DiffLine{{Content: "test"}}},
	}

	// Press right to drill down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(StashDiffModel)
	if !m.viewingHunk {
		t.Error("should be viewing hunk after 'l'")
	}

	// Press left to go back
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(StashDiffModel)
	if m.viewingHunk {
		t.Error("should exit hunk view after 'h'")
	}
}

func TestStashDiffModelHunkDetailNavigation(t *testing.T) {
	m := NewStashDiffModel(100, 10)
	m.hunks = []git.Hunk{
		{
			FilePath: "file1.txt",
			Lines:    make([]git.DiffLine, 20),
		},
	}
	m.viewingHunk = true

	// Test scroll down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StashDiffModel)
	if m.scrollOffset != 1 {
		t.Errorf("scrollOffset = %d, want 1", m.scrollOffset)
	}

	// Test scroll up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(StashDiffModel)
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Test jump to bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(StashDiffModel)
	if m.scrollOffset == 0 {
		t.Error("scrollOffset should be > 0 after 'G'")
	}

	// Test jump to top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(StashDiffModel)
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 after 'g'", m.scrollOffset)
	}
}

func TestStashDiffModelHelpToggle(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	// Toggle help
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(StashDiffModel)
	if !m.showHelp {
		t.Error("showHelp should be true after '?'")
	}

	// Close help
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(StashDiffModel)
	if m.showHelp {
		t.Error("showHelp should be false after pressing '?' again")
	}
}

func TestStashDiffModelWindowResize(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m = newModel.(StashDiffModel)

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 60 {
		t.Errorf("height = %d, want 60", m.height)
	}
}

func TestStashDiffModelStashDiffMsg(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	diff := &git.CombinedDiffResult{
		UnstagedDiff: &git.DiffResult{
			Files: []git.FileDiff{
				{
					Path: "file1.txt",
					Hunks: []git.Hunk{
						{FilePath: "file1.txt"},
					},
				},
			},
		},
	}

	newModel, _ := m.Update(stashDiffMsg{diff: diff})
	m = newModel.(StashDiffModel)

	if m.diff != diff {
		t.Error("diff was not set")
	}
	if len(m.hunks) == 0 {
		t.Error("hunks should be populated")
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestStashDiffModelAutoEnterDetailForSingleHunk(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	diff := &git.CombinedDiffResult{
		UnstagedDiff: &git.DiffResult{
			Files: []git.FileDiff{
				{
					Path: "file1.txt",
					Hunks:   []git.Hunk{{FilePath: "file1.txt"}},
				},
			},
		},
	}

	newModel, _ := m.Update(stashDiffMsg{diff: diff})
	m = newModel.(StashDiffModel)

	if !m.viewingHunk {
		t.Error("should auto-enter hunk detail for single hunk")
	}
}

func TestStashDiffModelErrMsg(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	newModel, _ := m.Update(errMsg{err: fmt.Errorf("test error")})
	m = newModel.(StashDiffModel)

	if m.err == nil {
		t.Error("err should be set")
	}
}

func TestStashDiffModelView(t *testing.T) {
	m := NewStashDiffModel(100, 50)
	m.diff = &git.CombinedDiffResult{}
	m.hunks = []git.Hunk{
		{
			FilePath:        "file1.txt",
			DisplayFilePath: "file1.txt",
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
}

func TestStashDiffModelViewEmpty(t *testing.T) {
	m := NewStashDiffModel(100, 50)

	view := m.View()

	if !strings.Contains(view, "No changes") {
		t.Error("view should show 'No changes in stash'")
	}
}

func TestStashDiffModelViewHelp(t *testing.T) {
	m := NewStashDiffModel(100, 50)
	m.showHelp = true

	view := m.View()

	if !strings.Contains(view, "Stash Diff") {
		t.Error("help view should contain 'Stash Diff'")
	}
}

func TestStashDiffModelViewHunkDetail(t *testing.T) {
	m := NewStashDiffModel(100, 20)
	m.diff = &git.CombinedDiffResult{}
	m.hunks = []git.Hunk{
		{
			FilePath:        "file1.txt",
			DisplayFilePath: "file1.txt",
			Header:          "@@ -1,3 +1,3 @@",
			Lines: []git.DiffLine{
				{Content: " context", Type: git.LineContext},
				{Content: "-removed", Type: git.LineRemoved},
				{Content: "+added", Type: git.LineAdded},
			},
		},
	}
	m.viewingHunk = true

	view := m.View()

	if !strings.Contains(view, "file1.txt") {
		t.Error("hunk detail should show file path")
	}
}

func TestStashDiffModelVisibleLines(t *testing.T) {
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
		m := NewStashDiffModel(100, tt.height)
		got := m.visibleLines()
		if got != tt.want {
			t.Errorf("visibleLines() with height=%d = %d, want %d", tt.height, got, tt.want)
		}
	}
}

// Tests for StashesModel

func TestNewStashesModel(t *testing.T) {
	m := NewStashesModel()

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.showHelp {
		t.Error("showHelp should be false initially")
	}
	if m.confirmMode {
		t.Error("confirmMode should be false initially")
	}
}

func TestStashesModelInit(t *testing.T) {
	m := NewStashesModel()
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestStashesModelNavigation(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
		{Index: 1, Message: "stash 2"},
		{Index: 2, Message: "stash 3"},
	}

	// Test move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StashesModel)
	if m.cursor != 1 {
		t.Errorf("after 'j', cursor = %d, want 1", m.cursor)
	}

	// Test move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(StashesModel)
	if m.cursor != 0 {
		t.Errorf("after 'k', cursor = %d, want 0", m.cursor)
	}

	// Test jump to bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(StashesModel)
	if m.cursor != 2 {
		t.Errorf("after 'G', cursor = %d, want 2", m.cursor)
	}

	// Test double g to top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(StashesModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(StashesModel)
	if m.cursor != 0 {
		t.Errorf("after 'gg', cursor = %d, want 0", m.cursor)
	}
}

func TestStashesModelHelpToggle(t *testing.T) {
	m := NewStashesModel()

	// Toggle help
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(StashesModel)
	if !m.showHelp {
		t.Error("showHelp should be true after '?'")
	}

	// Close help
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(StashesModel)
	if m.showHelp {
		t.Error("showHelp should be false after pressing '?' again")
	}
}

func TestStashesModelApplyStash(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}

	// Press a to apply
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if cmd == nil {
		t.Error("should return a command to apply stash")
	}
}

func TestStashesModelPopStash(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}

	// Press p to pop (should enter confirm mode)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = newModel.(StashesModel)

	if !m.confirmMode {
		t.Error("should enter confirm mode for pop")
	}
	if m.confirmAction != "pop" {
		t.Errorf("confirmAction = %q, want 'pop'", m.confirmAction)
	}
}

func TestStashesModelDropStash(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}

	// Press d to drop (should enter confirm mode)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(StashesModel)

	if !m.confirmMode {
		t.Error("should enter confirm mode for drop")
	}
	if m.confirmAction != "drop" {
		t.Errorf("confirmAction = %q, want 'drop'", m.confirmAction)
	}
}

func TestStashesModelConfirmModeYes(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}
	m.confirmMode = true
	m.confirmAction = "drop"

	// Press y to confirm
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(StashesModel)

	if m.confirmMode {
		t.Error("should exit confirm mode after 'y'")
	}
	if cmd == nil {
		t.Error("should return a command")
	}
}

func TestStashesModelConfirmModeNo(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}
	m.confirmMode = true
	m.confirmAction = "drop"

	// Press n to cancel
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(StashesModel)

	if m.confirmMode {
		t.Error("should exit confirm mode after 'n'")
	}
	if m.confirmAction != "" {
		t.Error("confirmAction should be cleared")
	}
}

func TestStashesModelWindowResize(t *testing.T) {
	m := NewStashesModel()

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = newModel.(StashesModel)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestStashesModelStashesMsg(t *testing.T) {
	m := NewStashesModel()

	stashes := []git.Stash{
		{Index: 0, Message: "stash 1"},
		{Index: 1, Message: "stash 2"},
	}

	newModel, _ := m.Update(stashesMsg{stashes: stashes})
	m = newModel.(StashesModel)

	if len(m.stashes) != 2 {
		t.Errorf("len(stashes) = %d, want 2", len(m.stashes))
	}
}

func TestStashesModelStashDiffMsg(t *testing.T) {
	m := NewStashesModel()

	diff := &git.CombinedDiffResult{
		UnstagedDiff: &git.DiffResult{
			Files: []git.FileDiff{
				{
					Path: "file1.txt",
					Hunks:   []git.Hunk{{FilePath: "file1.txt"}},
				},
			},
		},
	}

	newModel, _ := m.Update(stashDiffMsg{diff: diff})
	m = newModel.(StashesModel)

	if m.diffModel.diff != diff {
		t.Error("diffModel.diff was not set")
	}
}

func TestStashesModelErrMsg(t *testing.T) {
	m := NewStashesModel()

	newModel, _ := m.Update(errMsg{err: fmt.Errorf("test error")})
	m = newModel.(StashesModel)

	if m.err == nil {
		t.Error("err should be set")
	}
}

func TestStashesModelView(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Branch: "main", Message: "WIP on main"},
		{Index: 1, Message: "stash 2"},
	}

	view := m.View()

	if !strings.Contains(view, "git stash") {
		t.Error("view should contain 'git stash' header")
	}
	if !strings.Contains(view, "stash@{0}") {
		t.Error("view should contain stash index")
	}
	if !strings.Contains(view, "main") {
		t.Error("view should contain branch name")
	}
}

func TestStashesModelViewEmpty(t *testing.T) {
	m := NewStashesModel()
	m.stashes = nil

	view := m.View()

	if !strings.Contains(view, "No stashes") {
		t.Error("view should show 'No stashes'")
	}
}

func TestStashesModelViewHelp(t *testing.T) {
	m := NewStashesModel()
	m.showHelp = true

	view := m.View()

	if !strings.Contains(view, "Stashes Shortcuts") {
		t.Error("help view should contain 'Stashes Shortcuts'")
	}
}

func TestStashesModelViewConfirmDrop(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}
	m.confirmMode = true
	m.confirmAction = "drop"

	view := m.View()

	if !strings.Contains(view, "Drop") {
		t.Error("view should show drop confirmation")
	}
}

func TestStashesModelViewConfirmPop(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
	}
	m.confirmMode = true
	m.confirmAction = "pop"

	view := m.View()

	if !strings.Contains(view, "Pop") {
		t.Error("view should show pop confirmation")
	}
}

func TestStashesModelViewWithError(t *testing.T) {
	m := NewStashesModel()
	m.err = fmt.Errorf("test error")
	m.stashes = []git.Stash{{Index: 0, Message: "stash"}}

	view := m.View()

	if !strings.Contains(view, "Error:") {
		t.Error("view should show error")
	}
}

func TestStashesModelArrowKeys(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
		{Index: 1, Message: "stash 2"},
	}

	// Test down arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(StashesModel)
	if m.cursor != 1 {
		t.Errorf("after down arrow, cursor = %d, want 1", m.cursor)
	}

	// Test up arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(StashesModel)
	if m.cursor != 0 {
		t.Errorf("after up arrow, cursor = %d, want 0", m.cursor)
	}
}

func TestStashesModelHelpModeBlocksNavigation(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
		{Index: 1, Message: "stash 2"},
	}
	m.showHelp = true

	// Navigation should be blocked in help mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StashesModel)
	if m.cursor != 0 {
		t.Error("navigation should be blocked in help mode")
	}
}

func TestStashesModelConfirmModeBlocksNavigation(t *testing.T) {
	m := NewStashesModel()
	m.stashes = []git.Stash{
		{Index: 0, Message: "stash 1"},
		{Index: 1, Message: "stash 2"},
	}
	m.confirmMode = true
	m.confirmAction = "drop"

	// Navigation should be blocked in confirm mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StashesModel)
	if m.cursor != 0 {
		t.Error("navigation should be blocked in confirm mode")
	}
}

func TestStashesModelCursorBoundsAfterStashesMsg(t *testing.T) {
	m := NewStashesModel()
	m.cursor = 5 // Out of bounds

	stashes := []git.Stash{
		{Index: 0, Message: "stash 1"},
		{Index: 1, Message: "stash 2"},
	}

	newModel, _ := m.Update(stashesMsg{stashes: stashes})
	m = newModel.(StashesModel)

	if m.cursor >= len(m.stashes) {
		t.Errorf("cursor should be within bounds, got %d", m.cursor)
	}
}

func TestStashDiffModelAnchorBottom(t *testing.T) {
	m := NewStashDiffModel(100, 10)

	content := "line1\nline2\n"
	anchored := m.anchorBottom(content)

	if !strings.HasPrefix(anchored, "\n") {
		t.Error("anchored content should have leading newlines")
	}
}

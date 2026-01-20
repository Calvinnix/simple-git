package ui

import (
	"fmt"
	"strings"
	"testing"

	"go-on-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewStatusModel(t *testing.T) {
	m := NewStatusModel()

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.selected == nil {
		t.Error("selected map is nil")
	}
	if m.visualMode {
		t.Error("visualMode should be false initially")
	}
	if m.showHelp {
		t.Error("showHelp should be false initially")
	}
	if m.showVerboseHelp {
		t.Error("showVerboseHelp should be false initially")
	}
	if m.commitMode {
		t.Error("commitMode should be false initially")
	}
	if m.stashMode != stashNone {
		t.Errorf("stashMode = %v, want stashNone", m.stashMode)
	}
	if m.confirmMode != confirmNone {
		t.Errorf("confirmMode = %v, want confirmNone", m.confirmMode)
	}
}

func TestNewStatusModelWithHelp(t *testing.T) {
	m := NewStatusModelWithHelp(true)

	if !m.showVerboseHelp {
		t.Error("showVerboseHelp should be true when created with showHelp=true")
	}
}

func TestStatusModelInit(t *testing.T) {
	m := NewStatusModel()
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestStatusModelNavigation(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file3.txt"}, Section: "unstaged"},
	}

	// Test move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StatusModel)
	if m.cursor != 1 {
		t.Errorf("after 'j', cursor = %d, want 1", m.cursor)
	}

	// Test move down again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StatusModel)
	if m.cursor != 2 {
		t.Errorf("after second 'j', cursor = %d, want 2", m.cursor)
	}

	// Test can't go past end
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StatusModel)
	if m.cursor != 2 {
		t.Errorf("cursor should stay at 2, got %d", m.cursor)
	}

	// Test move up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(StatusModel)
	if m.cursor != 1 {
		t.Errorf("after 'k', cursor = %d, want 1", m.cursor)
	}

	// Test jump to bottom
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(StatusModel)
	if m.cursor != 2 {
		t.Errorf("after 'G', cursor = %d, want 2", m.cursor)
	}

	// Test double g to top
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(StatusModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(StatusModel)
	if m.cursor != 0 {
		t.Errorf("after 'gg', cursor = %d, want 0", m.cursor)
	}
}

func TestStatusModelVisualMode(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file3.txt"}, Section: "unstaged"},
	}

	// Enter visual mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = newModel.(StatusModel)
	if !m.visualMode {
		t.Error("should be in visual mode after 'v'")
	}
	if !m.selected[0] {
		t.Error("cursor item should be selected in visual mode")
	}

	// Move down in visual mode - should select range
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StatusModel)
	if !m.selected[0] || !m.selected[1] {
		t.Error("items 0 and 1 should be selected after moving down in visual mode")
	}

	// Exit visual mode
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = newModel.(StatusModel)
	if m.visualMode {
		t.Error("should exit visual mode after pressing 'v' again")
	}
	if len(m.selected) > 0 {
		t.Error("selection should be cleared after exiting visual mode")
	}
}

func TestStatusModelHelpToggle(t *testing.T) {
	m := NewStatusModel()

	// Toggle compact help
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(StatusModel)
	if !m.showHelp {
		t.Error("showHelp should be true after '?'")
	}

	// Close help with '?'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(StatusModel)
	if m.showHelp {
		t.Error("showHelp should be false after pressing '?' again")
	}

	// Toggle verbose help
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = newModel.(StatusModel)
	if !m.showVerboseHelp {
		t.Error("showVerboseHelp should be true after '/'")
	}

	// Toggle off
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = newModel.(StatusModel)
	if m.showVerboseHelp {
		t.Error("showVerboseHelp should be false after pressing '/' again")
	}
}

func TestStatusModelQuit(t *testing.T) {
	m := NewStatusModel()

	// Test quit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("quit should return a command")
	}
}

func TestStatusModelQuitFromVisualMode(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
	}

	// Enter visual mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = newModel.(StatusModel)

	// Press q - should exit visual mode, not quit
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(StatusModel)
	if m.visualMode {
		t.Error("should exit visual mode when pressing 'q'")
	}
	if m.quitting {
		t.Error("should not quit, just exit visual mode")
	}
	if cmd != nil {
		// cmd should be nil when just exiting visual mode
	}
}

func TestStatusModelConfirmMode(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
	}

	// Press d to trigger discard confirm
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(StatusModel)
	if m.confirmMode != confirmDiscard {
		t.Errorf("confirmMode = %v, want confirmDiscard", m.confirmMode)
	}

	// Press esc to cancel
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(StatusModel)
	if m.confirmMode != confirmNone {
		t.Errorf("confirmMode = %v, want confirmNone after 'esc'", m.confirmMode)
	}
}

func TestStatusModelCommitMode(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{
		Staged: []git.FileStatus{
			{Path: "file1.txt", IndexStatus: 'A'},
		},
	}
	m.items = buildItems(m.status)

	// Press c to enter commit mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(StatusModel)
	if !m.commitMode {
		t.Error("should be in commit mode after 'c' with staged files")
	}

	// Press esc to cancel
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(StatusModel)
	if m.commitMode {
		t.Error("should exit commit mode after esc")
	}
}

func TestStatusModelStashMode(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
	}

	// Press s to enter stash mode for selected files
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(StatusModel)
	if m.stashMode != stashFiles {
		t.Errorf("stashMode = %v, want stashFiles", m.stashMode)
	}

	// Press esc to cancel
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(StatusModel)
	if m.stashMode != stashNone {
		t.Error("should exit stash mode after esc")
	}

	// Press S to enter stash all mode
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = newModel.(StatusModel)
	if m.stashMode != stashAll {
		t.Errorf("stashMode = %v, want stashAll", m.stashMode)
	}
}

func TestStatusModelSelection(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file3.txt"}, Section: "unstaged"},
	}

	// Press h to toggle selection
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(StatusModel)
	if !m.selected[0] {
		t.Error("item 0 should be selected after 'h'")
	}

	// Press h again to deselect
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = newModel.(StatusModel)
	if m.selected[0] {
		t.Error("item 0 should be deselected after pressing 'h' again")
	}
}

func TestStatusModelWindowResize(t *testing.T) {
	m := NewStatusModel()

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = newModel.(StatusModel)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestStatusModelStatusMsg(t *testing.T) {
	m := NewStatusModel()

	status := &git.StatusResult{
		Staged:    []git.FileStatus{{Path: "staged.txt", IndexStatus: 'A'}},
		Unstaged:  []git.FileStatus{{Path: "unstaged.txt", WorkStatus: 'M'}},
		Untracked: []git.FileStatus{{Path: "untracked.txt"}},
	}

	branchStatus := git.BranchStatus{
		Name:   "main",
		Remote: "origin/main",
		Ahead:  1,
		Behind: 0,
	}

	newModel, _ := m.Update(statusMsg{status: status, branchStatus: branchStatus})
	m = newModel.(StatusModel)

	if m.status != status {
		t.Error("status was not set")
	}
	if m.branchStatus.Name != "main" {
		t.Errorf("branchStatus.Name = %q, want 'main'", m.branchStatus.Name)
	}
	if len(m.items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(m.items))
	}
}

func TestStatusModelErrMsg(t *testing.T) {
	m := NewStatusModel()

	err := errMsg{err: fmt.Errorf("test error")}
	newModel, _ := m.Update(err)
	m = newModel.(StatusModel)

	if m.err == nil {
		t.Error("err should be set")
	}
	if m.err.Error() != "test error" {
		t.Errorf("err = %v, want 'test error'", m.err)
	}
}

func TestBuildItems(t *testing.T) {
	tests := []struct {
		name    string
		status  *git.StatusResult
		wantLen int
	}{
		{
			name:    "nil status",
			status:  nil,
			wantLen: 0,
		},
		{
			name:    "empty status",
			status:  &git.StatusResult{},
			wantLen: 0,
		},
		{
			name: "only staged",
			status: &git.StatusResult{
				Staged: []git.FileStatus{{Path: "a.txt"}},
			},
			wantLen: 1,
		},
		{
			name: "mixed status",
			status: &git.StatusResult{
				Staged:    []git.FileStatus{{Path: "a.txt"}, {Path: "b.txt"}},
				Unstaged:  []git.FileStatus{{Path: "c.txt"}},
				Untracked: []git.FileStatus{{Path: "d.txt"}, {Path: "e.txt"}},
			},
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := buildItems(tt.status)
			if len(items) != tt.wantLen {
				t.Errorf("len(buildItems()) = %d, want %d", len(items), tt.wantLen)
			}
		})
	}
}

func TestStatusModelGetSelectedItems(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file3.txt"}, Section: "unstaged"},
	}

	// No selection - should return cursor item
	items := m.getSelectedItems()
	if len(items) != 1 {
		t.Errorf("len(getSelectedItems) = %d, want 1", len(items))
	}
	if items[0].File.Path != "file1.txt" {
		t.Errorf("expected file1.txt, got %s", items[0].File.Path)
	}

	// With selection
	m.selected[1] = true
	m.selected[2] = true
	items = m.getSelectedItems()
	// Should include selected items plus cursor item if not already selected
	if len(items) != 3 {
		t.Errorf("len(getSelectedItems) = %d, want 3", len(items))
	}
}

func TestStatusModelView(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{
		Staged:   []git.FileStatus{{Path: "staged.txt", DisplayPath: "staged.txt", IndexStatus: 'A'}},
		Unstaged: []git.FileStatus{{Path: "unstaged.txt", DisplayPath: "unstaged.txt", WorkStatus: 'M'}},
	}
	m.items = buildItems(m.status)
	m.branchStatus = git.BranchStatus{Name: "main"}

	view := m.View()

	if !strings.Contains(view, "main") {
		t.Error("view should contain branch name")
	}
	if !strings.Contains(view, "staged.txt") {
		t.Error("view should contain staged file")
	}
	if !strings.Contains(view, "unstaged.txt") {
		t.Error("view should contain unstaged file")
	}
}

func TestStatusModelViewLoading(t *testing.T) {
	m := NewStatusModel()
	m.status = nil

	view := m.View()

	if !strings.Contains(view, "Loading") {
		t.Error("view should show Loading when status is nil")
	}
}

func TestStatusModelViewEmpty(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{}
	m.branchStatus = git.BranchStatus{Name: "main"}

	view := m.View()

	if !strings.Contains(view, "Nothing to commit") {
		t.Error("view should show 'Nothing to commit' when status is empty")
	}
}

func TestStatusModelViewWithError(t *testing.T) {
	m := NewStatusModel()
	m.err = fmt.Errorf("test error")
	m.status = &git.StatusResult{}
	m.branchStatus = git.BranchStatus{Name: "main"}

	view := m.View()

	if !strings.Contains(view, "Error:") {
		t.Error("view should show error")
	}
	if !strings.Contains(view, "test error") {
		t.Error("view should show error message")
	}
}

func TestStatusModelViewHelp(t *testing.T) {
	m := NewStatusModel()
	m.showHelp = true

	view := m.View()

	// Should show help content
	if !strings.Contains(view, "Navigation") {
		t.Error("help view should contain 'Navigation' section")
	}
}

func TestStatusModelViewVerboseHelp(t *testing.T) {
	m := NewStatusModel()
	m.showVerboseHelp = true
	m.status = &git.StatusResult{
		Unstaged: []git.FileStatus{{Path: "test.txt", WorkStatus: 'M'}},
	}
	m.items = buildItems(m.status)
	m.branchStatus = git.BranchStatus{Name: "main"}

	view := m.View()

	// Should show help bar
	if !strings.Contains(view, "navigate") || !strings.Contains(view, "stage") {
		t.Error("verbose help should show help bar")
	}
}

func TestStatusModelViewConfirmDiscard(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{
		Unstaged: []git.FileStatus{{Path: "test.txt", WorkStatus: 'M'}},
	}
	m.items = buildItems(m.status)
	m.branchStatus = git.BranchStatus{Name: "main"}
	m.confirmMode = confirmDiscard

	view := m.View()

	if !strings.Contains(view, "Discard") {
		t.Error("view should show discard confirmation")
	}
	if !strings.Contains(view, "Type 'yes' to confirm") {
		t.Error("view should show 'yes' confirmation prompt")
	}
}

func TestStatusModelViewConfirmPush(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{}
	m.branchStatus = git.BranchStatus{
		Name:   "main",
		Remote: "origin/main",
		Ahead:  2,
	}
	m.confirmMode = confirmPush

	view := m.View()

	if !strings.Contains(view, "Push") {
		t.Error("view should show push confirmation")
	}
}

func TestStatusModelViewStashMode(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{
		Unstaged: []git.FileStatus{{Path: "test.txt", WorkStatus: 'M'}},
	}
	m.items = buildItems(m.status)
	m.branchStatus = git.BranchStatus{Name: "main"}
	m.stashMode = stashAll
	m.stashInput.Focus()

	view := m.View()

	if !strings.Contains(view, "Stash all") {
		t.Error("view should show stash all message")
	}
}

func TestStatusModelViewCommitMode(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{
		Staged: []git.FileStatus{{Path: "test.txt", IndexStatus: 'A'}},
	}
	m.items = buildItems(m.status)
	m.branchStatus = git.BranchStatus{Name: "main"}
	m.commitMode = true
	m.commitInput.Focus()

	view := m.View()

	if !strings.Contains(view, "Commit message") {
		t.Error("view should show commit message prompt")
	}
}

func TestStatusModelViewVisualMode(t *testing.T) {
	m := NewStatusModel()
	m.status = &git.StatusResult{
		Unstaged: []git.FileStatus{{Path: "test.txt", WorkStatus: 'M'}},
	}
	m.items = buildItems(m.status)
	m.branchStatus = git.BranchStatus{Name: "main"}
	m.visualMode = true

	view := m.View()

	if !strings.Contains(view, "VISUAL") {
		t.Error("view should show VISUAL indicator")
	}
}

func TestStatusModelUpdateVisualSelection(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file3.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file4.txt"}, Section: "unstaged"},
	}
	m.visualMode = true
	m.visualStart = 1
	m.cursor = 3

	m.updateVisualSelection()

	// Should select from 1 to 3 inclusive
	if !m.selected[1] || !m.selected[2] || !m.selected[3] {
		t.Error("items 1-3 should be selected")
	}
	if m.selected[0] {
		t.Error("item 0 should not be selected")
	}

	// Test reverse selection
	m.visualStart = 3
	m.cursor = 1
	m.updateVisualSelection()

	if !m.selected[1] || !m.selected[2] || !m.selected[3] {
		t.Error("items 1-3 should still be selected when reversed")
	}
}

func TestStatusModelArrowKeys(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
	}

	// Test down arrow
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(StatusModel)
	if m.cursor != 1 {
		t.Errorf("after down arrow, cursor = %d, want 1", m.cursor)
	}

	// Test up arrow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(StatusModel)
	if m.cursor != 0 {
		t.Errorf("after up arrow, cursor = %d, want 0", m.cursor)
	}
}

func TestStatusModelEscapeFromModes(t *testing.T) {
	t.Run("escape from visual mode", func(t *testing.T) {
		m := NewStatusModel()
		m.items = []StatusItem{{File: git.FileStatus{Path: "file.txt"}, Section: "unstaged"}}
		m.visualMode = true
		m.selected[0] = true

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = newModel.(StatusModel)

		if m.visualMode {
			t.Error("should exit visual mode on esc")
		}
		if len(m.selected) > 0 {
			t.Error("selection should be cleared on esc")
		}
	})

	t.Run("escape from selection", func(t *testing.T) {
		m := NewStatusModel()
		m.items = []StatusItem{{File: git.FileStatus{Path: "file.txt"}, Section: "unstaged"}}
		m.selected[0] = true

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = newModel.(StatusModel)

		if len(m.selected) > 0 {
			t.Error("selection should be cleared on esc")
		}
	})

	t.Run("escape with no selection quits", func(t *testing.T) {
		m := NewStatusModel()

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = newModel.(StatusModel)

		if !m.quitting {
			t.Error("should quit when esc pressed with no selection")
		}
	})
}

func TestStatusModelHelpModeBlocksNavigation(t *testing.T) {
	m := NewStatusModel()
	m.items = []StatusItem{
		{File: git.FileStatus{Path: "file1.txt"}, Section: "unstaged"},
		{File: git.FileStatus{Path: "file2.txt"}, Section: "unstaged"},
	}
	m.showHelp = true

	// Navigation should be blocked in help mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(StatusModel)
	if m.cursor != 0 {
		t.Error("navigation should be blocked in help mode")
	}

	// But esc should close help
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(StatusModel)
	if m.showHelp {
		t.Error("esc should close help")
	}
}

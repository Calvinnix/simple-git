package ui

import (
	"go-on-git/internal/git"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const watchInterval = 500 * time.Millisecond

type tickMsg time.Time

type viewMode int

const (
	viewStatus   viewMode = iota
	viewFileDiff          // drill-down from status to file diff
	viewFullDiff          // full diff view (all hunks)
	viewBranches
	viewStashes
	viewStashDiff // drill-down from stashes to stash diff
	viewLog
)

// FileFilter specifies which hunks to show for a file
type FileFilter struct {
	Path       string
	ShowStaged bool
	Untracked  bool
}

// AppModel is the root model that manages views
type AppModel struct {
	mode         viewMode
	status       StatusModel
	diff         DiffModel
	branches     BranchesModel
	stashes      StashesModel
	log          LogModel
	currentFiles []FileFilter // files being viewed in diff mode
	width        int
	height       int
}

// NewAppModel creates a new app model starting in status view
func NewAppModel() AppModel {
	return NewAppModelWithOptions(false)
}

// NewAppModelWithOptions creates a new app model with options
func NewAppModelWithOptions(showHelp bool) AppModel {
	return AppModel{
		mode:     viewStatus,
		status:   NewStatusModelWithHelp(showHelp),
		branches: NewBranchesModel(),
		stashes:  NewStashesModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(m.status.Init(), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(watchInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to all views
		m.status.width = msg.Width
		m.status.height = msg.Height
		m.diff.width = msg.Width
		m.diff.height = msg.Height
		m.branches.width = msg.Width
		m.branches.height = msg.Height
		m.stashes.width = msg.Width
		m.stashes.height = msg.Height
		m.log.width = msg.Width
		m.log.height = msg.Height

	case tickMsg:
		// Only auto-refresh in status view when not in a blocking mode
		if m.mode == viewStatus && !m.status.isBlocking() {
			return m, tea.Batch(refreshStatus, tickCmd())
		}
		return m, tickCmd()

	case tea.KeyMsg:
		key := msg.String()

		// Ctrl+C always quits
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.mode {
		case viewStatus:
			// Skip navigation when in input modes
			if m.status.commitMode || m.status.stashMode != stashNone || m.status.confirmMode != confirmNone {
				break
			}
			// Handle navigation keys from status
			if key == Keys.FileDiff || key == Keys.Right || key == "right" || key == "enter" {
				// Enter file diff view for selected file(s)
				items := m.status.getSelectedItems()
				if len(items) > 0 {
					m.currentFiles = make([]FileFilter, len(items))
					for i, item := range items {
						m.currentFiles[i] = FileFilter{
							Path:       item.File.Path,
							ShowStaged: item.Section == "staged",
							Untracked:  item.Section == "untracked",
						}
					}
					m.diff = NewDiffModelWithFilters(m.currentFiles, m.width, m.height)
					m.mode = viewFileDiff
					return m, tea.Batch(tea.EnterAltScreen, m.diff.Init())
				}
				return m, nil
			} else if key == Keys.AllDiffs {
				// Enter full diff view
				m.diff = NewDiffModelWithSize(nil, m.width, m.height)
				m.mode = viewFullDiff
				return m, tea.Batch(tea.EnterAltScreen, m.diff.Init())
			} else if key == Keys.Branches {
				// Enter branches view
				m.branches = NewBranchesModelWithOptions(m.status.showVerboseHelp)
				m.branches.width = m.width
				m.branches.height = m.height
				m.mode = viewBranches
				return m, tea.Batch(tea.EnterAltScreen, m.branches.Init())
			} else if key == Keys.Stashes {
				// Enter stashes view
				m.stashes = NewStashesModelWithOptions(m.status.showVerboseHelp)
				m.stashes.width = m.width
				m.stashes.height = m.height
				m.mode = viewStashes
				return m, tea.Batch(tea.EnterAltScreen, m.stashes.Init())
			} else if key == Keys.Log {
				// Enter log view
				m.log = NewLogModelWithOptions(m.width, m.height, m.status.showVerboseHelp)
				m.mode = viewLog
				return m, tea.Batch(tea.EnterAltScreen, m.log.Init())
			}

		case viewFileDiff:
			// Handle back navigation from file diff
			if key == Keys.Left || key == "left" || key == "esc" {
				inHunkDetail := m.diff.IsViewingHunk()
				if !inHunkDetail || (len(m.diff.hunks) == 1 && !m.diff.showHelp && !m.diff.confirmMode) {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}

		case viewFullDiff:
			// Handle back navigation from full diff
			if key == Keys.Left || key == "left" || key == "esc" {
				inHunkDetail := m.diff.IsViewingHunk()
				if !inHunkDetail || (len(m.diff.hunks) == 1 && !m.diff.showHelp && !m.diff.confirmMode) {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}

		case viewBranches:
			// Handle back navigation from branches
			if key == Keys.Left || key == "left" || key == "esc" {
				if !m.branches.showHelp && !m.branches.deleteConfirmMode && !m.branches.inputMode && !m.branches.forceDeleteMode {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}
			// Override quit to go back instead
			if key == Keys.Quit {
				if !m.branches.showHelp && !m.branches.deleteConfirmMode && !m.branches.inputMode && !m.branches.forceDeleteMode {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}

		case viewStashes:
			// Handle drill-down to stash diff
			if key == Keys.Right || key == "right" {
				if len(m.stashes.stashes) > 0 && m.stashes.cursor < len(m.stashes.stashes) {
					if !m.stashes.showHelp && !m.stashes.confirmMode {
						stash := m.stashes.stashes[m.stashes.cursor]
						m.stashes.diffModel = NewStashDiffModel(m.width, m.height)
						m.mode = viewStashDiff
						return m, func() tea.Msg {
							diff, err := git.GetStashDiff(stash.Index)
							if err != nil {
								return errMsg{err}
							}
							return stashDiffMsg{diff}
						}
					}
				}
				return m, nil
			}
			// Handle back navigation from stashes
			if key == Keys.Left || key == "left" || key == "esc" {
				if !m.stashes.showHelp && !m.stashes.confirmMode {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}
			// Override quit to go back instead
			if key == Keys.Quit {
				if !m.stashes.showHelp && !m.stashes.confirmMode {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}

		case viewStashDiff:
			// Handle back navigation from stash diff
			if key == Keys.Left || key == "left" || key == "esc" {
				if !m.stashes.diffModel.showHelp {
					if m.stashes.diffModel.viewingHunk {
						// Exit hunk detail first
						m.stashes.diffModel.viewingHunk = false
						m.stashes.diffModel.scrollOffset = 0
						return m, nil
					}
					m.mode = viewStashes
					return m, nil
				}
			}
			// Override quit to go back
			if key == Keys.Quit {
				if !m.stashes.diffModel.showHelp {
					m.mode = viewStashes
					return m, nil
				}
			}

		case viewLog:
			// Handle back navigation from log
			if key == Keys.Left || key == "left" || key == "esc" || key == Keys.Quit || key == Keys.Log {
				if !m.log.showHelp {
					m.mode = viewStatus
					return m, tea.Batch(tea.ExitAltScreen, refreshStatus)
				}
			}
		}
	}

	// Delegate to current view
	return m.updateCurrentView(msg)
}

func (m AppModel) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case viewFileDiff, viewFullDiff:
		newDiff, cmd := m.diff.Update(msg)
		m.diff = newDiff.(DiffModel)
		return m, cmd
	case viewBranches:
		newBranches, cmd := m.branches.Update(msg)
		m.branches = newBranches.(BranchesModel)
		return m, cmd
	case viewStashes:
		newStashes, cmd := m.stashes.Update(msg)
		m.stashes = newStashes.(StashesModel)
		return m, cmd
	case viewStashDiff:
		newDiff, cmd := m.stashes.diffModel.Update(msg)
		m.stashes.diffModel = newDiff.(StashDiffModel)
		return m, cmd
	case viewLog:
		newLog, cmd := m.log.Update(msg)
		m.log = newLog.(LogModel)
		return m, cmd
	default:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(StatusModel)
		return m, cmd
	}
}

func (m AppModel) View() string {
	switch m.mode {
	case viewFileDiff, viewFullDiff:
		return m.diff.View()
	case viewBranches:
		return m.branches.View()
	case viewStashes:
		return m.stashes.View()
	case viewStashDiff:
		return m.stashes.diffModel.View()
	case viewLog:
		return m.log.View()
	default:
		return m.status.View()
	}
}

// Shared message types
type statusMsg struct {
	status       *git.StatusResult
	branchStatus git.BranchStatus
}

type errMsg struct {
	err error
}

type diffMsg struct {
	diff *git.DiffResult
}

type combinedDiffMsg struct {
	diff *git.CombinedDiffResult
}

type branchesMsg struct {
	branches []git.Branch
}

type branchDeleteFailedMsg struct {
	branchName string
	err        error
}

type stashesMsg struct {
	stashes []git.Stash
}

type stashDiffMsg struct {
	diff *git.CombinedDiffResult
}

func refreshStatus() tea.Msg {
	status, err := git.GetStatus()
	if err != nil {
		return errMsg{err}
	}
	branchStatus := git.GetBranchStatus()
	return statusMsg{status, branchStatus}
}

package ui

import (
	"fmt"
	"strings"

	"simple-git/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// LogModel is the bubbletea model for the log view
type LogModel struct {
	lines        []string
	scrollOffset int
	err          error
	width        int
	height       int
}

// NewLogModel creates a new log model
func NewLogModel() LogModel {
	return LogModel{}
}

// NewLogModelWithSize creates a new log model with dimensions
func NewLogModelWithSize(width, height int) LogModel {
	return LogModel{
		width:  width,
		height: height,
	}
}

type logMsg struct {
	content string
}

func refreshLog() tea.Msg {
	content, err := git.GetLog(100)
	if err != nil {
		return errMsg{err}
	}
	return logMsg{content}
}

// Init initializes the model
func (m LogModel) Init() tea.Cmd {
	return refreshLog
}

// Update handles messages
func (m LogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		visibleLines := m.height - 2
		if visibleLines < 1 {
			visibleLines = 10
		}
		maxOffset := len(m.lines) - visibleLines
		if maxOffset < 0 {
			maxOffset = 0
		}

		switch key {
		case "j", "down":
			m.scrollOffset = min(m.scrollOffset+1, maxOffset)
			return m, nil
		case "k", "up":
			m.scrollOffset = max(m.scrollOffset-1, 0)
			return m, nil
		case "G":
			m.scrollOffset = maxOffset
			return m, nil
		case "g":
			m.scrollOffset = 0
			return m, nil
		case "ctrl+d":
			m.scrollOffset = min(m.scrollOffset+visibleLines/2, maxOffset)
			return m, nil
		case "ctrl+u":
			m.scrollOffset = max(m.scrollOffset-visibleLines/2, 0)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case logMsg:
		m.lines = strings.Split(msg.content, "\n")
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

// View renders the log view
func (m LogModel) View() string {
	var content strings.Builder

	if m.err != nil {
		content.WriteString(StyleUnstaged.Render(fmt.Sprintf("Error: %v", m.err)))
		content.WriteString("\n")
		return m.anchorBottom(content.String())
	}

	if len(m.lines) == 0 {
		content.WriteString(StyleMuted.Render("Loading..."))
		content.WriteString("\n")
		return m.anchorBottom(content.String())
	}

	visibleLines := m.height - 2
	if visibleLines < 1 {
		visibleLines = 20
	}

	endIdx := m.scrollOffset + visibleLines
	if endIdx > len(m.lines) {
		endIdx = len(m.lines)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		line := m.lines[i]
		if strings.HasPrefix(line, "commit ") {
			content.WriteString(StyleStaged.Render(line))
		} else if strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") {
			content.WriteString(StyleMuted.Render(line))
		} else {
			content.WriteString(line)
		}
		content.WriteString("\n")
	}

	return m.anchorBottom(content.String())
}

func (m LogModel) anchorBottom(content string) string {
	lines := strings.Count(content, "\n")
	if m.height <= lines {
		return content
	}
	padding := m.height - lines - 1
	return strings.Repeat("\n", padding) + content
}

package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Using ANSI color numbers for broad terminal compatibility
	colorGreen  = lipgloss.Color("2") // Green
	colorRed    = lipgloss.Color("1") // Red
	colorYellow = lipgloss.Color("3") // Yellow
	colorBlue   = lipgloss.Color("4") // Blue
	colorGray   = lipgloss.Color("8") // Bright black / gray

	// Base styles
	StyleNormal = lipgloss.NewStyle()
	StyleMuted  = lipgloss.NewStyle().Foreground(colorGray)

	// File status styles
	StyleStaged    = lipgloss.NewStyle().Foreground(colorGreen)
	StyleUnstaged  = lipgloss.NewStyle().Foreground(colorRed)
	StyleUntracked = lipgloss.NewStyle().Foreground(colorYellow)

	// Selection styles
	StyleSelected = lipgloss.NewStyle().
			Background(colorGray).
			Bold(true)

	// Visual mode selection (vim-like)
	StyleVisual = lipgloss.NewStyle().
			Background(colorGray).
			Foreground(colorBlue)

	// Section headers
	StyleSectionHeader = lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true)

	// Diff styles
	StyleDiffAdded         = lipgloss.NewStyle().Foreground(colorGreen)
	StyleDiffRemoved       = lipgloss.NewStyle().Foreground(colorRed)
	StyleDiffContext       = lipgloss.NewStyle().Foreground(colorGray)
	StyleDiffHeader        = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	StyleHunkHeaderStaged   = lipgloss.NewStyle().Foreground(colorGreen)
	StyleHunkHeaderUnstaged = lipgloss.NewStyle().Foreground(colorRed)

	// Help styles
	StyleHelpKey   = lipgloss.NewStyle().Foreground(colorYellow)
	StyleHelpDesc  = lipgloss.NewStyle().Foreground(colorGray)
	StyleHelpTitle = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)

	// Status bar
	StyleStatusBar = lipgloss.NewStyle().Foreground(colorGray)

	// Confirm dialog
	StyleConfirm = lipgloss.NewStyle().Foreground(colorRed).Bold(true)

	// Empty state
	StyleEmpty = lipgloss.NewStyle()
)

// StatusChar returns the styled status word for display based on the section
func StatusChar(indexStatus, workStatus byte, section string) string {
	return StatusCharStyled(indexStatus, workStatus, section, StyleNormal)
}

// StatusCharStyled returns the status word with extra styling (e.g., selection highlight).
func StatusCharStyled(indexStatus, workStatus byte, section string, extra lipgloss.Style) string {
	var word string
	var style lipgloss.Style

	switch section {
	case "staged":
		word = indexStatusWord(indexStatus)
		style = StyleStaged
	case "unstaged":
		word = workStatusWord(workStatus)
		style = StyleUnstaged
	case "untracked":
		// No status prefix for untracked files (like git status)
		return ""
	default:
		return ""
	}

	// Pad to 12 chars for alignment (like git status)
	padded := fmt.Sprintf("%-12s", word)
	return style.Inherit(extra).Render(padded)
}

func indexStatusWord(status byte) string {
	switch status {
	case 'A':
		return "new file:"
	case 'M':
		return "modified:"
	case 'D':
		return "deleted:"
	case 'R':
		return "renamed:"
	case 'C':
		return "copied:"
	default:
		return string(status)
	}
}

func workStatusWord(status byte) string {
	switch status {
	case 'M':
		return "modified:"
	case 'D':
		return "deleted:"
	default:
		return string(status)
	}
}

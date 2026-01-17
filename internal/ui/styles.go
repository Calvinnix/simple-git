package ui

import "github.com/charmbracelet/lipgloss"

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
	StyleEmpty = lipgloss.NewStyle().Foreground(colorGray).Italic(true)
)

// StatusChar returns the styled status character for display
func StatusChar(indexStatus, workStatus byte) string {
	return StatusCharStyled(indexStatus, workStatus, StyleNormal)
}

// StatusCharStyled returns the status character with extra styling (e.g., selection highlight).
func StatusCharStyled(indexStatus, workStatus byte, extra lipgloss.Style) string {
	idx := string(indexStatus)
	work := string(workStatus)

	if indexStatus == '?' {
		return StyleUntracked.Inherit(extra).Render("??")
	}

	idxStyle := StyleNormal
	if indexStatus != ' ' {
		idxStyle = StyleStaged
	}

	workStyle := StyleNormal
	if workStatus != ' ' {
		workStyle = StyleUnstaged
	}

	return idxStyle.Inherit(extra).Render(idx) + workStyle.Inherit(extra).Render(work)
}

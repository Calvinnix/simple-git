package main

import (
	"fmt"
	"os"

	"simple-git/internal/git"
	"simple-git/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.3.0"

func main() {
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "fatal: not a git repository")
		os.Exit(1)
	}

	args := os.Args[1:]
	showHelp := true

	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		case "--version", "-v":
			fmt.Printf("simple-git version %s\n", version)
			os.Exit(0)
		case "--hide-help":
			showHelp = false
		default:
			fmt.Fprintf(os.Stderr, "unknown option: %s\n", arg)
			printHelp()
			os.Exit(1)
		}
	}

	model := ui.NewAppModelWithOptions(showHelp)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`simple-git - Lightweight Git TUI

Usage:
  simple-git [options]

Options:
  --hide-help   Start with help bar hidden
  -h, --help    Show this help message
  -v, --version Show version

Navigation:
  l/→         View file diff (from status)
  i           View all diffs (staged and unstaged)
  b           View branches
  e           View stashes
  o           View commit log
  h/←/ESC     Go back

Key Bindings:
  j/k, ↑/↓    Move down/up
  gg/G        Go to top/bottom
  v           Visual mode
  SPACE       Stage/unstage file or hunk
  a/A         Stage file(s) / Stage all
  u/U         Unstage file(s) / Unstage all
  s/S         Stash file(s) / Stash all
  d           Discard/delete (with confirmation)
  c           Commit staged changes
  p           Push commits
  n           Create new branch (in branches view)
  ?           Toggle quick help
  /           Toggle verbose help
  q/ESC       Quit`)
}

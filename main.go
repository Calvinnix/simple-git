package main

import (
	"fmt"
	"os"
	"strings"

	"go-on-git/internal/git"
	"go-on-git/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.6.0"

func main() {
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "fatal: not a git repository")
		os.Exit(1)
	}

	args := os.Args[1:]
	showHelp := true

	for _, arg := range args {
		switch {
		case arg == "--help" || arg == "-h":
			printHelp()
			os.Exit(0)
		case arg == "--version" || arg == "-v":
			fmt.Printf("go-on-git version %s\n", version)
			os.Exit(0)
		case arg == "--hide-help":
			showHelp = false
		case strings.HasPrefix(arg, "--key."):
			// Parse keymap override: --key.action=key
			override := strings.TrimPrefix(arg, "--key.")
			action, key, valid := ui.ParseKeymapArg(override)
			if !valid {
				fmt.Fprintf(os.Stderr, "invalid keymap format: %s\n", arg)
				fmt.Fprintln(os.Stderr, "expected format: --key.action=key")
				os.Exit(1)
			}
			if !ui.Keys.ApplyOverride(action, key) {
				fmt.Fprintf(os.Stderr, "unknown keymap action: %s\n", action)
				fmt.Fprintf(os.Stderr, "available actions: %s\n", strings.Join(ui.ListKeymapActions(), ", "))
				os.Exit(1)
			}
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
	fmt.Println(`go-on-git - Lightweight Git TUI

Usage:
  go-on-git [options]

Options:
  --hide-help         Start with help bar hidden
  --key.action=key    Override a key binding (see below)
  -h, --help          Show this help message
  -v, --version       Show version

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
  c/C         Commit inline / with editor
  p           Push commits
  n           Create new branch (in branches view)
  ?           Toggle quick help
  /           Toggle verbose help
  q/ESC       Quit

Keymap Overrides:
  Override default keys with --key.action=key
  Example: --key.down=n --key.up=e --key.commit=w

  Available actions:
    up, down, left, right, top, bottom, select, back, quit,
    stage, stage-all, unstage, unstage-all, discard,
    commit, commit-edit, push, stash, stash-all,
    file-diff, all-diffs, branches, stashes, log,
    visual, help, verbose-help, new-branch, delete`)
}

# simple-git

A lightweight terminal user interface for Git, built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Requirements

- Go 1.24 or later
- Git

## Installation

Clone the repository and build:

```bash
git clone git@github.com:Calvinnix/simple-git.git
cd simple-git
go build .
```

Move the binary to a directory in your PATH:

```bash
mv simple-git ~/.local/bin/
```

### Setting up an alias

For convenience, add an alias to your shell configuration (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
alias g='simple-git'
```

Then reload your shell or run `source ~/.bashrc` (or equivalent).

## Usage

Run from within a Git repository:

```bash
simple-git             # Interactive status view
simple-git --hide-help # Start with help bar hidden
simple-git --help      # Show help
simple-git --version   # Show version
```

Or with the alias:

```bash
g      # Interactive status view
```

## Views

simple-git has multiple views you can navigate between:

- **Status View** (default) - Stage/unstage files, commit, push
- **Diff View** - View and stage/unstage individual hunks
- **Branches View** - Switch, create, and delete branches
- **Stashes View** - Apply, pop, and drop stashes
- **Log View** - Browse commit history

## Key Bindings

### Status View (Main)

| Key | Action |
|-----|--------|
| `j/k`, `↑/↓` | Move up/down |
| `gg/G` | Go to top/bottom |
| `h/←` | Toggle select file (multi-select) |
| `v/V` | Visual mode (contiguous selection) |
| `SPACE` | Stage/unstage file |
| `a/A` | Stage file(s) / Stage all |
| `u/U` | Unstage file(s) / Unstage all |
| `s/S` | Stash file(s) / Stash all |
| `d` | Discard change (with confirmation) |
| `c` | Commit staged changes |
| `p` | Push commits |
| `l/→` | View file diff |
| `i` | View all diffs |
| `b` | View branches |
| `e` | View stashes |
| `o` | View commit log |
| `?` | Toggle quick help overlay |
| `/` | Toggle verbose help bar |
| `q/ESC` | Quit |

### Diff View

| Key | Action |
|-----|--------|
| `j/k`, `↑/↓` | Navigate hunks / scroll |
| `gg/G` | Go to top/bottom |
| `l/→` | View hunk detail (scrollable) |
| `h/←/ESC` | Go back |
| `SPACE` | Toggle stage/unstage hunk |
| `a` | Stage hunk |
| `u` | Unstage hunk |
| `d` | Discard hunk (unstaged only) |
| `?` | Toggle help |
| `q` | Quit |

### Branches View

| Key | Action |
|-----|--------|
| `j/k`, `↑/↓` | Move up/down |
| `gg/G` | Go to top/bottom |
| `Enter/l` | Checkout branch |
| `n` | Create new branch |
| `d` | Delete branch (with confirmation) |
| `?` | Toggle help |
| `h/←/ESC/q` | Go back |

### Stashes View

| Key | Action |
|-----|--------|
| `j/k`, `↑/↓` | Move up/down |
| `gg/G` | Go to top/bottom |
| `l/→` | View stash diff |
| `a` | Apply stash (keep in list) |
| `p` | Pop stash (apply and remove) |
| `d` | Drop stash (delete) |
| `?` | Toggle help |
| `h/←/ESC/q` | Go back |

### Log View

| Key | Action |
|-----|--------|
| `j/k`, `↑/↓` | Scroll up/down |
| `g/G` | Go to top/bottom |
| `ctrl+d/ctrl+u` | Page down/up |
| `h/←/ESC/q/o` | Go back |

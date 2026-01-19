# simple-git

A lightweight terminal user interface for Git, built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Installation

### Homebrew

```bash
brew install Calvinnix/tap/simple-git
```

## Usage

Run from within a Git repository:

```bash
simple-git             # Interactive status view
simple-git --hide-help # Start with help bar hidden
simple-git --help      # Show help
simple-git --version   # Show version
```

### Setting up an alias

For convenience, add an alias to your shell configuration (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
alias g='simple-git'
```

Then reload your shell or run `source ~/.bashrc` (or equivalent).

## Views

simple-git has multiple views you can navigate between:

- **Status View** (default) - Stage/unstage files, commit, push
- **Diff View** - View and stage/unstage individual hunks
- **Branches View** - Switch, create, and delete branches
- **Stashes View** - Apply, pop, and drop stashes
- **Log View** - Browse commit history

## Default Keymaps

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `gg` | Go to top |
| `G` | Go to bottom |
| `h` / `←` | Select item / Go back |
| `l` / `→` / `Enter` | View diff / Drill down |
| `q` / `ESC` | Quit / Go back |

### Views

| Key | Action |
|-----|--------|
| `l` | File diff (from status) |
| `i` | All diffs (staged and unstaged) |
| `b` | Branches |
| `e` | Stashes |
| `o` | Commit log |

### Actions

| Key | Action |
|-----|--------|
| `Space` | Stage/unstage file or hunk |
| `a` | Stage selected file(s) |
| `A` | Stage all |
| `u` | Unstage selected file(s) |
| `U` | Unstage all |
| `d` | Discard changes (with confirmation) |
| `c` | Commit with inline message |
| `C` | Commit with editor |
| `p` | Push commits |
| `s` | Stash selected file(s) |
| `S` | Stash all |

### Other

| Key | Action |
|-----|--------|
| `v` | Visual mode (select multiple) |
| `?` | Toggle quick help |
| `/` | Toggle verbose help |
| `n` | New branch (in branches view) |

## Custom Keymaps

You can override default key bindings using command line arguments:

```bash
simple-git --key.action=key
```

### Available Actions

| Action | Default | Description |
|--------|---------|-------------|
| `up` | `k` | Move cursor up |
| `down` | `j` | Move cursor down |
| `left` | `h` | Go back / Select |
| `right` | `l` | Drill down |
| `top` | `g` | Go to top (press twice: gg) |
| `bottom` | `G` | Go to bottom |
| `quit` | `q` | Quit |
| `stage` | `a` | Stage file(s) |
| `stage-all` | `A` | Stage all |
| `unstage` | `u` | Unstage file(s) |
| `unstage-all` | `U` | Unstage all |
| `discard` | `d` | Discard changes |
| `commit` | `c` | Commit inline |
| `commit-edit` | `C` | Commit with editor |
| `push` | `p` | Push |
| `stash` | `s` | Stash file(s) |
| `stash-all` | `S` | Stash all |
| `file-diff` | `l` | View file diff |
| `all-diffs` | `i` | View all diffs |
| `branches` | `b` | View branches |
| `stashes` | `e` | View stashes |
| `log` | `o` | View log |
| `visual` | `v` | Visual mode |
| `help` | `?` | Quick help |
| `verbose-help` | `/` | Verbose help |
| `new-branch` | `n` | Create branch |
| `delete` | `d` | Delete |

### Examples

```bash
# Colemak layout
simple-git --key.up=e --key.down=n --key.left=m --key.right=i

# Dvorak-friendly
simple-git --key.up=c --key.down=t

# Vim muscle memory alternatives
simple-git --key.quit=ZZ

# Multiple overrides
simple-git --key.commit=w --key.push=P --key.quit=x
```

### Shell Alias with Custom Keys

Add to your shell configuration for persistent custom keymaps:

```bash
# ~/.bashrc or ~/.zshrc
alias g='simple-git --key.up=e --key.down=n'
```

package ui

import "strings"

// Keymap holds all configurable key bindings
type Keymap struct {
	// Navigation
	Up     string
	Down   string
	Left   string
	Right  string
	Top    string
	Bottom string
	Select string
	Back   string
	Quit   string

	// Actions
	Stage      string
	StageAll   string
	Unstage    string
	UnstageAll string
	Discard    string
	Commit     string
	CommitEdit string
	Push       string
	Stash      string
	StashAll   string

	// Views
	FileDiff string
	AllDiffs string
	FullDiff string
	Branches string
	Stashes  string
	Log      string

	// Modes
	Visual      string
	Help        string
	VerboseHelp string
	NewBranch   string
	Delete      string
}

type keymapBinding struct {
	action string
	key    func(*Keymap) *string
}

var keymapBindings = []keymapBinding{
	{action: "up", key: func(k *Keymap) *string { return &k.Up }},
	{action: "down", key: func(k *Keymap) *string { return &k.Down }},
	{action: "left", key: func(k *Keymap) *string { return &k.Left }},
	{action: "right", key: func(k *Keymap) *string { return &k.Right }},
	{action: "top", key: func(k *Keymap) *string { return &k.Top }},
	{action: "bottom", key: func(k *Keymap) *string { return &k.Bottom }},
	{action: "select", key: func(k *Keymap) *string { return &k.Select }},
	{action: "back", key: func(k *Keymap) *string { return &k.Back }},
	{action: "quit", key: func(k *Keymap) *string { return &k.Quit }},
	{action: "stage", key: func(k *Keymap) *string { return &k.Stage }},
	{action: "stage-all", key: func(k *Keymap) *string { return &k.StageAll }},
	{action: "unstage", key: func(k *Keymap) *string { return &k.Unstage }},
	{action: "unstage-all", key: func(k *Keymap) *string { return &k.UnstageAll }},
	{action: "discard", key: func(k *Keymap) *string { return &k.Discard }},
	{action: "commit", key: func(k *Keymap) *string { return &k.Commit }},
	{action: "commit-edit", key: func(k *Keymap) *string { return &k.CommitEdit }},
	{action: "push", key: func(k *Keymap) *string { return &k.Push }},
	{action: "stash", key: func(k *Keymap) *string { return &k.Stash }},
	{action: "stash-all", key: func(k *Keymap) *string { return &k.StashAll }},
	{action: "file-diff", key: func(k *Keymap) *string { return &k.FileDiff }},
	{action: "all-diffs", key: func(k *Keymap) *string { return &k.AllDiffs }},
	{action: "full-diff", key: func(k *Keymap) *string { return &k.FullDiff }},
	{action: "branches", key: func(k *Keymap) *string { return &k.Branches }},
	{action: "stashes", key: func(k *Keymap) *string { return &k.Stashes }},
	{action: "log", key: func(k *Keymap) *string { return &k.Log }},
	{action: "visual", key: func(k *Keymap) *string { return &k.Visual }},
	{action: "help", key: func(k *Keymap) *string { return &k.Help }},
	{action: "verbose-help", key: func(k *Keymap) *string { return &k.VerboseHelp }},
	{action: "new-branch", key: func(k *Keymap) *string { return &k.NewBranch }},
	{action: "delete", key: func(k *Keymap) *string { return &k.Delete }},
}

// DefaultKeymap returns the default key bindings
func DefaultKeymap() *Keymap {
	return &Keymap{
		// Navigation
		Up:     "k",
		Down:   "j",
		Left:   "h",
		Right:  "l",
		Top:    "g",
		Bottom: "G",
		Select: "h",
		Back:   "h",
		Quit:   "q",

		// Actions
		Stage:      "a",
		StageAll:   "A",
		Unstage:    "u",
		UnstageAll: "U",
		Discard:    "d",
		Commit:     "c",
		CommitEdit: "C",
		Push:       "p",
		Stash:      "s",
		StashAll:   "S",

		// Views
		FileDiff: "l",
		AllDiffs: "i",
		FullDiff: "f",
		Branches: "b",
		Stashes:  "e",
		Log:      "o",

		// Modes
		Visual:      "v",
		Help:        "?",
		VerboseHelp: "/",
		NewBranch:   "n",
		Delete:      "d",
	}
}

// ParseKeymapArg parses a keymap override argument in the format "action=key"
// Returns the action name and key, or empty strings if invalid
func ParseKeymapArg(arg string) (action, key string, valid bool) {
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ApplyOverride applies a single keymap override
func (k *Keymap) ApplyOverride(action, key string) bool {
	for _, binding := range keymapBindings {
		if binding.action == action {
			*binding.key(k) = key
			return true
		}
	}
	return false
}

// ListKeymapActions returns all available action names for help text
func ListKeymapActions() []string {
	actions := make([]string, 0, len(keymapBindings))
	for _, binding := range keymapBindings {
		actions = append(actions, binding.action)
	}
	return actions
}

// KeymapConflict describes a key that maps to more than one action.
type KeymapConflict struct {
	Key     string
	Actions []string
}

// FindKeymapOverrideConflicts reports key conflicts introduced relative to defaults.
func FindKeymapOverrideConflicts(defaults, current *Keymap) []KeymapConflict {
	defaultKeyActions := make(map[string]map[string]bool, len(keymapBindings))
	for _, binding := range keymapBindings {
		key := *binding.key(defaults)
		if defaultKeyActions[key] == nil {
			defaultKeyActions[key] = make(map[string]bool)
		}
		defaultKeyActions[key][binding.action] = true
	}

	currentKeyActions := make(map[string][]string, len(keymapBindings))
	for _, binding := range keymapBindings {
		key := *binding.key(current)
		currentKeyActions[key] = append(currentKeyActions[key], binding.action)
	}

	seen := make(map[string]bool, len(keymapBindings))
	var conflicts []KeymapConflict
	for _, binding := range keymapBindings {
		key := *binding.key(current)
		if seen[key] {
			continue
		}
		seen[key] = true
		actions := currentKeyActions[key]
		if len(actions) < 2 {
			continue
		}
		defaultActions := defaultKeyActions[key]
		conflict := false
		for _, action := range actions {
			if defaultActions == nil || !defaultActions[action] {
				conflict = true
				break
			}
		}
		if conflict {
			conflicts = append(conflicts, KeymapConflict{Key: key, Actions: actions})
		}
	}

	return conflicts
}

// Global keymap instance
var Keys = DefaultKeymap()

package ui

import "strings"

func formatKeyList(keys ...string) string {
	seen := make(map[string]bool, len(keys))
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		key = formatKeyLabel(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		parts = append(parts, key)
	}
	return strings.Join(parts, "/")
}

func formatDoubleKey(key string) string {
	key = formatKeyLabel(key)
	if key == "" {
		return ""
	}
	if len([]rune(key)) == 1 {
		return key + key
	}
	return key + " " + key
}

func formatKeyLabel(key string) string {
	switch key {
	case "esc":
		return "ESC"
	case "enter":
		return "Enter"
	case " ":
		return "SPACE"
	default:
		return key
	}
}

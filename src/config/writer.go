package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// UpdateFile rewrites scalar values in a YAML config while keeping comments,
// key order, and unrelated keys untouched. Keys are dotted paths such as
// "server.port" or "supervisor.restart.max_delay". A key that is missing from
// the file is appended to its section (the section itself must already exist).
func UpdateFile(path string, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件: %w", err)
	}

	lines := strings.Split(string(raw), "\n")
	pending := make(map[string]string, len(updates))
	maps.Copy(pending, updates)

	// path stack of (indent, key) pairs describing the current YAML position.
	type level struct {
		indent int
		key    string
	}
	var stack []level
	// Track the last line index belonging to each section, so a missing key can
	// be appended in the right place.
	sectionEnd := map[string]int{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		name, rest, found := strings.Cut(trimmed, ":")
		if !found {
			continue
		}
		name = strings.TrimSpace(name)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		prefix := ""
		if len(stack) > 0 {
			parts := make([]string, len(stack))
			for j, l := range stack {
				parts[j] = l.key
			}
			prefix = strings.Join(parts, ".") + "."
		}
		full := prefix + name

		if strings.TrimSpace(rest) == "" {
			// A mapping key: descend into it.
			stack = append(stack, level{indent: indent, key: name})
			continue
		}
		for _, ancestor := range ancestorsOf(full) {
			sectionEnd[ancestor] = i
		}
		if value, ok := pending[full]; ok {
			lines[i] = strings.Repeat(" ", indent) + name + ": " + formatYAMLValue(value) + inlineComment(rest)
			delete(pending, full)
		}
	}

	// Append keys that were not present in the file.
	for _, key := range sortedKeys(pending) {
		section, name, found := cutLast(key, ".")
		if !found {
			lines = append(lines, key+": "+formatYAMLValue(pending[key]))
			continue
		}
		at, ok := sectionEnd[section]
		if !ok {
			return fmt.Errorf("配置中不存在 %s 段落，无法写入 %s", section, key)
		}
		indent := strings.Count(section, ".")*2 + 2
		insert := strings.Repeat(" ", indent) + name + ": " + formatYAMLValue(pending[key])
		lines = append(lines[:at+1], append([]string{insert}, lines[at+1:]...)...)
		// Every recorded position after the insert shifts down by one.
		for sec, idx := range sectionEnd {
			if idx > at {
				sectionEnd[sec] = idx + 1
			}
		}
	}

	return writeFileAtomic(path, []byte(strings.Join(lines, "\n")))
}

func ancestorsOf(full string) []string {
	parts := strings.Split(full, ".")
	ancestors := make([]string, 0, len(parts))
	for i := 1; i < len(parts); i++ {
		ancestors = append(ancestors, strings.Join(parts[:i], "."))
	}
	return ancestors
}

func cutLast(s, sep string) (before, after string, found bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// inlineComment keeps a trailing comment that followed the old value,
// including the exact whitespace that separated it.
func inlineComment(rest string) string {
	hash := strings.Index(rest, "#")
	if hash <= 0 {
		return ""
	}
	start := hash
	for start > 0 && (rest[start-1] == ' ' || rest[start-1] == '\t') {
		start--
	}
	if start == hash {
		// "#" glued to the value is part of the value, not a comment.
		return ""
	}
	return rest[start:]
}

// formatYAMLValue quotes anything that isn't an unambiguous scalar.
func formatYAMLValue(value string) string {
	if value == "" {
		return `""`
	}
	if value == "true" || value == "false" {
		return value
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

// writeFileAtomic avoids leaving a truncated config behind if writing fails.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return fmt.Errorf("创建临时配置文件: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("写入临时配置文件: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("同步临时配置文件: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时配置文件: %w", err)
	}
	if info, err := os.Stat(path); err == nil {
		_ = os.Chmod(tmpName, info.Mode())
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("替换配置文件: %w", err)
	}
	return nil
}

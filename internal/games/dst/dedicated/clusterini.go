package dedicated

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

const maxClusterINIBytes = 1024 * 1024

func ConsoleEnabled(clusterINI string) bool {
	body, err := os.ReadFile(filepath.Clean(clusterINI))
	if err != nil || len(body) > maxClusterINIBytes {
		return false
	}
	section := ""
	for _, raw := range strings.Split(string(body), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		if section != "misc" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "console_enabled") {
			return strings.EqualFold(strings.TrimSpace(parts[1]), "true")
		}
	}
	return false
}

// EnsureConsoleEnabled migrates AGMP away from DST's deprecated -console
// argument while preserving stdin console commands. It only touches the
// [MISC] console_enabled key in the selected Cluster's cluster.ini.
func EnsureConsoleEnabled(clusterINI string) error {
	clusterINI = filepath.Clean(strings.TrimSpace(clusterINI))
	if clusterINI == "." || clusterINI == "" {
		return errors.New("cluster.ini path is empty")
	}
	info, err := os.Stat(clusterINI)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("cluster.ini 不可用: %w", err)
	}
	if info.Size() > maxClusterINIBytes {
		return errors.New("cluster.ini 异常过大")
	}
	body, err := os.ReadFile(clusterINI)
	if err != nil {
		return err
	}
	if ConsoleEnabled(clusterINI) {
		return nil
	}

	newline := "\n"
	if bytes.Contains(body, []byte("\r\n")) {
		newline = "\r\n"
	}
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	miscStart, nextSection, keyLine := -1, len(lines), -1
	section := ""
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			newSection := strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if section == "misc" && nextSection == len(lines) {
				nextSection = i
			}
			section = newSection
			if section == "misc" && miscStart < 0 {
				miscStart = i
			}
			continue
		}
		if section == "misc" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "console_enabled") {
				keyLine = i
			}
		}
	}
	if keyLine >= 0 {
		lines[keyLine] = "console_enabled = true"
	} else if miscStart >= 0 {
		insertAt := nextSection
		lines = append(lines, "")
		copy(lines[insertAt+1:], lines[insertAt:])
		lines[insertAt] = "console_enabled = true"
	} else {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "[MISC]", "console_enabled = true")
	}
	updated := strings.Join(lines, newline)
	tmp, err := os.CreateTemp(filepath.Dir(clusterINI), ".cluster-ini-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(updated); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return platformfiles.AtomicReplace(tmpName, clusterINI)
}

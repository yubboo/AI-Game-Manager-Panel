package steam

import (
	"regexp"
	"strconv"
	"strings"
)

var quotedPairPattern = regexp.MustCompile(`^\s*"((?:\\.|[^"])*)"\s+"((?:\\.|[^"])*)"\s*$`)

func parseLibraryFolderPaths(content string) []string {
	paths := []string{}
	for _, line := range strings.Split(content, "\n") {
		match := quotedPairPattern.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if len(match) != 3 {
			continue
		}
		key := unescapeVDF(match[1])
		value := unescapeVDF(match[2])
		if strings.EqualFold(key, "path") {
			paths = append(paths, value)
			continue
		}
		// Older Steam libraryfolders.vdf files stored additional libraries as
		// direct numeric key/value pairs, for example: "1" "D:\\SteamLibrary".
		// Modern nested app IDs are also numeric, so only accept values that
		// actually look like filesystem paths.
		if _, err := strconv.Atoi(key); err == nil && looksLikePath(value) {
			paths = append(paths, value)
		}
	}
	return paths
}

// parseKeyValueDocument is intentionally small: Steam app manifests place the
// fields AI Game Manager Panel currently needs (name, appid, installdir) as quoted key/value
// pairs. Nested objects are ignored until a real use-case requires a fuller VDF parser.
func parseKeyValueDocument(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		match := quotedPairPattern.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if len(match) != 3 {
			continue
		}
		key := strings.ToLower(unescapeVDF(match[1]))
		values[key] = unescapeVDF(match[2])
	}
	return values
}

func looksLikePath(value string) bool {
	return strings.Contains(value, `\`) || strings.Contains(value, "/") || strings.Contains(value, ":")
}

func unescapeVDF(value string) string {
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value
}

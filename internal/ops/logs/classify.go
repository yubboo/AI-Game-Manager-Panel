package loghub

import (
	"encoding/json"
	"regexp"
	"strings"
)

var coreTimestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}`)

func classifyLine(number uint64, text string) LogLine {
	line := LogLine{LineNumber: number, Text: text, Level: "info", Category: "general"}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return line
	}

	// 操作审计使用 JSON Lines，结构化字段优先于关键词启发式分类。
	var structured struct {
		TS       string `json:"ts"`
		Level    string `json:"level"`
		Category string `json:"category"`
	}
	if strings.HasPrefix(trimmed, "{") && json.Unmarshal([]byte(trimmed), &structured) == nil {
		if value := normalizeLevel(structured.Level); value != "" {
			line.Level = value
		}
		if value := normalizeCategory(structured.Category); value != "" {
			line.Category = value
		}
		line.Timestamp = strings.TrimSpace(structured.TS)
		return line
	}

	lower := strings.ToLower(trimmed)
	line.Level = classifyLevel(lower)
	line.Category = classifyCategory(lower)
	if coreTimestampPattern.MatchString(trimmed) {
		line.Timestamp = trimmed[:19]
	} else if strings.HasPrefix(trimmed, "[") {
		if end := strings.Index(trimmed, "]"); end > 1 && end < 16 {
			line.Timestamp = trimmed[1:end]
		}
	}
	return line
}

func normalizeLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "info", "warning", "error", "debug":
		return strings.ToLower(strings.TrimSpace(value))
	case "warn":
		return "warning"
	default:
		return ""
	}
}

func normalizeCategory(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "general", "system", "operation", "deployment", "steam", "network", "auth", "security", "audit", "player", "world", "mod", "backup", "ai", "task", "file":
		return value
	default:
		return ""
	}
}

func classifyLevel(lower string) string {
	switch {
	case strings.Contains(lower, "[error]"), strings.Contains(lower, " error:"), strings.HasPrefix(lower, "error:"), strings.Contains(lower, "fatal"), strings.Contains(lower, "panic"), strings.Contains(lower, "failed to start"), strings.Contains(lower, "failed to run"):
		return "error"
	case strings.Contains(lower, "[warning]"), strings.Contains(lower, "[warn]"), strings.Contains(lower, " warning:"), strings.HasPrefix(lower, "warning:"), strings.Contains(lower, "could not confirm port"):
		return "warning"
	case strings.Contains(lower, "[debug]"):
		return "debug"
	default:
		return "info"
	}
}

func classifyCategory(lower string) string {
	switch {
	case strings.Contains(lower, "operation"), strings.Contains(lower, "action="):
		return "operation"
	case strings.Contains(lower, "deploy"), strings.Contains(lower, "install"), strings.Contains(lower, "部署"):
		return "deployment"
	case strings.Contains(lower, "steam"), strings.Contains(lower, "appmanifest"), strings.Contains(lower, "workshop"):
		return "steam"
	case strings.Contains(lower, "firewall"), strings.Contains(lower, "port"), strings.Contains(lower, "network"), strings.Contains(lower, "socket"), strings.Contains(lower, "shard"):
		return "network"
	case strings.Contains(lower, "audit"), strings.Contains(lower, "审计"):
		return "audit"
	case strings.Contains(lower, "security"), strings.Contains(lower, "denied"), strings.Contains(lower, "forbidden"), strings.Contains(lower, "权限拒绝"):
		return "security"
	case strings.Contains(lower, "token"), strings.Contains(lower, "authenticate"), strings.Contains(lower, "permission"), strings.Contains(lower, "adminlist"), strings.Contains(lower, "whitelist"), strings.Contains(lower, "blocklist"):
		return "auth"
	case strings.Contains(lower, "player"), strings.Contains(lower, "client connected"), strings.Contains(lower, "client disconnected"), strings.Contains(lower, "user id"):
		return "player"
	case strings.Contains(lower, "world"), strings.Contains(lower, "save"), strings.Contains(lower, "session"):
		return "world"
	case strings.Contains(lower, "mod"), strings.Contains(lower, "workshop"):
		return "mod"
	case strings.Contains(lower, "backup"), strings.Contains(lower, "restore"):
		return "backup"
	case strings.Contains(lower, " ai "), strings.Contains(lower, "agent"), strings.Contains(lower, "tool call"):
		return "ai"
	case strings.Contains(lower, "task"), strings.Contains(lower, "scheduler"), strings.Contains(lower, "cron"):
		return "task"
	case strings.Contains(lower, "file"), strings.Contains(lower, "path="):
		return "file"
	case strings.Contains(lower, "system"), strings.Contains(lower, "runtime"), strings.Contains(lower, "agmp core"):
		return "system"
	default:
		return "general"
	}
}

func lineMatches(line LogLine, query, level, category string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query != "" && !strings.Contains(strings.ToLower(line.Text), query) {
		return false
	}
	level = strings.ToLower(strings.TrimSpace(level))
	if level != "" && level != "all" && line.Level != level {
		return false
	}
	category = strings.ToLower(strings.TrimSpace(category))
	if category != "" && category != "all" && line.Category != category {
		return false
	}
	return true
}

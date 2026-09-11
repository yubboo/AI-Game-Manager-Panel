package logcenter

import "time"

const (
	DefaultListLimit   = 100
	MaxListLimit       = 500
	DefaultReadLimit   = 500
	MaxReadLimit       = 2000
	DefaultSearchLimit = 200
	MaxSearchLimit     = 1000
)

type Session struct {
	ID               string `json:"id"`
	ClusterName      string `json:"clusterName"`
	ClusterPath      string `json:"clusterPath"`
	ShardName        string `json:"shardName"`
	Status           string `json:"status"`
	PID              int    `json:"pid"`
	WorldReady       bool   `json:"worldReady"`
	StartedAt        int64  `json:"startedAt"`
	EndedAt          int64  `json:"endedAt"`
	ExitCode         *int   `json:"exitCode"`
	Error            string `json:"error"`
	LineCount        uint64 `json:"lineCount"`
	ByteSize         int64  `json:"byteSize"`
	DroppedLines     uint64 `json:"droppedLines"`
	PersistenceError string `json:"persistenceError"`
	LogPath          string `json:"-"`
	MetaPath         string `json:"-"`
}

type ListRequest struct {
	ClusterPath string `json:"clusterPath"`
	ShardName   string `json:"shardName"`
	Limit       int    `json:"limit"`
}

type ReadRequest struct {
	SessionID string `json:"sessionId"`
	Cursor    int64  `json:"cursor"`
	StartLine uint64 `json:"startLine"`
	Limit     int    `json:"limit"`
}

type LogLine struct {
	LineNumber uint64 `json:"lineNumber"`
	Text       string `json:"text"`
	Level      string `json:"level"`
	Category   string `json:"category"`
}

type ReadPage struct {
	Session    Session   `json:"session"`
	Lines      []LogLine `json:"lines"`
	NextCursor int64     `json:"nextCursor"`
	NextLine   uint64    `json:"nextLine"`
	EOF        bool      `json:"eof"`
}

type TailRequest struct {
	SessionID string `json:"sessionId"`
	Limit     int    `json:"limit"`
}

type SearchRequest struct {
	SessionID string `json:"sessionId"`
	Query     string `json:"query"`
	Level     string `json:"level"`
	Category  string `json:"category"`
	Limit     int    `json:"limit"`
}

type SearchResult struct {
	Session    Session   `json:"session"`
	Matches    []LogLine `json:"matches"`
	Scanned    uint64    `json:"scanned"`
	Truncated  bool      `json:"truncated"`
	DurationMs int64     `json:"durationMs"`
}

type DiagnosticIssue struct {
	Code        string   `json:"code"`
	Severity    string   `json:"severity"`
	Title       string   `json:"title"`
	Detail      string   `json:"detail"`
	Count       int      `json:"count"`
	Suggestions []string `json:"suggestions"`
	Evidence    []string `json:"evidence"`
	Certain     bool     `json:"certain"`
}

type Diagnostics struct {
	Session      Session           `json:"session"`
	ErrorCount   int               `json:"errorCount"`
	WarningCount int               `json:"warningCount"`
	SteamReady   bool              `json:"steamReady"`
	Registered   bool              `json:"registered"`
	TokenLoaded  bool              `json:"tokenLoaded"`
	NetworkReady bool              `json:"networkReady"`
	WorldReady   bool              `json:"worldReady"`
	Healthy      bool              `json:"healthy"`
	Issues       []DiagnosticIssue `json:"issues"`
	ScannedLines uint64            `json:"scannedLines"`
	DurationMs   int64             `json:"durationMs"`
}

type ExportResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type FileRef struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
}

type BundleRequest struct {
	ClusterPath string   `json:"clusterPath"`
	ShardNames  []string `json:"shardNames"`
}

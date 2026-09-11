package loghub

// CatalogRequest 是日志目录查询条件。所有筛选都在 Go 端执行，前端只接收当前页元数据，
// 避免日志文件数量增长后把整个目录和大文件一次性送入 WebView。
type CatalogRequest struct {
	Query      string `json:"query"`
	Source     string `json:"source"`
	Kind       string `json:"kind"`
	GameID     string `json:"gameId"`
	InstanceID string `json:"instanceId"`
	Shard      string `json:"shard"`
	Status     string `json:"status"`
	DateFrom   int64  `json:"dateFrom"`
	DateTo     int64  `json:"dateTo"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

// Summary 是当前筛选条件下的真实日志统计。
type Summary struct {
	Files       int    `json:"files"`
	Lines       uint64 `json:"lines"`
	Bytes       int64  `json:"bytes"`
	ActiveFiles int    `json:"activeFiles"`
}

// LogFile 描述一个可在全局日志中心管理的真实日志文件。
// RelativePath 始终相对于 AI Game Manager Panel log 根目录，不向前端暴露 root 之外的路径。
type LogFile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelativePath string `json:"relativePath"`
	Source       string `json:"source"`
	SourceLabel  string `json:"sourceLabel"`
	Kind         string `json:"kind"`
	GameID       string `json:"gameId"`
	GameName     string `json:"gameName"`
	InstanceID   string `json:"instanceId"`
	InstanceName string `json:"instanceName"`
	Shard        string `json:"shard"`
	Status       string `json:"status"`
	Active       bool   `json:"active"`
	StartedAt    int64  `json:"startedAt"`
	EndedAt      int64  `json:"endedAt"`
	ModifiedAt   int64  `json:"modifiedAt"`
	LineCount    uint64 `json:"lineCount"`
	ByteSize     int64  `json:"byteSize"`
}

// CatalogPage 返回当前筛选后的目录页和真实统计。
type CatalogPage struct {
	Items      []LogFile `json:"items"`
	Summary    Summary   `json:"summary"`
	Total      int       `json:"total"`
	Offset     int       `json:"offset"`
	Limit      int       `json:"limit"`
	DurationMs int64     `json:"durationMs"`
}

// ReadRequest 支持从头、从尾或按游标继续读取。Direction 取 head/tail/next。
type ReadRequest struct {
	ID        string `json:"id"`
	Direction string `json:"direction"`
	Cursor    int64  `json:"cursor"`
	StartLine uint64 `json:"startLine"`
	Limit     int    `json:"limit"`
	Query     string `json:"query"`
	Level     string `json:"level"`
	Category  string `json:"category"`
}

// LogLine 是统一结构化日志行。Text 保留原始行，Level/Category 仅用于展示与筛选。
type LogLine struct {
	LineNumber uint64 `json:"lineNumber"`
	Text       string `json:"text"`
	Level      string `json:"level"`
	Category   string `json:"category"`
	Timestamp  string `json:"timestamp"`
}

// ReadPage 是单个日志文件的分页/搜索结果。
type ReadPage struct {
	File       LogFile   `json:"file"`
	Lines      []LogLine `json:"lines"`
	NextCursor int64     `json:"nextCursor"`
	NextLine   uint64    `json:"nextLine"`
	EOF        bool      `json:"eof"`
	Scanned    uint64    `json:"scanned"`
	Matched    uint64    `json:"matched"`
	DurationMs int64     `json:"durationMs"`
}

// ExportResult 描述写入 AI Game Manager Panel 配置日志根目录 exports 子目录的导出产物。
type ExportResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// MutationResult 用于删除单条、删除筛选结果和清理历史日志。
type MutationResult struct {
	Deleted      int      `json:"deleted"`
	Skipped      int      `json:"skipped"`
	BytesFreed   int64    `json:"bytesFreed"`
	DeletedIDs   []string `json:"deletedIds"`
	SkippedIDs   []string `json:"skippedIds"`
	ErrorDetails []string `json:"errorDetails"`
}

// DeleteFilteredRequest 删除当前筛选结果。活动日志自动跳过。
type DeleteFilteredRequest struct {
	Filter CatalogRequest `json:"filter"`
}

// OperationRecord 是 AI Game Manager Panel 公共操作审计的安全结构。
// Detail 不允许写 Token、密码、API Key 或完整控制台命令。
type OperationRecord struct {
	Level    string `json:"level"`
	Source   string `json:"source"`
	Action   string `json:"action"`
	Target   string `json:"target"`
	Result   string `json:"result"`
	Detail   string `json:"detail"`
	GameID   string `json:"gameId"`
	Instance string `json:"instance"`
}

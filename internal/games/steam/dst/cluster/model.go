package cluster

type ImportRequest struct {
	SourcePath string `json:"sourcePath"`
	TargetName string `json:"targetName"`
}

type ImportResult struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	SourcePath   string   `json:"sourcePath"`
	CopiedFiles  int      `json:"copiedFiles"`
	CopiedBytes  int64    `json:"copiedBytes"`
	TokenSkipped bool     `json:"tokenSkipped"`
	Warnings     []string `json:"warnings"`
}

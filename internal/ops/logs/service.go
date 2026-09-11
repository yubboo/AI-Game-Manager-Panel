package loghub

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	defaultCatalogLimit = 100
	maxCatalogLimit     = 500
	defaultReadLimit    = 600
	maxReadLimit        = 2000
)

var (
	ErrLogNotFound = errors.New("日志文件不存在")
	ErrLogActive   = errors.New("运行中的日志受保护，停止对应实例后才能删除")
)

type lineCacheEntry struct {
	Size             int64
	ModifiedAt       int64
	Lines            uint64
	EndedWithNewline bool
}

// Service 是全平台唯一日志中心。
// 它只管理 AI Game Manager Panel log 根目录中的日志，不允许读取/删除 root 之外的任意文件。
type Service struct {
	root         string
	exportRoot   string
	coreDir      string
	operationDir string
	auditDir     string
	aiDir        string
	steamDir     string
	gamesDir     string
	nodesDir     string
	cacheMu      sync.Mutex
	lineCache    map[string]lineCacheEntry
	recorder     *Recorder
}

// Options 来自 configs/logging.json，日志目录名称和刷新参数不在业务代码中写死。
type Options struct {
	ExportSubdir    string
	CoreDir         string
	OperationDir    string
	AuditDir        string
	AIDir           string
	SteamDir        string
	GamesDir        string
	NodesDir        string
	OperationAudit  bool
	FlushIntervalMS int
}

func New(root string, options Options) *Service {
	root = filepath.Clean(root)
	exportSubdir := strings.TrimSpace(options.ExportSubdir)
	if exportSubdir == "" {
		exportSubdir = "exports"
	}
	coreDir := optionDir(options.CoreDir, "agmp")
	operationDir := optionDir(options.OperationDir, "operations")
	auditDir := optionDir(options.AuditDir, "audit")
	aiDir := optionDir(options.AIDir, "ai")
	steamDir := optionDir(options.SteamDir, "steam")
	gamesDir := optionDir(options.GamesDir, "games")
	nodesDir := optionDir(options.NodesDir, "nodes")
	s := &Service{
		root:         root,
		exportRoot:   filepath.Join(root, filepath.Clean(exportSubdir)),
		coreDir:      coreDir,
		operationDir: operationDir,
		auditDir:     auditDir,
		aiDir:        aiDir,
		steamDir:     steamDir,
		gamesDir:     gamesDir,
		nodesDir:     nodesDir,
		lineCache:    make(map[string]lineCacheEntry),
	}
	if options.OperationAudit {
		s.recorder = NewRecorder(filepath.Join(root, filepath.Clean(operationDir)), options.FlushIntervalMS)
	}
	return s
}

func (s *Service) Root() string { return s.root }

func optionDir(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) {
		return fallback
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return fallback
	}
	return cleaned
}

func (s *Service) Close() {
	if s.recorder != nil {
		s.recorder.Close()
		s.recorder = nil
	}
}

func (s *Service) RecordOperation(value OperationRecord) {
	if s.recorder != nil {
		s.recorder.Record(value)
	}
}

func (s *Service) ensureRoot() error {
	if strings.TrimSpace(s.root) == "" || s.root == "." {
		return fmt.Errorf("日志根目录无效")
	}
	return os.MkdirAll(s.root, 0o755)
}

func (s *Service) safePath(relative string) (string, error) {
	relative = strings.TrimSpace(relative)
	if relative == "" || filepath.IsAbs(relative) {
		return "", ErrLogNotFound
	}
	candidate := filepath.Clean(filepath.Join(s.root, relative))
	rootAbs, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	prefix := rootAbs + string(os.PathSeparator)
	if candidateAbs != rootAbs && !strings.HasPrefix(strings.ToLower(candidateAbs), strings.ToLower(prefix)) {
		return "", fmt.Errorf("拒绝访问日志根目录之外的路径")
	}
	return candidateAbs, nil
}

// Package xiaoyuruntime connects AGMP to XiaoYu's Rust Agent Runtime.
//
// 0.2.15 keeps the same Host-authorized cross-platform Terminal RPC and focuses on proving Linux PTY / Windows ConPTY through complete CI evidence. Go remains the source
// of truth for AGMP domain services, while generic Agent capabilities move to
// Rust incrementally. Existing Go execution paths remain compatible until the
// Rust equivalents have protocol tests and Agent Bench coverage.
package xiaoyuruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

const ProtocolVersion = "xiaoyu.v1"

type Options struct {
	Root           string
	BinaryOverride string
	Timeout        time.Duration
}

type Status struct {
	Enabled      bool     `json:"enabled"`
	Available    bool     `json:"available"`
	Ready        bool     `json:"ready"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Protocol     string   `json:"protocol"`
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
	ToolCount    int      `json:"toolCount"`
	Message      string   `json:"message"`
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Risk        string         `json:"risk"`
	Category    string         `json:"category"`
	Manual      bool           `json:"manual"`
	XiaoYu      bool           `json:"xiaoyu"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Source      string         `json:"source,omitempty"`
}

type ToolSearchRequest struct {
	Query string     `json:"query"`
	Tools []ToolSpec `json:"tools"`
	Limit int        `json:"limit,omitempty"`
}

type ToolSearchHit struct {
	Tool  ToolSpec `json:"tool"`
	Score uint32   `json:"score"`
}

type ToolSearchResponse struct {
	Query string          `json:"query"`
	Hits  []ToolSearchHit `json:"hits"`
}

type ToolCallRequest struct {
	Tool       string         `json:"tool"`
	Arguments  map[string]any `json:"arguments"`
	ApprovalID string         `json:"approvalId,omitempty"`
}

type ToolCallResult struct {
	Tool       string `json:"tool"`
	Decision   string `json:"decision"`
	Summary    string `json:"summary"`
	Data       any    `json:"data"`
	ApprovalID string `json:"approvalId,omitempty"`
	Pending    bool   `json:"pending,omitempty"`
	Risk       string `json:"risk,omitempty"`
}

type CommandRequest struct {
	Command          string `json:"command"`
	WorkingDirectory string `json:"workingDirectory"`
	ApprovalID       string `json:"approvalId,omitempty"`
}

type CommandResult struct {
	Command    string `json:"command"`
	Cwd        string `json:"cwd"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Decision   string `json:"decision"`
	ApprovalID string `json:"approvalId,omitempty"`
	Pending    bool   `json:"pending,omitempty"`
}

type Service struct {
	root           string
	binaryOverride string
	timeout        time.Duration

	// rpcMu serializes JSON-RPC frames over one long-lived stdio worker. The
	// Rust Runtime owns stateful Session/Job registries, so a fresh child per
	// request would destroy the very state these APIs are meant to preserve.
	rpcMu      sync.Mutex
	worker     *rpcWorker
	requestSeq uint64
}

func New(options Options) *Service {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Service{
		root:           filepath.Clean(options.Root),
		binaryOverride: strings.TrimSpace(options.BinaryOverride),
		timeout:        timeout,
	}
}

func (s *Service) Status() Status {
	path, err := s.binaryPath()
	if err != nil {
		return Status{
			Enabled:   true,
			Available: false,
			Ready:     false,
			Protocol:  ProtocolVersion,
			Message:   err.Error(),
		}
	}
	var payload struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Protocol     string   `json:"protocol"`
		Ready        bool     `json:"ready"`
		Capabilities []string `json:"capabilities"`
		ToolCount    int      `json:"toolCount"`
	}
	if err := s.runJSON(&payload, path, "--root", s.root, "doctor", "--json"); err != nil {
		return Status{
			Enabled:   true,
			Available: true,
			Ready:     false,
			Path:      path,
			Protocol:  ProtocolVersion,
			Message:   err.Error(),
		}
	}
	message := "小鱼核心已就绪"
	if payload.Protocol != ProtocolVersion {
		message = fmt.Sprintf("小鱼核心协议版本不匹配：runtime=%s core=%s", payload.Protocol, ProtocolVersion)
	}
	return Status{
		Enabled:      true,
		Available:    true,
		Ready:        payload.Ready && payload.Protocol == ProtocolVersion,
		Name:         payload.Name,
		Version:      payload.Version,
		Protocol:     payload.Protocol,
		Path:         path,
		Capabilities: payload.Capabilities,
		ToolCount:    payload.ToolCount,
		Message:      message,
	}
}

// SearchTools delegates capability relevance ranking to the Rust Agent Runtime.
// A search hit does not grant execution permission; approval/RBAC remains
// independent from capability discovery.
func (s *Service) SearchTools(ctx context.Context, query string, tools []ToolSpec, limit int) (ToolSearchResponse, error) {
	var value ToolSearchResponse
	request := ToolSearchRequest{Query: strings.TrimSpace(query), Tools: tools, Limit: limit}
	if err := s.runRPC(ctx, "tools/search", request, &value); err != nil {
		return value, err
	}
	return value, nil
}

// Ready implements host.BrainPolicy. A configured model is not considered a
// usable XiaoYu Brain unless the Rust policy core is present and protocol-ready.
func (s *Service) Ready(_ context.Context) (bool, string) {
	status := s.Status()
	return status.Ready, status.Message
}

// Prepare asks xiaoyu-core to build the provider-neutral reasoning prompt for
// one Agent frame. Model HTTP transport stays in AGMP Go Host; planning policy
// stays in Rust.
func (s *Service) Prepare(ctx context.Context, frame xiaoyuhost.Frame) (xiaoyuhost.BrainPrompt, error) {
	var value xiaoyuhost.BrainPrompt
	if err := s.runRPC(ctx, "brain/prepare", frame, &value); err != nil {
		return value, err
	}
	return value, nil
}

// Resolve sends a normalized model turn back to xiaoyu-core, which validates
// the Agent decision grammar. Go does not silently reinterpret free-form model
// text into executable intent.
func (s *Service) Resolve(ctx context.Context, turn xiaoyuhost.ModelTurn) (xiaoyuhost.Decision, error) {
	var value xiaoyuhost.Decision
	if err := s.runRPC(ctx, "brain/resolve", turn, &value); err != nil {
		return value, err
	}
	return value, nil
}

func (s *Service) Start(ctx context.Context) error {
	var status struct {
		Protocol string `json:"protocol"`
		Ready    bool   `json:"ready"`
	}
	if err := s.runRPC(ctx, "initialize", map[string]any{}, &status); err != nil {
		return err
	}
	if status.Protocol != ProtocolVersion {
		return fmt.Errorf("XiaoYu Persistent Runtime Worker 协议不匹配：runtime=%s host=%s", status.Protocol, ProtocolVersion)
	}
	if !status.Ready {
		return errors.New("XiaoYu Persistent Runtime Worker 尚未就绪")
	}
	return nil
}

// Close terminates the supervised Rust stdio worker. It is safe to call more
// than once and does not affect AGMP domain services.
func (s *Service) Close() {
	s.rpcMu.Lock()
	defer s.rpcMu.Unlock()
	s.stopWorkerLocked()
}

// SessionInfo, Job* and Terminal* structures mirror xiaoyu.v1. They are Host-internal
// primitives in 0.2.15, including Linux PTY and Windows ConPTY/resize; model-visible execution still goes through AGMP Tool
// contracts and the existing RBAC/approval boundary.
type SessionInfo struct {
	ID           string `json:"id"`
	Cwd          string `json:"cwd"`
	ApprovalMode string `json:"approvalMode"`
}

type JobStartRequest struct {
	SessionID      string   `json:"sessionId,omitempty"`
	Executable     string   `json:"executable"`
	Arguments      []string `json:"arguments,omitempty"`
	Cwd            string   `json:"cwd,omitempty"`
	MaxOutputBytes int      `json:"maxOutputBytes,omitempty"`
	HostAuthorized bool     `json:"hostAuthorized"`
}

type JobSnapshot struct {
	ID              string   `json:"id"`
	SessionID       string   `json:"sessionId,omitempty"`
	Executable      string   `json:"executable"`
	Arguments       []string `json:"arguments"`
	Cwd             string   `json:"cwd"`
	State           string   `json:"state"`
	PID             *int     `json:"pid,omitempty"`
	ExitCode        *int     `json:"exitCode,omitempty"`
	CreatedAt       uint64   `json:"createdAt"`
	StartedAt       *uint64  `json:"startedAt,omitempty"`
	FinishedAt      *uint64  `json:"finishedAt,omitempty"`
	OutputTruncated bool     `json:"outputTruncated"`
}

type JobOutputChunk struct {
	Sequence uint64 `json:"sequence"`
	Stream   string `json:"stream"`
	Text     string `json:"text"`
}

type JobOutputResponse struct {
	ID         string           `json:"id"`
	Chunks     []JobOutputChunk `json:"chunks"`
	NextCursor uint64           `json:"nextCursor"`
	Truncated  bool             `json:"truncated"`
}

type TerminalStartRequest struct {
	SessionID      string   `json:"sessionId,omitempty"`
	Executable     string   `json:"executable"`
	Arguments      []string `json:"arguments,omitempty"`
	Cwd            string   `json:"cwd,omitempty"`
	MaxOutputBytes int      `json:"maxOutputBytes,omitempty"`
	Rows           uint16   `json:"rows,omitempty"`
	Cols           uint16   `json:"cols,omitempty"`
	HostAuthorized bool     `json:"hostAuthorized"`
}

type TerminalWriteRequest struct {
	ID             string `json:"id"`
	Data           string `json:"data"`
	AppendNewline  bool   `json:"appendNewline"`
	HostAuthorized bool   `json:"hostAuthorized"`
}

type TerminalResizeRequest struct {
	ID             string `json:"id"`
	Rows           uint16 `json:"rows"`
	Cols           uint16 `json:"cols"`
	HostAuthorized bool   `json:"hostAuthorized"`
}

type TerminalSnapshot struct {
	ID              string   `json:"id"`
	SessionID       string   `json:"sessionId,omitempty"`
	Executable      string   `json:"executable"`
	Arguments       []string `json:"arguments"`
	Cwd             string   `json:"cwd"`
	State           string   `json:"state"`
	PID             *int     `json:"pid,omitempty"`
	ExitCode        *int     `json:"exitCode,omitempty"`
	CreatedAt       uint64   `json:"createdAt"`
	FinishedAt      *uint64  `json:"finishedAt,omitempty"`
	OutputTruncated bool     `json:"outputTruncated"`
	Backend         string   `json:"backend"`
	Rows            uint16   `json:"rows"`
	Cols            uint16   `json:"cols"`
}

type TerminalOutputResponse struct {
	ID         string           `json:"id"`
	Chunks     []JobOutputChunk `json:"chunks"`
	NextCursor uint64           `json:"nextCursor"`
	Truncated  bool             `json:"truncated"`
}

func (s *Service) CreateSession(ctx context.Context, cwd, approvalMode string) (SessionInfo, error) {
	var value SessionInfo
	err := s.runRPC(ctx, "session/create", map[string]any{"cwd": cwd, "approvalMode": approvalMode}, &value)
	return value, err
}

func (s *Service) GetSession(ctx context.Context, id string) (SessionInfo, error) {
	var value SessionInfo
	err := s.runRPC(ctx, "session/get", map[string]any{"id": id}, &value)
	return value, err
}

func (s *Service) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	var value []SessionInfo
	err := s.runRPC(ctx, "session/list", map[string]any{}, &value)
	return value, err
}

func (s *Service) CloseSession(ctx context.Context, id string) (bool, error) {
	var value struct {
		Closed bool `json:"closed"`
	}
	err := s.runRPC(ctx, "session/close", map[string]any{"id": id}, &value)
	return value.Closed, err
}

func (s *Service) StartJob(ctx context.Context, request JobStartRequest) (JobSnapshot, error) {
	var value JobSnapshot
	if !request.HostAuthorized {
		return value, errors.New("XiaoYu Rust Job 必须先经过 Host 授权")
	}
	err := s.runRPC(ctx, "jobs/start", request, &value)
	return value, err
}

func (s *Service) GetJob(ctx context.Context, id string) (JobSnapshot, error) {
	var value JobSnapshot
	err := s.runRPC(ctx, "jobs/get", map[string]any{"id": id}, &value)
	return value, err
}

func (s *Service) ListJobs(ctx context.Context) ([]JobSnapshot, error) {
	var value []JobSnapshot
	err := s.runRPC(ctx, "jobs/list", map[string]any{}, &value)
	return value, err
}

func (s *Service) JobOutput(ctx context.Context, id string, after uint64, limit int) (JobOutputResponse, error) {
	var value JobOutputResponse
	err := s.runRPC(ctx, "jobs/output", map[string]any{"id": id, "after": after, "limit": limit}, &value)
	return value, err
}

func (s *Service) CancelJob(ctx context.Context, id string) (JobSnapshot, error) {
	var value JobSnapshot
	err := s.runRPC(ctx, "jobs/cancel", map[string]any{"id": id}, &value)
	return value, err
}

func (s *Service) StartTerminal(ctx context.Context, request TerminalStartRequest) (TerminalSnapshot, error) {
	var value TerminalSnapshot
	if !request.HostAuthorized {
		return value, errors.New("XiaoYu Rust Terminal 必须先经过 Host 授权")
	}
	err := s.runRPC(ctx, "terminal/start", request, &value)
	return value, err
}

func (s *Service) GetTerminal(ctx context.Context, id string) (TerminalSnapshot, error) {
	var value TerminalSnapshot
	err := s.runRPC(ctx, "terminal/get", map[string]any{"id": id}, &value)
	return value, err
}

func (s *Service) ListTerminals(ctx context.Context) ([]TerminalSnapshot, error) {
	var value []TerminalSnapshot
	err := s.runRPC(ctx, "terminal/list", map[string]any{}, &value)
	return value, err
}

func (s *Service) WriteTerminal(ctx context.Context, request TerminalWriteRequest) (TerminalSnapshot, error) {
	var value TerminalSnapshot
	if !request.HostAuthorized {
		return value, errors.New("XiaoYu Rust Terminal 输入必须先经过 Host 授权")
	}
	err := s.runRPC(ctx, "terminal/write", request, &value)
	return value, err
}

func (s *Service) TerminalOutput(ctx context.Context, id string, after uint64, limit int) (TerminalOutputResponse, error) {
	var value TerminalOutputResponse
	err := s.runRPC(ctx, "terminal/output", map[string]any{"id": id, "after": after, "limit": limit}, &value)
	return value, err
}

func (s *Service) ResizeTerminal(ctx context.Context, request TerminalResizeRequest) (TerminalSnapshot, error) {
	var value TerminalSnapshot
	if !request.HostAuthorized {
		return value, errors.New("XiaoYu Rust Terminal 调整尺寸必须先经过 Host 授权")
	}
	err := s.runRPC(ctx, "terminal/resize", request, &value)
	return value, err
}

func (s *Service) CloseTerminal(ctx context.Context, id string) (TerminalSnapshot, error) {
	var value TerminalSnapshot
	err := s.runRPC(ctx, "terminal/close", map[string]any{"id": id}, &value)
	return value, err
}

func (s *Service) runRPC(parent context.Context, method string, params any, target any) error {
	path, err := s.binaryPath()
	if err != nil {
		return err
	}
	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	s.rpcMu.Lock()
	defer s.rpcMu.Unlock()
	if s.worker == nil {
		worker, startErr := startRPCWorker(path, s.root)
		if startErr != nil {
			return startErr
		}
		s.worker = worker
	}
	s.requestSeq++
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("agmp-%d", s.requestSeq),
		"method":  method,
		"params":  params,
	}
	line, err := s.worker.call(ctx, request)
	if err != nil {
		s.stopWorkerLocked()
		return err
	}

	var response struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(line, &response); err != nil {
		s.stopWorkerLocked()
		return fmt.Errorf("解析 XiaoYu Persistent Runtime Worker 响应失败：%w", err)
	}
	if response.Error != nil {
		return fmt.Errorf("XiaoYu Agent Runtime RPC 错误 %d：%s", response.Error.Code, response.Error.Message)
	}
	if len(response.Result) == 0 {
		return errors.New("XiaoYu Agent Runtime RPC 没有返回 result")
	}
	if err := json.Unmarshal(response.Result, target); err != nil {
		return fmt.Errorf("解析 XiaoYu Agent Runtime RPC result 失败：%w", err)
	}
	return nil
}

func (s *Service) stopWorkerLocked() {
	if s.worker == nil {
		return
	}
	s.worker.close()
	s.worker = nil
}

// Domain Tool dispatch still belongs to AGMP Go services. This bridge now also
// exposes Rust-owned capability search; generic native execution will migrate
// behind this protocol incrementally without bypassing Host approval/RBAC.

func (s *Service) binaryPath() (string, error) {
	candidates := make([]string, 0, 8)
	if s.binaryOverride != "" {
		candidates = append(candidates, s.binaryOverride)
	}
	if env := strings.TrimSpace(os.Getenv("AGMP_XIAOYU_RUNTIME")); env != "" {
		candidates = append(candidates, env)
	}
	// 兼容 0.1.80 及更早版本的开发环境变量。
	if env := strings.TrimSpace(os.Getenv("AGMP_AGENT_RUNTIME")); env != "" {
		candidates = append(candidates, env)
	}
	name := "AI-Game-Manager-XiaoYu"
	rustName := "xiaoyu"
	if runtime.GOOS == "windows" {
		name += ".exe"
		rustName += ".exe"
	}
	candidates = append(candidates,
		// 生产包优先从 AGMP 内部组件目录发现 AI Core；普通用户不需要也不应单独启动该文件。
		filepath.Join(s.root, "internal", "xiaoyu", name),
		// 兼容 0.1.80 的旧 AI Core 文件名。
		filepath.Join(s.root, "internal", "xiaoyu", legacyBinaryName()),
		// 兼容 0.1.74-0.1.78 的旧安装/便携布局，以及开发构建输出。
		filepath.Join(s.root, name),
		filepath.Join(s.root, "bin", name),
		filepath.Join(s.root, "rust", "target", "release", rustName),
		filepath.Join(s.root, "rust", "target", "debug", rustName),
	)
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			abs, absErr := filepath.Abs(candidate)
			if absErr == nil {
				return abs, nil
			}
			return candidate, nil
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	return "", errors.New("未找到小鱼核心运行时；生产安装包应自带该组件，源码开发时请构建 rust/crates/xiaoyu-core")
}

func legacyBinaryName() string {
	name := "AI-Game-Manager-Agent"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func (s *Service) runJSON(target any, path string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	result, err := platformruntime.Run(ctx, platformruntime.RunSpec{
		Spec: platformruntime.Spec{
			Executable:       path,
			Arguments:        args,
			WorkingDirectory: s.root,
		},
		MaxOutputBytes: 4 * 1024 * 1024,
	})
	if result.TimedOut {
		return fmt.Errorf("小鱼核心执行超时：%w", ctx.Err())
	}
	if errors.Is(err, platformruntime.ErrOutputTruncated) {
		return errors.New("小鱼核心输出超过安全上限，已拒绝解析")
	}
	if err != nil {
		message := strings.TrimSpace(result.Stderr)
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("小鱼核心执行失败：%s", message)
	}
	if err := json.Unmarshal([]byte(result.Stdout), target); err != nil {
		return fmt.Errorf("解析小鱼核心输出失败：%w", err)
	}
	return nil
}

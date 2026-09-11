// Package approvals 管理 小鱼（AGMP 内置智能核心） 的审批模式与待审批请求。
//
// 审批记录只保存操作摘要和请求指纹，不保存完整命令、Token、API Key、密码等敏感参数。
// 未决审批没有自动超时；用户未明确批准或拒绝前，小鱼不得继续对应操作。
package control

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

const stateVersion = 1

var (
	ErrApprovalNotFound = errors.New("审批请求不存在")
	ErrApprovalPending  = errors.New("审批请求仍在等待用户决定")
	ErrApprovalRejected = errors.New("用户已拒绝该操作")
	ErrApprovalMismatch = errors.New("审批请求与当前操作不匹配")
	ErrApprovalConsumed = errors.New("审批已使用，不能重复执行")
)

type State string

const (
	StatePending  State = "pending"
	StateApproved State = "approved"
	StateRejected State = "rejected"
	StateConsumed State = "consumed"
)

type RequestKind string

const (
	KindTool    RequestKind = "tool"
	KindCommand RequestKind = "command"
)

// Request 是可以安全展示给管理员的审批元数据。
// RequestHash 绑定真实调用参数，但不会泄露真实参数本身。
type Request struct {
	ID          string      `json:"id"`
	Kind        RequestKind `json:"kind"`
	Subject     string      `json:"subject"`
	Risk        string      `json:"risk"`
	Summary     string      `json:"summary"`
	RequestHash string      `json:"requestHash"`
	State       State       `json:"state"`
	CreatedAt   int64       `json:"createdAt"`
	DecidedAt   int64       `json:"decidedAt,omitempty"`
	DecidedBy   string      `json:"decidedBy,omitempty"`
	ConsumedAt  int64       `json:"consumedAt,omitempty"`
}

type Snapshot struct {
	Mode         Mode   `json:"mode"`
	ModeLabel    string `json:"modeLabel"`
	UpdatedAt    int64  `json:"updatedAt"`
	UpdatedBy    string `json:"updatedBy,omitempty"`
	PendingCount int    `json:"pendingCount"`
}

type SetModeRequest struct {
	Mode string `json:"mode"`
}

type ResolveRequest struct {
	Decision string `json:"decision"`
}

type fileState struct {
	Version   int       `json:"version"`
	Mode      Mode      `json:"mode"`
	UpdatedAt int64     `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	Requests  []Request `json:"requests"`
}

type Store struct {
	path        string
	defaultMode Mode
	now         func() time.Time
	mu          sync.Mutex
	state       fileState
	loadErr     error
}

func New(path string, defaultMode Mode) *Store {
	if !ValidMode(defaultMode) {
		defaultMode = ModeAsk
	}
	s := &Store{
		path:        filepath.Clean(path),
		defaultMode: defaultMode,
		now:         time.Now,
		state:       fileState{Version: stateVersion, Mode: defaultMode, Requests: []Request{}},
	}
	if err := s.load(); err != nil {
		s.loadErr = err
		// 配置损坏时回退到最保守的“请求批准”，但所有会产生副作用的审批写入仍会 fail-closed。
		s.state = fileState{Version: stateVersion, Mode: ModeAsk, Requests: []Request{}}
	}
	return s
}

func (s *Store) Ready() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadErr
}

func (s *Store) Mode() Mode {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !ValidMode(s.state.Mode) {
		return ModeAsk
	}
	return s.state.Mode
}

func (s *Store) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending := 0
	for _, item := range s.state.Requests {
		if item.State == StatePending {
			pending++
		}
	}
	return Snapshot{
		Mode:         s.effectiveModeLocked(),
		ModeLabel:    ModeLabel(s.effectiveModeLocked()),
		UpdatedAt:    s.state.UpdatedAt,
		UpdatedBy:    s.state.UpdatedBy,
		PendingCount: pending,
	}
}

func (s *Store) SetMode(mode Mode, actor string) (Snapshot, error) {
	if !ValidMode(mode) {
		return Snapshot{}, fmt.Errorf("未知审批模式：%s", mode)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return Snapshot{}, fmt.Errorf("审批状态文件不可用：%w", s.loadErr)
	}
	s.state.Mode = mode
	s.state.UpdatedAt = s.now().Unix()
	s.state.UpdatedBy = safeText(actor, 80)
	if err := s.saveLocked(); err != nil {
		return Snapshot{}, err
	}
	return s.snapshotLocked(), nil
}

func (s *Store) Pending() ([]Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return nil, fmt.Errorf("审批状态文件不可用：%w", s.loadErr)
	}
	result := make([]Request, 0)
	for _, item := range s.state.Requests {
		if item.State == StatePending {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt < result[j].CreatedAt })
	return result, nil
}

// CreatePending 创建或复用相同请求指纹的待审批记录。
// 相同操作不会因为 小鱼重试而不断制造重复审批卡片。
func (s *Store) CreatePending(kind RequestKind, subject, risk, summary, requestHash string) (Request, error) {
	requestHash = strings.TrimSpace(requestHash)
	if requestHash == "" {
		return Request{}, errors.New("审批请求指纹不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return Request{}, fmt.Errorf("审批状态文件不可用：%w", s.loadErr)
	}
	for _, item := range s.state.Requests {
		if item.RequestHash == requestHash && item.State == StatePending {
			return item, nil
		}
	}
	id, err := newID()
	if err != nil {
		return Request{}, fmt.Errorf("生成审批 ID 失败：%w", err)
	}
	item := Request{
		ID:          id,
		Kind:        kind,
		Subject:     safeText(subject, 120),
		Risk:        safeText(risk, 32),
		Summary:     safeText(summary, 180),
		RequestHash: requestHash,
		State:       StatePending,
		CreatedAt:   s.now().Unix(),
	}
	s.state.Requests = append(s.state.Requests, item)
	s.trimHistoryLocked()
	if err := s.saveLocked(); err != nil {
		return Request{}, err
	}
	return item, nil
}

func (s *Store) Resolve(id string, approve bool, actor string) (Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return Request{}, fmt.Errorf("审批状态文件不可用：%w", s.loadErr)
	}
	index := s.indexLocked(id)
	if index < 0 {
		return Request{}, ErrApprovalNotFound
	}
	item := &s.state.Requests[index]
	if item.State != StatePending {
		return Request{}, fmt.Errorf("审批请求当前状态为 %s，不能重复决定", item.State)
	}
	if approve {
		item.State = StateApproved
	} else {
		item.State = StateRejected
	}
	item.DecidedAt = s.now().Unix()
	item.DecidedBy = safeText(actor, 80)
	if err := s.saveLocked(); err != nil {
		return Request{}, err
	}
	return *item, nil
}

// ConsumeApproval 只允许“已批准 + 同一请求指纹”的操作执行一次，避免批准被重放到另一条命令。
func (s *Store) ConsumeApproval(id, requestHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return fmt.Errorf("审批状态文件不可用：%w", s.loadErr)
	}
	index := s.indexLocked(id)
	if index < 0 {
		return ErrApprovalNotFound
	}
	item := &s.state.Requests[index]
	if item.RequestHash != strings.TrimSpace(requestHash) {
		return ErrApprovalMismatch
	}
	switch item.State {
	case StatePending:
		return ErrApprovalPending
	case StateRejected:
		return ErrApprovalRejected
	case StateConsumed:
		return ErrApprovalConsumed
	case StateApproved:
		item.State = StateConsumed
		item.ConsumedAt = s.now().Unix()
		if err := s.saveLocked(); err != nil {
			return err
		}
		return nil
	default:
		return errors.New("审批状态无效")
	}
}

// Fingerprint 对真实调用做不可逆绑定。只把哈希写入审批文件，不持久化原始参数。
func Fingerprint(kind RequestKind, subject string, payload any) (string, error) {
	raw, err := json.Marshal(struct {
		Kind    RequestKind `json:"kind"`
		Subject string      `json:"subject"`
		Payload any         `json:"payload"`
	}{Kind: kind, Subject: strings.TrimSpace(subject), Payload: payload})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取审批状态失败：%w", err)
	}
	var state fileState
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("解析审批状态失败：%w", err)
	}
	if state.Version != stateVersion {
		return fmt.Errorf("不支持的审批状态版本：%d", state.Version)
	}
	if !ValidMode(state.Mode) {
		state.Mode = s.defaultMode
	}
	if state.Requests == nil {
		state.Requests = []Request{}
	}
	s.state = state
	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("创建审批状态目录失败：%w", err)
	}
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("编码审批状态失败：%w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), ".approval-state-*.tmp")
	if err != nil {
		return fmt.Errorf("创建审批状态临时文件失败：%w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(raw); err != nil {
		_ = temp.Close()
		return fmt.Errorf("写入审批状态失败：%w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("同步审批状态失败：%w", err)
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(tempName, s.path); err != nil {
		return fmt.Errorf("原子替换审批状态失败：%w", err)
	}
	return nil
}

func (s *Store) indexLocked(id string) int {
	id = strings.TrimSpace(id)
	for i := range s.state.Requests {
		if s.state.Requests[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) effectiveModeLocked() Mode {
	if ValidMode(s.state.Mode) {
		return s.state.Mode
	}
	return ModeAsk
}

func (s *Store) snapshotLocked() Snapshot {
	pending := 0
	for _, item := range s.state.Requests {
		if item.State == StatePending {
			pending++
		}
	}
	mode := s.effectiveModeLocked()
	return Snapshot{Mode: mode, ModeLabel: ModeLabel(mode), UpdatedAt: s.state.UpdatedAt, UpdatedBy: s.state.UpdatedBy, PendingCount: pending}
}

func (s *Store) trimHistoryLocked() {
	const maxRecords = 200
	if len(s.state.Requests) <= maxRecords {
		return
	}
	// 永不自动删除 pending；优先丢弃最老的已结束记录。
	pending := make([]Request, 0, len(s.state.Requests))
	finished := make([]Request, 0, len(s.state.Requests))
	for _, item := range s.state.Requests {
		if item.State == StatePending || item.State == StateApproved {
			pending = append(pending, item)
		} else {
			finished = append(finished, item)
		}
	}
	keepFinished := maxRecords - len(pending)
	if keepFinished < 0 {
		keepFinished = 0
	}
	if len(finished) > keepFinished {
		finished = finished[len(finished)-keepFinished:]
	}
	s.state.Requests = append(finished, pending...)
}

func newID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "APR-" + strings.ToUpper(hex.EncodeToString(b[:])), nil
}

func safeText(value string, maxRunes int) string {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
	runes := []rune(value)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return value
}

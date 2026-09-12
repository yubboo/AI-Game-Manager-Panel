package control

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const DefaultCapabilityLeaseTTL = 30 * time.Second

var (
	ErrLeaseNotFound = errors.New("能力租约不存在")
	ErrLeaseExpired  = errors.New("能力租约已过期")
	ErrLeaseMismatch = errors.New("能力租约与当前操作不匹配")
	ErrLeaseConsumed = errors.New("能力租约已使用，不能重复执行")
)

// CapabilityLease is a short-lived, server-owned execution grant. It stores
// only stable identifiers and the irreversible request fingerprint; raw Tool
// arguments, commands, credentials and tokens never enter the lease record.
type CapabilityLease struct {
	ID          string `json:"id"`
	Scope       string `json:"scope"`
	Tool        string `json:"tool"`
	RunID       string `json:"runId"`
	Principal   string `json:"principal"`
	RequestHash string `json:"requestHash"`
	IssuedAt    int64  `json:"issuedAt"`
	ExpiresAt   int64  `json:"expiresAt"`
	MaxUses     int    `json:"maxUses"`
	Uses        int    `json:"uses"`
}

// CapabilityLeaseStore intentionally stays in memory. A Host restart revokes
// every outstanding lease instead of restoring execution authority from disk.
type CapabilityLeaseStore struct {
	ttl    time.Duration
	now    func() time.Time
	mu     sync.Mutex
	leases map[string]CapabilityLease
}

func NewCapabilityLeaseStore(ttl time.Duration) *CapabilityLeaseStore {
	if ttl <= 0 || ttl > 5*time.Minute {
		ttl = DefaultCapabilityLeaseTTL
	}
	return &CapabilityLeaseStore{
		ttl:    ttl,
		now:    time.Now,
		leases: make(map[string]CapabilityLease),
	}
}

func (s *CapabilityLeaseStore) Issue(scope, tool, runID, principal, requestHash string, maxUses int) (CapabilityLease, error) {
	if s == nil {
		return CapabilityLease{}, errors.New("能力租约服务不可用")
	}
	scope = strings.TrimSpace(scope)
	tool = strings.TrimSpace(tool)
	runID = strings.TrimSpace(runID)
	principal = strings.TrimSpace(principal)
	requestHash = strings.TrimSpace(requestHash)
	if scope == "" || tool == "" || runID == "" || principal == "" || requestHash == "" {
		return CapabilityLease{}, errors.New("能力租约缺少绑定字段")
	}
	if maxUses <= 0 || maxUses > 8 {
		return CapabilityLease{}, errors.New("能力租约使用次数必须在 1-8 之间")
	}
	id, err := newLeaseID()
	if err != nil {
		return CapabilityLease{}, fmt.Errorf("生成能力租约 ID 失败：%w", err)
	}
	now := s.now()
	item := CapabilityLease{
		ID:          id,
		Scope:       scope,
		Tool:        tool,
		RunID:       runID,
		Principal:   principal,
		RequestHash: requestHash,
		IssuedAt:    now.UnixMilli(),
		ExpiresAt:   now.Add(s.ttl).UnixMilli(),
		MaxUses:     maxUses,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	s.leases[item.ID] = item
	return item, nil
}

// Consume validates every binding and atomically spends one use. A mismatch
// never consumes the lease, allowing the exact approved action to continue.
func (s *CapabilityLeaseStore) Consume(id, scope, tool, runID, principal, requestHash string) error {
	if s == nil {
		return errors.New("能力租约服务不可用")
	}
	id = strings.TrimSpace(id)
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.leases[id]
	if !ok {
		return ErrLeaseNotFound
	}
	if now.UnixMilli() >= item.ExpiresAt {
		delete(s.leases, id)
		return ErrLeaseExpired
	}
	if item.Scope != strings.TrimSpace(scope) ||
		item.Tool != strings.TrimSpace(tool) ||
		item.RunID != strings.TrimSpace(runID) ||
		item.Principal != strings.TrimSpace(principal) ||
		item.RequestHash != strings.TrimSpace(requestHash) {
		return ErrLeaseMismatch
	}
	if item.Uses >= item.MaxUses {
		return ErrLeaseConsumed
	}
	item.Uses++
	s.leases[id] = item
	return nil
}

func (s *CapabilityLeaseStore) Revoke(id string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.leases, strings.TrimSpace(id))
}

func (s *CapabilityLeaseStore) pruneLocked(now time.Time) {
	stamp := now.UnixMilli()
	for id, item := range s.leases {
		if stamp >= item.ExpiresAt || item.Uses >= item.MaxUses {
			delete(s.leases, id)
		}
	}
}

func newLeaseID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "LEASE-" + strings.ToUpper(hex.EncodeToString(b[:])), nil
}

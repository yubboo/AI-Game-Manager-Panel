package host

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

type IntelligenceVisibility string

const (
	VisibilityPrivate      IntelligenceVisibility = "private"
	VisibilityGroup        IntelligenceVisibility = "group"
	VisibilityOrganization IntelligenceVisibility = "organization"
)

type MemoryKind string

const (
	MemorySession    MemoryKind = "session"
	MemoryTask       MemoryKind = "task"
	MemoryUser       MemoryKind = "user"
	MemoryServer     MemoryKind = "server"
	MemoryInstance   MemoryKind = "instance"
	MemoryExperience MemoryKind = "experience"
)

type MemorySensitivity string

const (
	MemoryNormal    MemorySensitivity = "normal"
	MemorySensitive MemorySensitivity = "sensitive"
)

type MemoryScope struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type MemoryRecord struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organizationId"`
	GroupID        string                 `json:"groupId,omitempty"`
	OwnerUserID    string                 `json:"ownerUserId"`
	Kind           MemoryKind             `json:"kind"`
	Scope          MemoryScope            `json:"scope"`
	Content        string                 `json:"content"`
	Source         string                 `json:"source"`
	Sensitivity    MemorySensitivity      `json:"sensitivity"`
	Confidence     float64                `json:"confidence"`
	Visibility     IntelligenceVisibility `json:"visibility"`
	Enabled        bool                   `json:"enabled"`
	ExpiresAt      *time.Time             `json:"expiresAt,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type SkillDefinition struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organizationId,omitempty"`
	GroupID        string                 `json:"groupId,omitempty"`
	OwnerUserID    string                 `json:"ownerUserId,omitempty"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Prompt         string                 `json:"prompt"`
	Tags           []string               `json:"tags,omitempty"`
	GameIDs        []string               `json:"gameIds,omitempty"`
	ToolAllowlist  []string               `json:"toolAllowlist,omitempty"`
	Checklist      []string               `json:"checklist,omitempty"`
	Validators     []string               `json:"validators,omitempty"`
	Enabled        bool                   `json:"enabled"`
	Visibility     IntelligenceVisibility `json:"visibility"`
	Builtin        bool                   `json:"builtin"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type ExpertDefinition struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organizationId,omitempty"`
	GroupID        string                 `json:"groupId,omitempty"`
	OwnerUserID    string                 `json:"ownerUserId,omitempty"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Prompt         string                 `json:"prompt"`
	Domains        []string               `json:"domains,omitempty"`
	GameIDs        []string               `json:"gameIds,omitempty"`
	Knowledge      []string               `json:"knowledge,omitempty"`
	SkillIDs       []string               `json:"skillIds,omitempty"`
	ToolAllowlist  []string               `json:"toolAllowlist,omitempty"`
	Checklist      []string               `json:"checklist,omitempty"`
	Validators     []string               `json:"validators,omitempty"`
	RecoveryRules  []string               `json:"recoveryRules,omitempty"`
	Enabled        bool                   `json:"enabled"`
	Visibility     IntelligenceVisibility `json:"visibility"`
	Builtin        bool                   `json:"builtin"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type MemorySaveRequest struct {
	ID          string                 `json:"id,omitempty"`
	Kind        MemoryKind             `json:"kind"`
	Scope       MemoryScope            `json:"scope"`
	Content     string                 `json:"content"`
	Source      string                 `json:"source,omitempty"`
	Sensitivity MemorySensitivity      `json:"sensitivity,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
	Visibility  IntelligenceVisibility `json:"visibility,omitempty"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
}

type SkillSaveRequest struct {
	ID            string                 `json:"id,omitempty"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Prompt        string                 `json:"prompt"`
	Tags          []string               `json:"tags,omitempty"`
	GameIDs       []string               `json:"gameIds,omitempty"`
	ToolAllowlist []string               `json:"toolAllowlist,omitempty"`
	Checklist     []string               `json:"checklist,omitempty"`
	Validators    []string               `json:"validators,omitempty"`
	Enabled       *bool                  `json:"enabled,omitempty"`
	Visibility    IntelligenceVisibility `json:"visibility,omitempty"`
}

type ExpertSaveRequest struct {
	ID            string                 `json:"id,omitempty"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Prompt        string                 `json:"prompt"`
	Domains       []string               `json:"domains,omitempty"`
	GameIDs       []string               `json:"gameIds,omitempty"`
	Knowledge     []string               `json:"knowledge,omitempty"`
	SkillIDs      []string               `json:"skillIds,omitempty"`
	ToolAllowlist []string               `json:"toolAllowlist,omitempty"`
	Checklist     []string               `json:"checklist,omitempty"`
	Validators    []string               `json:"validators,omitempty"`
	RecoveryRules []string               `json:"recoveryRules,omitempty"`
	Enabled       *bool                  `json:"enabled,omitempty"`
	Visibility    IntelligenceVisibility `json:"visibility,omitempty"`
}

type IntelligenceCatalog struct {
	Memories []MemoryRecord     `json:"memories"`
	Skills   []SkillDefinition  `json:"skills"`
	Experts  []ExpertDefinition `json:"experts"`
}

type SystemModuleKnowledge struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category,omitempty"`
	Route    string   `json:"route,omitempty"`
	Status   string   `json:"status,omitempty"`
	Phase    string   `json:"phase,omitempty"`
	Features []string `json:"features,omitempty"`
}

type SystemModuleSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
	Route  string `json:"route,omitempty"`
}

type IntelligenceContext struct {
	Memories      []MemoryRecord          `json:"memories,omitempty"`
	Skills        []SkillDefinition       `json:"skills,omitempty"`
	Experts       []ExpertDefinition      `json:"experts,omitempty"`
	ModuleCatalog []SystemModuleSummary   `json:"moduleCatalog,omitempty"`
	Modules       []SystemModuleKnowledge `json:"modules,omitempty"`
}

type intelligenceStoreData struct {
	Version  int                `json:"version"`
	Memories []MemoryRecord     `json:"memories"`
	Skills   []SkillDefinition  `json:"skills"`
	Experts  []ExpertDefinition `json:"experts"`
}

type IntelligenceStore struct {
	path string
	mu   sync.RWMutex
	now  func() time.Time
}

func NewIntelligenceStore(path string) *IntelligenceStore {
	return &IntelligenceStore{path: filepath.Clean(path), now: func() time.Time { return time.Now().UTC() }}
}

func (s *IntelligenceStore) Ready() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, err := s.loadLocked()
	return err
}

func (s *IntelligenceStore) CatalogFor(organizationID, groupID, userID string) IntelligenceCatalog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := s.loadLocked()
	if err != nil {
		return IntelligenceCatalog{Memories: []MemoryRecord{}, Skills: builtinSkills(), Experts: builtinExperts()}
	}
	now := s.now()
	catalog := IntelligenceCatalog{Memories: []MemoryRecord{}, Skills: builtinSkills(), Experts: builtinExperts()}
	for _, item := range data.Memories {
		if intelligenceVisibleTo(item.OrganizationID, item.GroupID, item.OwnerUserID, item.Visibility, organizationID, groupID, userID) && !memoryExpired(item, now) {
			catalog.Memories = append(catalog.Memories, cloneMemory(item))
		}
	}
	for _, item := range data.Skills {
		if intelligenceVisibleTo(item.OrganizationID, item.GroupID, item.OwnerUserID, item.Visibility, organizationID, groupID, userID) {
			catalog.Skills = append(catalog.Skills, cloneSkill(item))
		}
	}
	for _, item := range data.Experts {
		if intelligenceVisibleTo(item.OrganizationID, item.GroupID, item.OwnerUserID, item.Visibility, organizationID, groupID, userID) {
			catalog.Experts = append(catalog.Experts, cloneExpert(item))
		}
	}
	return catalog
}

// ContextFor returns only intelligence that is safe to send to a configured
// model provider. Sensitive Memory is intentionally excluded here even though
// authorized users can still see/manage it in the local catalog.
func (s *IntelligenceStore) ContextFor(organizationID, groupID, userID string, run RunContext) IntelligenceContext {
	return s.ContextForGoal(organizationID, groupID, userID, run, "")
}

// ContextForGoal retrieves a compact, goal-relevant working set. 0.1.96 used
// to send almost every built-in Skill/Expert whenever GameID was empty. That
// flooded the model with unrelated knowledge and made "intelligence" look like
// a static documentation dump. XiaoYu now always receives the AGMP/autonomy
// baseline, then only the domain knowledge that actually matches the goal/run.
func (s *IntelligenceStore) ContextForGoal(organizationID, groupID, userID string, run RunContext, goal string) IntelligenceContext {
	catalog := s.CatalogFor(organizationID, groupID, userID)
	now := s.now()
	ctx := IntelligenceContext{}
	goalSurface := strings.ToLower(strings.TrimSpace(goal + " " + run.GameID + " " + run.UIRoute))
	eligibleMemories := make([]MemoryRecord, 0, len(catalog.Memories))
	for _, item := range catalog.Memories {
		if !item.Enabled || item.Sensitivity == MemorySensitive || item.Confidence < 0.25 || memoryExpired(item, now) || !memoryMatchesRun(item, userID, run) {
			continue
		}
		eligibleMemories = append(eligibleMemories, item)
	}
	for _, item := range rankMemories(eligibleMemories, goalSurface, run, now) {
		if len(ctx.Memories) >= 24 {
			break
		}
		ctx.Memories = append(ctx.Memories, item)
	}

	for _, item := range rankSkills(catalog.Skills, goalSurface, run.GameID) {
		if len(ctx.Skills) >= 10 {
			break
		}
		ctx.Skills = append(ctx.Skills, item)
	}
	for _, item := range rankExperts(catalog.Experts, goalSurface, run.GameID) {
		if len(ctx.Experts) >= 6 {
			break
		}
		ctx.Experts = append(ctx.Experts, item)
	}
	return ctx
}

type scoredMemory struct {
	item  MemoryRecord
	score int
}

func rankMemories(items []MemoryRecord, surface string, run RunContext, now time.Time) []MemoryRecord {
	scored := make([]scoredMemory, 0, len(items))
	for _, item := range items {
		score := int(item.Confidence * 20)
		switch item.Kind {
		case MemoryInstance:
			if run.InstanceID != "" && item.Scope.ID == run.InstanceID {
				score += 120
			}
		case MemoryServer:
			if run.ServerID != "" && item.Scope.ID == run.ServerID {
				score += 100
			}
		case MemoryTask:
			if run.TaskID != "" && item.Scope.ID == run.TaskID {
				score += 90
			}
		case MemorySession:
			if run.SessionID != "" && item.Scope.ID == run.SessionID {
				score += 80
			}
		case MemoryExperience:
			score += 35
		case MemoryUser:
			score += 25
		}
		score += intelligenceDefinitionScore(surface, item.Content, item.Source)
		age := now.Sub(item.UpdatedAt)
		if !item.UpdatedAt.IsZero() {
			switch {
			case age <= 7*24*time.Hour:
				score += 12
			case age <= 30*24*time.Hour:
				score += 6
			case age <= 180*24*time.Hour:
				score += 2
			}
		}
		scored = append(scored, scoredMemory{item: item, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].item.UpdatedAt.After(scored[j].item.UpdatedAt)
		}
		return scored[i].score > scored[j].score
	})
	out := make([]MemoryRecord, 0, len(scored))
	for _, value := range scored {
		out = append(out, value.item)
	}
	return out
}

type scoredSkill struct {
	item  SkillDefinition
	score int
}
type scoredExpert struct {
	item  ExpertDefinition
	score int
}

func rankSkills(items []SkillDefinition, surface, gameID string) []SkillDefinition {
	scored := make([]scoredSkill, 0, len(items))
	for _, item := range items {
		if !item.Enabled || !definitionMatchesGame(item.GameIDs, gameID) {
			continue
		}
		score := intelligenceDefinitionScore(surface, item.ID, item.Name, item.Description, item.Prompt, strings.Join(item.Tags, " "), strings.Join(item.GameIDs, " "), strings.Join(item.Checklist, " "))
		if item.ID == "builtin.skill.agmp-autonomy" {
			score += 1000
		}
		if item.ID == "builtin.skill.safe-change" {
			score += 50
		}
		if score > 0 {
			scored = append(scored, scoredSkill{item: item, score: score})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	out := make([]SkillDefinition, 0, len(scored))
	for _, v := range scored {
		out = append(out, v.item)
	}
	return out
}

func rankExperts(items []ExpertDefinition, surface, gameID string) []ExpertDefinition {
	scored := make([]scoredExpert, 0, len(items))
	for _, item := range items {
		if !item.Enabled || !definitionMatchesGame(item.GameIDs, gameID) {
			continue
		}
		score := intelligenceDefinitionScore(surface, item.ID, item.Name, item.Description, item.Prompt, strings.Join(item.Domains, " "), strings.Join(item.GameIDs, " "), strings.Join(item.Knowledge, " "))
		if item.ID == "builtin.expert.agmp" {
			score += 1000
		}
		if score > 0 {
			scored = append(scored, scoredExpert{item: item, score: score})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	out := make([]ExpertDefinition, 0, len(scored))
	for _, v := range scored {
		out = append(out, v.item)
	}
	return out
}

func intelligenceDefinitionScore(surface string, parts ...string) int {
	if surface == "" {
		return 1
	}
	score := 0
	for _, raw := range parts {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if strings.Contains(surface, v) {
			score += 20
		}
		for _, token := range strings.FieldsFunc(v, func(r rune) bool {
			return r == ' ' || r == ',' || r == '，' || r == '/' || r == '|' || r == ':' || r == '：' || r == '-' || r == '_' || r == '.'
		}) {
			token = strings.TrimSpace(token)
			if len([]rune(token)) >= 2 && strings.Contains(surface, token) {
				score += 3
			}
		}
	}
	return score
}

func (s *IntelligenceStore) SaveMemory(organizationID, groupID, userID string, request MemorySaveRequest) (MemoryRecord, error) {
	organizationID, groupID, userID = strings.TrimSpace(organizationID), strings.TrimSpace(groupID), strings.TrimSpace(userID)
	if organizationID == "" || userID == "" {
		return MemoryRecord{}, errors.New("memory organization/user scope is required")
	}
	content := strings.TrimSpace(request.Content)
	if content == "" || len([]byte(content)) > 16*1024 {
		return MemoryRecord{}, errors.New("Memory 内容不能为空且不能超过 16KiB")
	}
	if containsSecretMaterial(content + "\n" + request.Source) {
		return MemoryRecord{}, errors.New("Memory 疑似包含密码、Token、API Key 或私钥，拒绝保存")
	}
	if !validMemoryKind(request.Kind) {
		return MemoryRecord{}, errors.New("Memory kind 无效")
	}
	request.Scope.Type = string(request.Kind)
	request.Scope.ID = strings.TrimSpace(request.Scope.ID)
	if request.Scope.ID == "" {
		return MemoryRecord{}, errors.New("Memory scope.id 不能为空")
	}
	visibility := normalizeVisibility(request.Visibility)
	if visibility == VisibilityGroup && groupID == "" {
		return MemoryRecord{}, errors.New("当前账号没有可用 Group scope")
	}
	sensitivity := request.Sensitivity
	if sensitivity == "" {
		sensitivity = MemoryNormal
	}
	if sensitivity != MemoryNormal && sensitivity != MemorySensitive {
		return MemoryRecord{}, errors.New("Memory sensitivity 无效")
	}
	confidence := request.Confidence
	if confidence == 0 {
		confidence = 1
	}
	if confidence < 0 || confidence > 1 {
		return MemoryRecord{}, errors.New("Memory confidence 必须在 0..1")
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return MemoryRecord{}, err
	}
	now := s.now()
	item := MemoryRecord{ID: strings.TrimSpace(request.ID), OrganizationID: organizationID, GroupID: scopeGroupID(visibility, groupID), OwnerUserID: userID, Kind: request.Kind, Scope: request.Scope, Content: content, Source: cleanText(request.Source, 128), Sensitivity: sensitivity, Confidence: confidence, Visibility: visibility, Enabled: enabled, ExpiresAt: request.ExpiresAt, CreatedAt: now, UpdatedAt: now}
	if item.Source == "" {
		item.Source = "user"
	}
	if item.ID == "" {
		item.ID, err = newIntelligenceID("mem")
		if err != nil {
			return MemoryRecord{}, err
		}
		data.Memories = append(data.Memories, item)
	} else {
		idx := memoryIndex(data.Memories, item.ID)
		if idx < 0 {
			return MemoryRecord{}, errors.New("Memory 不存在")
		}
		existing := data.Memories[idx]
		if existing.OrganizationID != organizationID || existing.OwnerUserID != userID {
			return MemoryRecord{}, errors.New("不能修改其他用户或组织的 Memory")
		}
		item.CreatedAt = existing.CreatedAt
		data.Memories[idx] = item
	}
	if err := s.saveLocked(data); err != nil {
		return MemoryRecord{}, err
	}
	return cloneMemory(item), nil
}

func (s *IntelligenceStore) SaveSkill(organizationID, groupID, userID string, request SkillSaveRequest) (SkillDefinition, error) {
	organizationID, groupID, userID = strings.TrimSpace(organizationID), strings.TrimSpace(groupID), strings.TrimSpace(userID)
	if organizationID == "" || userID == "" {
		return SkillDefinition{}, errors.New("skill organization/user scope is required")
	}
	name, prompt := cleanText(request.Name, 160), strings.TrimSpace(request.Prompt)
	if name == "" || prompt == "" || len([]byte(prompt)) > 32*1024 {
		return SkillDefinition{}, errors.New("Skill 名称/Prompt 不能为空且 Prompt 不能超过 32KiB")
	}
	skillSecretSurface := strings.Join(append([]string{name, request.Description, prompt}, append(append(append(append([]string{}, request.Tags...), request.GameIDs...), request.ToolAllowlist...), append(request.Checklist, request.Validators...)...)...), "\n")
	if containsSecretMaterial(skillSecretSurface) {
		return SkillDefinition{}, errors.New("Skill 疑似包含密码、Token、API Key 或私钥，拒绝保存")
	}
	visibility := normalizeVisibility(request.Visibility)
	if visibility == VisibilityGroup && groupID == "" {
		return SkillDefinition{}, errors.New("当前账号没有可用 Group scope")
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return SkillDefinition{}, err
	}
	now := s.now()
	item := SkillDefinition{ID: strings.TrimSpace(request.ID), OrganizationID: organizationID, GroupID: scopeGroupID(visibility, groupID), OwnerUserID: userID, Name: name, Description: cleanText(request.Description, 1000), Prompt: prompt, Tags: cleanList(request.Tags, 32, 80), GameIDs: cleanList(request.GameIDs, 24, 80), ToolAllowlist: cleanList(request.ToolAllowlist, 64, 160), Checklist: cleanList(request.Checklist, 64, 500), Validators: cleanList(request.Validators, 64, 500), Enabled: enabled, Visibility: visibility, CreatedAt: now, UpdatedAt: now}
	if item.ID == "" {
		item.ID, err = newIntelligenceID("skill")
		if err != nil {
			return SkillDefinition{}, err
		}
		data.Skills = append(data.Skills, item)
	} else {
		idx := skillIndex(data.Skills, item.ID)
		if idx < 0 {
			return SkillDefinition{}, errors.New("Skill 不存在")
		}
		existing := data.Skills[idx]
		if existing.OrganizationID != organizationID || existing.OwnerUserID != userID || existing.Builtin {
			return SkillDefinition{}, errors.New("不能修改其他用户、组织或内置 Skill")
		}
		item.CreatedAt = existing.CreatedAt
		data.Skills[idx] = item
	}
	if err := s.saveLocked(data); err != nil {
		return SkillDefinition{}, err
	}
	return cloneSkill(item), nil
}

func (s *IntelligenceStore) SaveExpert(organizationID, groupID, userID string, request ExpertSaveRequest) (ExpertDefinition, error) {
	organizationID, groupID, userID = strings.TrimSpace(organizationID), strings.TrimSpace(groupID), strings.TrimSpace(userID)
	if organizationID == "" || userID == "" {
		return ExpertDefinition{}, errors.New("expert organization/user scope is required")
	}
	name, prompt := cleanText(request.Name, 160), strings.TrimSpace(request.Prompt)
	if name == "" || prompt == "" || len([]byte(prompt)) > 32*1024 {
		return ExpertDefinition{}, errors.New("Expert 名称/Prompt 不能为空且 Prompt 不能超过 32KiB")
	}
	expertParts := []string{name, request.Description, prompt}
	expertParts = append(expertParts, request.Domains...)
	expertParts = append(expertParts, request.GameIDs...)
	expertParts = append(expertParts, request.Knowledge...)
	expertParts = append(expertParts, request.SkillIDs...)
	expertParts = append(expertParts, request.ToolAllowlist...)
	expertParts = append(expertParts, request.Checklist...)
	expertParts = append(expertParts, request.Validators...)
	expertParts = append(expertParts, request.RecoveryRules...)
	if containsSecretMaterial(strings.Join(expertParts, "\n")) {
		return ExpertDefinition{}, errors.New("Expert 疑似包含密码、Token、API Key 或私钥，拒绝保存")
	}
	visibility := normalizeVisibility(request.Visibility)
	if visibility == VisibilityGroup && groupID == "" {
		return ExpertDefinition{}, errors.New("当前账号没有可用 Group scope")
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return ExpertDefinition{}, err
	}
	now := s.now()
	item := ExpertDefinition{ID: strings.TrimSpace(request.ID), OrganizationID: organizationID, GroupID: scopeGroupID(visibility, groupID), OwnerUserID: userID, Name: name, Description: cleanText(request.Description, 1000), Prompt: prompt, Domains: cleanList(request.Domains, 32, 100), GameIDs: cleanList(request.GameIDs, 24, 80), Knowledge: cleanList(request.Knowledge, 128, 1000), SkillIDs: cleanList(request.SkillIDs, 64, 160), ToolAllowlist: cleanList(request.ToolAllowlist, 64, 160), Checklist: cleanList(request.Checklist, 96, 500), Validators: cleanList(request.Validators, 64, 500), RecoveryRules: cleanList(request.RecoveryRules, 96, 800), Enabled: enabled, Visibility: visibility, CreatedAt: now, UpdatedAt: now}
	if item.ID == "" {
		item.ID, err = newIntelligenceID("expert")
		if err != nil {
			return ExpertDefinition{}, err
		}
		data.Experts = append(data.Experts, item)
	} else {
		idx := expertIndex(data.Experts, item.ID)
		if idx < 0 {
			return ExpertDefinition{}, errors.New("Expert 不存在")
		}
		existing := data.Experts[idx]
		if existing.OrganizationID != organizationID || existing.OwnerUserID != userID || existing.Builtin {
			return ExpertDefinition{}, errors.New("不能修改其他用户、组织或内置 Expert")
		}
		item.CreatedAt = existing.CreatedAt
		data.Experts[idx] = item
	}
	if err := s.saveLocked(data); err != nil {
		return ExpertDefinition{}, err
	}
	return cloneExpert(item), nil
}

func (s *IntelligenceStore) Delete(organizationID, userID, kind, id string, allowOrganization bool) error {
	organizationID, userID, kind, id = strings.TrimSpace(organizationID), strings.TrimSpace(userID), strings.ToLower(strings.TrimSpace(kind)), strings.TrimSpace(id)
	if organizationID == "" || userID == "" || id == "" {
		return errors.New("Intelligence 删除范围无效")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	allowed := func(org, owner string, visibility IntelligenceVisibility) bool {
		return org == organizationID && (owner == userID || (allowOrganization && visibility != VisibilityPrivate))
	}
	removed := false
	switch kind {
	case "memory":
		idx := memoryIndex(data.Memories, id)
		if idx >= 0 && allowed(data.Memories[idx].OrganizationID, data.Memories[idx].OwnerUserID, data.Memories[idx].Visibility) {
			data.Memories = append(data.Memories[:idx], data.Memories[idx+1:]...)
			removed = true
		}
	case "skill":
		idx := skillIndex(data.Skills, id)
		if idx >= 0 && !data.Skills[idx].Builtin && allowed(data.Skills[idx].OrganizationID, data.Skills[idx].OwnerUserID, data.Skills[idx].Visibility) {
			data.Skills = append(data.Skills[:idx], data.Skills[idx+1:]...)
			removed = true
		}
	case "expert":
		idx := expertIndex(data.Experts, id)
		if idx >= 0 && !data.Experts[idx].Builtin && allowed(data.Experts[idx].OrganizationID, data.Experts[idx].OwnerUserID, data.Experts[idx].Visibility) {
			data.Experts = append(data.Experts[:idx], data.Experts[idx+1:]...)
			removed = true
		}
	default:
		return errors.New("Intelligence kind 无效")
	}
	if !removed {
		return errors.New("Intelligence 项不存在或没有删除权限")
	}
	return s.saveLocked(data)
}

func (s *IntelligenceStore) loadLocked() (intelligenceStoreData, error) {
	data := intelligenceStoreData{Version: 1, Memories: []MemoryRecord{}, Skills: []SkillDefinition{}, Experts: []ExpertDefinition{}}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return intelligenceStoreData{}, fmt.Errorf("解析 XiaoYu Intelligence Store 失败: %w", err)
	}
	if data.Version != 1 {
		return intelligenceStoreData{}, fmt.Errorf("不支持的 XiaoYu Intelligence Store 版本: %d", data.Version)
	}
	return data, nil
}

func (s *IntelligenceStore) saveLocked(data intelligenceStoreData) error {
	data.Version = 1
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(temp, s.path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

func normalizeVisibility(value IntelligenceVisibility) IntelligenceVisibility {
	switch value {
	case VisibilityGroup, VisibilityOrganization:
		return value
	default:
		return VisibilityPrivate
	}
}
func scopeGroupID(v IntelligenceVisibility, groupID string) string {
	if v == VisibilityGroup {
		return groupID
	}
	return ""
}
func intelligenceVisibleTo(org, group, owner string, visibility IntelligenceVisibility, wantedOrg, wantedGroup, userID string) bool {
	if org != wantedOrg {
		return false
	}
	switch visibility {
	case VisibilityOrganization:
		return true
	case VisibilityGroup:
		return group != "" && group == wantedGroup
	default:
		return owner == userID
	}
}
func validMemoryKind(v MemoryKind) bool {
	switch v {
	case MemorySession, MemoryTask, MemoryUser, MemoryServer, MemoryInstance, MemoryExperience:
		return true
	}
	return false
}
func memoryExpired(v MemoryRecord, now time.Time) bool {
	return v.ExpiresAt != nil && !v.ExpiresAt.After(now)
}
func memoryMatchesRun(v MemoryRecord, userID string, run RunContext) bool {
	switch v.Kind {
	case MemoryUser, MemoryExperience:
		return v.Scope.ID == userID || v.Scope.ID == "*"
	case MemorySession:
		return run.SessionID != "" && v.Scope.ID == run.SessionID
	case MemoryTask:
		return run.TaskID != "" && v.Scope.ID == run.TaskID
	case MemoryServer:
		return run.ServerID != "" && v.Scope.ID == run.ServerID
	case MemoryInstance:
		return run.InstanceID != "" && v.Scope.ID == run.InstanceID
	default:
		return false
	}
}
func definitionMatchesGame(gameIDs []string, gameID string) bool {
	if len(gameIDs) == 0 || strings.TrimSpace(gameID) == "" {
		return true
	}
	for _, candidate := range gameIDs {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(gameID)) {
			return true
		}
	}
	return false
}
func newIntelligenceID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(b), nil
}
func cleanText(v string, max int) string {
	v = strings.TrimSpace(v)
	if len([]byte(v)) <= max {
		return v
	}
	b := []byte(v)
	return strings.TrimSpace(string(b[:max]))
}
func cleanList(values []string, maxItems, maxBytes int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, minInt(len(values), maxItems))
	for _, raw := range values {
		if len(out) >= maxItems {
			break
		}
		v := cleanText(raw, maxBytes)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func memoryIndex(v []MemoryRecord, id string) int {
	for i := range v {
		if v[i].ID == id {
			return i
		}
	}
	return -1
}
func skillIndex(v []SkillDefinition, id string) int {
	for i := range v {
		if v[i].ID == id {
			return i
		}
	}
	return -1
}
func expertIndex(v []ExpertDefinition, id string) int {
	for i := range v {
		if v[i].ID == id {
			return i
		}
	}
	return -1
}
func cloneMemory(v MemoryRecord) MemoryRecord { return v }
func cloneSkill(v SkillDefinition) SkillDefinition {
	v.Tags = append([]string(nil), v.Tags...)
	v.GameIDs = append([]string(nil), v.GameIDs...)
	v.ToolAllowlist = append([]string(nil), v.ToolAllowlist...)
	v.Checklist = append([]string(nil), v.Checklist...)
	v.Validators = append([]string(nil), v.Validators...)
	return v
}
func cloneExpert(v ExpertDefinition) ExpertDefinition {
	v.Domains = append([]string(nil), v.Domains...)
	v.GameIDs = append([]string(nil), v.GameIDs...)
	v.Knowledge = append([]string(nil), v.Knowledge...)
	v.SkillIDs = append([]string(nil), v.SkillIDs...)
	v.ToolAllowlist = append([]string(nil), v.ToolAllowlist...)
	v.Checklist = append([]string(nil), v.Checklist...)
	v.Validators = append([]string(nil), v.Validators...)
	v.RecoveryRules = append([]string(nil), v.RecoveryRules...)
	return v
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:api[_ -]?key|access[_ -]?token|refresh[_ -]?token|authorization|password|passwd|secret)\b\s*[:=]\s*\S{6,}`),
	regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/-]{10,}`),
	regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |PGP )?PRIVATE KEY-----`),
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{12,}`),
}

func containsSecretMaterial(value string) bool {
	for _, p := range secretPatterns {
		if p.MatchString(value) {
			return true
		}
	}
	return false
}

func builtinSkills() []SkillDefinition {
	return []SkillDefinition{
		{ID: "builtin.skill.agmp-autonomy", Name: "AGMP 引导式自主执行", Description: "XiaoYu 的基础执行策略：专业路径优先、通用能力兜底、模型自主判断、Host 审批执行。", Prompt: "你是具备通用能力的 XiaoYu，不是固定功能菜单。先参考相关 Expert/Skill/Memory，再优先走经过 AGMP 调教的 Domain Tool；Domain Tool 缺失、失败或覆盖不了异常时，继续使用 fs.* 等通用 Tool，必要时使用受控 shell.exec 解决。agmp.capability.search 用于发现能力，不是缺少专用按钮时认输的理由。ToolAllowlist 只是推荐工具提示，不限制其他 frame.tools。用户给的是最终目标，除非缺少密码/Token/不可推断选择或 Host 审批，否则持续执行、观察、验证、恢复。", Tags: []string{"agmp", "xiaoyu", "autonomy", "guided-autonomy", "capability", "tool", "shell", "filesystem", "自主", "系统", "工具"}, ToolAllowlist: []string{"agmp.capability.search", "system.info", "fs.stat", "fs.read", "fs.write", "fs.replace", "shell.exec", "memory.remember"}, Checklist: []string{"理解最终目标", "加载相关专家/Skill/记忆作为优先指导", "优先选择领域 Tool", "领域能力不足时组合通用 Tool", "必要时使用受控 Shell 后备能力", "读取真实 Observation", "变更后验证最终状态", "失败时改变策略并恢复", "有复用价值时总结经验"}, Validators: []string{"不能因为缺少某个专用 Tool 就直接认输", "不能把 Expert/Skill 的推荐 Tool 当作硬 allowlist", "不能在没有真实 Observation 的情况下声称操作完成", "审批由 Host 决定，模型不需要为了安全主动装傻"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.settings-ui", Name: "AGMP 设置与界面控制", Description: "AGMP 页面导航、主题、模型、环境、授权等设置类任务。", Prompt: "区分‘给人打开页面’和‘真正修改系统状态’。用户要求修改设置时优先调用 settings.* 领域 Tool；只有用户明确要求查看/打开页面时才 ui.navigate。修改后用读取 Tool 或结构化回执验证。", Tags: []string{"settings", "ui", "theme", "设置", "主题", "界面", "模型", "授权"}, ToolAllowlist: []string{"settings.get", "settings.theme.set", "ui.navigate", "agmp.capability.search"}, Checklist: []string{"识别查看还是修改", "优先领域 Tool", "验证修改后的设置"}, Validators: []string{"不能用 ui.navigate 冒充设置已修改"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.environment-doctor", Name: "运行环境诊断与修复", Description: "SteamCMD、Java、游戏运行环境、路径与依赖的自动诊断/修复流程。", Prompt: "先读取 environment.status/catalog/game_profile 和相关游戏状态，定位真实缺口；用户说‘检查并处理/修好’时，应在审批边界内继续修复，不要检查完后反问要不要继续。安装、切换默认或移除 Runtime 后都重新读取状态验证；结构化 Runtime Tool 无法覆盖特殊异常时再使用通用能力。", Tags: []string{"environment", "steamcmd", "java", "runtime", "环境", "依赖", "修复", "检查"}, ToolAllowlist: []string{"environment.status", "environment.catalog", "environment.game_profile", "environment.resolve_runtime", "environment.set_default_runtime", "environment.remove_runtime", "environment.install_java", "environment.install_steamcmd", "environment.install_system_prerequisite", "agmp.capability.search", "shell.exec"}, Checklist: []string{"读取环境状态", "识别缺失项", "选择最小必要修复", "执行安装/修复", "重新读取状态验证"}, Validators: []string{"检查任务若用户同时要求处理，不得在发现可修复缺口后主动停止"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.safe-change", Name: "安全变更", Description: "任何重要配置/更新/迁移前先观察、预检、建立恢复点并验证。", Prompt: "重要变更必须先读取真实状态，明确影响范围；可备份时先建立恢复点；执行后用结构化状态、日志或端口进行验证。", Tags: []string{"safety", "change"}, Checklist: []string{"读取当前状态", "确认目标与影响范围", "建立可验证恢复点", "执行最小必要变更", "验证结果"}, Validators: []string{"不能仅凭 Tool 返回 success 判断用户目标完成"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.game-server-one-click-deploy", Name: "游戏服务器一键部署", Description: "从用户的一句话目标出发，自动完成预检、依赖、安装、配置、启动、验证与安全恢复的通用开服流程。", Prompt: "用户要求‘开服/部署服务器/把服务器弄好’时，把最终可用状态作为目标，而不是只做安装。先识别游戏与实例，再做 Preflight（运行环境、SteamCMD/Java、磁盘、端口、现有数据）；已有实例先保护数据与配置。安装/更新后生成或校验实例配置，启动真实进程，并用状态、日志、端口、游戏就绪信号验证。未 Ready 时进入诊断/恢复循环。只有真实 Ready 才 complete。需要 Token/密码等用户独占凭据时才 wait；需要风险审批时交给 Host Approval。", Tags: []string{"deploy", "server", "one-click", "steamcmd", "java", "dst", "minecraft", "开服", "一键部署", "服务器", "部署"}, ToolAllowlist: []string{"agmp.capability.search", "system.info", "environment.status", "environment.catalog", "environment.game_profile", "environment.install_java", "environment.install_steamcmd", "games.list", "backup.list"}, Checklist: []string{"识别目标游戏与实例", "执行环境/依赖/磁盘/端口预检", "保护已有存档与配置", "补齐 SteamCMD/Java 等依赖", "安装或更新服务端", "创建/校验实例配置", "启动服务器", "读取真实状态/日志/端口", "未 Ready 则诊断并安全恢复", "Ready 后再完成"}, Validators: []string{"不能把下载完成当成开服完成", "不能只看到进程存在就判断游戏 Ready", "变更已有实例前优先确保可恢复性", "失败后先诊断并尝试安全恢复，不能立刻把步骤甩回用户"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.dst-operations", Name: "DST 运维", Description: "饥荒联机版部署、更新、Cluster 与分片运维。", Prompt: "按 DST 专用服务器、Cluster、Master/Caves 分片的真实结构操作；更新/迁移前保护存档和 cluster 配置。", Tags: []string{"dst", "steam"}, GameIDs: []string{"steam.dst", "dst"}, ToolAllowlist: []string{"games.list", "steam.status", "backup.list"}, Checklist: []string{"确认 Steam/DST 安装", "确认 Cluster 路径", "确认 Master/Caves 状态", "保护存档", "根据日志验证 Ready"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.minecraft-operations", Name: "Minecraft 运维", Description: "Minecraft Java 版本、服务端、插件与日志运维。", Prompt: "先识别服务端类型和 Minecraft 版本，再匹配 Java Runtime；任何升级必须保护 world/config/plugin 数据并验证启动日志。", Tags: []string{"minecraft", "java"}, GameIDs: []string{"minecraft"}, Checklist: []string{"识别 MC/服务端版本", "匹配 Java Runtime", "备份世界与配置", "验证 EULA/端口/启动日志"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.linux-service", Name: "Linux 服务运维", Description: "Headless Linux、systemd、权限、端口与反向代理检查。", Prompt: "Linux 生产环境优先使用最小权限、明确工作目录、systemd 状态和端口监听；不要假设存在图形桌面。", Tags: []string{"linux", "systemd"}, Checklist: []string{"检查发行版/架构", "检查用户权限", "检查磁盘/端口", "检查服务状态和日志"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.network-diagnosis", Name: "网络诊断", Description: "端口、监听、反向代理、域名与公网访问诊断。", Prompt: "先区分进程监听、本机访问、局域网、公网、DNS、TLS、反代各层，不用单一 ping 结果代替完整判断。", Tags: []string{"network", "proxy"}, Checklist: []string{"监听端口", "本机连通", "防火墙", "反向代理", "DNS/TLS"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.skill.backup-recovery", Name: "备份恢复", Description: "备份创建、校验、恢复和回滚。", Prompt: "恢复前必须识别备份来源/时间/目标，避免覆盖错误实例；重要恢复应先保护当前状态，并在恢复后做完整性验证。", Tags: []string{"backup", "recovery"}, Checklist: []string{"确认目标", "确认备份可读", "保护当前状态", "执行恢复", "验证恢复结果"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
	}
}

func builtinExperts() []ExpertDefinition {
	return []ExpertDefinition{
		{ID: "builtin.expert.agmp", Name: "AGMP 产品与运维专家", Description: "理解 AGMP 模块、页面、Domain Tool、通用能力、审批与 XiaoYu Guided Autonomy。", Prompt: "你提供 AGMP 专业知识，但不能把产品 UI/模块状态误当成 XiaoYu 的智能边界。frame.tools 是本轮真实可请求能力；优先 Domain Tool，缺失时允许通用 fs/process/network/shell 能力完成同一目标。skeleton/planned 只说明没有专门产品实现，不等于通用 Agent 一定做不了。需要发现能力时使用 agmp.capability.search。", Domains: []string{"AGMP", "XiaoYu", "UI", "Settings", "Deployment", "Environment", "Games", "Tools", "Agent Runtime"}, Knowledge: []string{"模型提供 XiaoYu 的通用智能，AGMP 提供持久身份、工具、记忆和专业增强", "Expert/Skill/Memory 是优先指导而不是能力限制", "Domain Tool 优先，但通用 Tool 与 shell.exec 是合法后备能力", "小鱼真实执行必须经过 Host Tool/RBAC/审批/Sandbox，不能模拟成功", "configs/modules.json 是产品模块与实现阶段地图"}, SkillIDs: []string{"builtin.skill.agmp-autonomy", "builtin.skill.settings-ui", "builtin.skill.safe-change"}, ToolAllowlist: []string{"agmp.capability.search", "system.info", "fs.stat", "fs.read", "fs.write", "fs.replace", "shell.exec"}, Checklist: []string{"先理解目标", "加载专业指导", "优先领域能力", "必要时使用通用能力", "完成后验证"}, Validators: []string{"禁止把 skeleton 模块描述成已有 Domain 实现", "禁止把缺少专用按钮等同于 XiaoYu 无能力", "禁止把导航页面描述成业务动作已经完成"}, RecoveryRules: []string{"能力不明确先搜索 Capability", "Domain Tool 缺失或失败时组合通用 Tool", "未知异常可使用受控 shell.exec 诊断和处理", "只有所有合法路径都不足时才明确阻塞点"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.environment", Name: "AGMP 运行环境专家", Description: "SteamCMD、Java、运行库、游戏 Profile、路径与安装状态专家。", Prompt: "以 environment.* Tool 的结构化状态为准。诊断后如果用户目标包含‘修复/处理/弄好/安装’，继续调用对应安装 Tool；只有缺少需要用户提供的凭据或 Host 审批时暂停。", Domains: []string{"Environment", "SteamCMD", "Java", "Runtime", "Paths"}, SkillIDs: []string{"builtin.skill.environment-doctor", "builtin.skill.safe-change"}, Validators: []string{"安装后必须重新读取 environment.status 或对应 profile 验证"}, RecoveryRules: []string{"安装失败先分类网络/权限/路径/依赖，不要直接终止整个任务"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.dst", Name: "饥荒联机版专家", Description: "DST 专用服务器、Steam、Cluster、Master/Caves、Mod 与日志专家。", Prompt: "你提供 DST 领域判断，但必须以 AGMP Tool Observation 的当前机器真实状态为准。不要把通用 Steam 客户端路径误认为专用服务器或 Cluster 路径。", Domains: []string{"Don't Starve Together", "SteamCMD", "Cluster", "Master/Caves"}, GameIDs: []string{"steam.dst", "dst"}, Knowledge: []string{"DST 专用服务器由 Steam App 343050 管理", "Cluster 存档与专用服务器程序目录是不同资源", "Master/Caves 分片必须分别观察真实状态"}, SkillIDs: []string{"builtin.skill.game-server-one-click-deploy", "builtin.skill.dst-operations", "builtin.skill.safe-change", "builtin.skill.backup-recovery"}, Checklist: []string{"确认专用服务器安装路径", "确认 Cluster 路径与 token 状态", "确认分片启动参数", "查看日志 Ready/错误信号"}, Validators: []string{"Steam 更新进度不能在底层任务未结束时虚报 100%", "服务器启动成功必须有进程/日志/端口等真实证据"}, RecoveryRules: []string{"更新失败先保留 Cluster，不删除用户存档", "分片异常时分别检查 Master/Caves 日志"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.minecraft", Name: "Minecraft 专家", Description: "Minecraft Java/Paper/Folia/插件/世界与服务端部署专家。", Prompt: "先确认 Minecraft 与服务端实现版本，再决定 Java、启动参数和插件兼容策略。未知插件行为必须观察实际日志/配置，不靠猜测。", Domains: []string{"Minecraft", "Paper", "Folia", "Java", "Plugins"}, GameIDs: []string{"minecraft"}, Knowledge: []string{"Java Runtime 兼容性取决于具体 Minecraft/服务端版本", "世界、插件和配置属于需要优先保护的数据"}, SkillIDs: []string{"builtin.skill.game-server-one-click-deploy", "builtin.skill.minecraft-operations", "builtin.skill.safe-change", "builtin.skill.backup-recovery"}, Validators: []string{"启动完成需要真实日志/进程/端口证据"}, RecoveryRules: []string{"升级失败优先回滚 jar/config/plugin 与 world 备份，不直接删除世界"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.steam", Name: "Steam 与部署专家", Description: "SteamCMD、下载、校验、更新和运行环境专家。", Prompt: "区分下载、更新、校验的真实底层生命周期；UI 进度必须跟随实际 Steam 任务，不能提前完成。", Domains: []string{"Steam", "SteamCMD", "Download", "Validate"}, SkillIDs: []string{"builtin.skill.game-server-one-click-deploy", "builtin.skill.safe-change"}, Validators: []string{"Steam 校验未结束时不能宣告完成"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.linux", Name: "Linux 系统专家", Description: "Linux Headless、systemd、文件权限、服务与部署专家。", Prompt: "默认按无桌面的服务器环境思考，使用 AGMP Host 的受控能力；关注服务用户、权限、路径、端口、systemd 和日志。", Domains: []string{"Linux", "systemd", "permissions", "headless"}, SkillIDs: []string{"builtin.skill.linux-service", "builtin.skill.safe-change"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.network", Name: "网络诊断专家", Description: "DNS、端口、防火墙、反代、TLS 与公网访问专家。", Prompt: "逐层定位网络问题，区分 AGMP 本身监听、系统防火墙、NAT/安全组、反代和 DNS/TLS。", Domains: []string{"Network", "DNS", "TLS", "Reverse Proxy"}, SkillIDs: []string{"builtin.skill.network-diagnosis"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
		{ID: "builtin.expert.backup", Name: "备份恢复专家", Description: "备份、校验、恢复、迁移和灾难恢复专家。", Prompt: "把可恢复性作为重要变更的前置条件；恢复操作必须先确认目标和备份身份，避免错误覆盖。", Domains: []string{"Backup", "Restore", "Migration", "Recovery"}, SkillIDs: []string{"builtin.skill.backup-recovery", "builtin.skill.safe-change"}, Enabled: true, Visibility: VisibilityOrganization, Builtin: true},
	}
}

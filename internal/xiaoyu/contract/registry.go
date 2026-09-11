package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	ErrToolNameRequired   = errors.New("xiaoyu tool name is required")
	ErrToolNameInvalid    = errors.New("xiaoyu tool name is invalid")
	ErrToolRiskInvalid    = errors.New("xiaoyu tool risk is invalid")
	ErrToolMetadataTooBig = errors.New("xiaoyu tool metadata exceeds safe limit")
	ErrToolSchemaInvalid  = errors.New("xiaoyu tool schema is invalid")
	ErrToolExists         = errors.New("xiaoyu tool already exists")
	ErrToolNotFound       = errors.New("xiaoyu tool not found")
	ErrToolNoHandler      = errors.New("xiaoyu tool has no handler")
)

var toolNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*\.[a-z][a-z0-9_.-]*$`)

// ToolExecution is the domain-owned result returned to XiaoYu after AGMP has
// completed permission/approval checks.
type ToolExecution struct {
	Summary string
	Data    any
}

// ToolHandler is implemented/wired by the owning AGMP domain. XiaoYu Core may
// plan and request a Tool, but it never receives direct OS/process/file access.
type ToolHandler func(context.Context, map[string]any) (ToolExecution, error)

type registryEntry struct {
	owner   string
	spec    ToolSpec
	handler ToolHandler
}

// Registry is AGMP's canonical Tool catalog and dispatch table. Tool ownership
// stays with Go domain modules; the Rust XiaoYu Brain only consumes contracts.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]registryEntry
}

func New() *Registry { return &Registry{tools: make(map[string]registryEntry)} }

// Register records metadata without an executor. Duplicate names are rejected
// rather than overwritten so one module cannot silently hijack another Tool.
func (r *Registry) Register(spec ToolSpec) error { return r.register("core", spec, nil) }

// RegisterHandler records metadata and the owning domain executor.
func (r *Registry) RegisterHandler(spec ToolSpec, handler ToolHandler) error {
	return r.register("core", spec, handler)
}

func (r *Registry) MustRegisterHandler(spec ToolSpec, handler ToolHandler) {
	if err := r.RegisterHandler(spec, handler); err != nil {
		panic(fmt.Sprintf("register XiaoYu Tool %q: %v", spec.Name, err))
	}
}

// RegisterOwned records a Tool on behalf of one Harness plugin. The owner is
// internal metadata used for reversible plugin unload; it is not exposed to the
// model and cannot change the Tool name/risk contract.
func (r *Registry) RegisterOwned(owner string, spec ToolSpec, handler ToolHandler) error {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return errors.New("xiaoyu tool owner is required")
	}
	return r.register(owner, spec, handler)
}

func (r *Registry) register(owner string, spec ToolSpec, handler ToolHandler) error {
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		return ErrToolNameRequired
	}
	if !toolNamePattern.MatchString(spec.Name) {
		return fmt.Errorf("%w: %s", ErrToolNameInvalid, spec.Name)
	}
	if !validRisk(spec.Risk) {
		return fmt.Errorf("%w: %s", ErrToolRiskInvalid, spec.Risk)
	}
	if len([]byte(spec.Description)) > 4096 || len([]byte(spec.Category)) > 128 || len([]byte(spec.Source)) > 512 {
		return ErrToolMetadataTooBig
	}
	if len(spec.Parameters) > 0 {
		raw, err := json.Marshal(spec.Parameters)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrToolSchemaInvalid, err)
		}
		if len(raw) > 64*1024 {
			return ErrToolMetadataTooBig
		}
		if typeName, ok := spec.Parameters["type"].(string); ok && strings.TrimSpace(typeName) != "" && strings.TrimSpace(typeName) != "object" {
			return fmt.Errorf("%w: root type must be object", ErrToolSchemaInvalid)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[spec.Name]; exists {
		return fmt.Errorf("%w: %s", ErrToolExists, spec.Name)
	}
	r.tools[spec.Name] = registryEntry{owner: owner, spec: spec, handler: handler}
	return nil
}

func validRisk(risk RiskLevel) bool {
	switch risk {
	case RiskRead, RiskOperate, RiskModify, RiskDestructive, RiskSystem:
		return true
	default:
		return false
	}
}

func (r *Registry) Get(name string) (ToolSpec, bool) {
	r.mu.RLock()
	entry, ok := r.tools[strings.TrimSpace(name)]
	r.mu.RUnlock()
	return entry.spec, ok
}

func (r *Registry) List() []ToolSpec {
	r.mu.RLock()
	result := make([]ToolSpec, 0, len(r.tools))
	for _, entry := range r.tools {
		result = append(result, entry.spec)
	}
	r.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// UnregisterOwner revokes every Tool contributed by one plugin. Core Tool
// registrations use owner "core" and are intentionally left alone unless that
// owner is explicitly requested.
func (r *Registry) UnregisterOwner(owner string) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, entry := range r.tools {
		if entry.owner == owner {
			delete(r.tools, name)
		}
	}
}

func (r *Registry) Owner(name string) (string, bool) {
	r.mu.RLock()
	entry, ok := r.tools[strings.TrimSpace(name)]
	r.mu.RUnlock()
	return entry.owner, ok
}

func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (ToolExecution, error) {
	r.mu.RLock()
	entry, ok := r.tools[strings.TrimSpace(name)]
	r.mu.RUnlock()
	if !ok {
		return ToolExecution{}, ErrToolNotFound
	}
	if entry.handler == nil {
		return ToolExecution{}, ErrToolNoHandler
	}
	if args == nil {
		args = map[string]any{}
	}
	if err := ValidateArguments(entry.spec.Parameters, args); err != nil {
		return ToolExecution{}, err
	}
	return entry.handler(ctx, args)
}

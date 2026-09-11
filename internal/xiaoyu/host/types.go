// Package host implements AGMP's XiaoYu Agent Harness and plugin kernel.
//
// Package host is the Go-side compatibility/orchestration layer during the Rust-first
// Agent Runtime migration. It composes AGMP Domain capabilities, owns plugin lifecycle
// and records observable events. New generic runtime execution belongs in Rust.
package host

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

type PluginState string

const (
	StatePending   PluginState = "pending"
	StateLoading   PluginState = "loading"
	StateActive    PluginState = "active"
	StateFailed    PluginState = "failed"
	StateUnloading PluginState = "unloading"
	StateDisposed  PluginState = "disposed"
)

type CapabilityKind string

const (
	CapabilityModel     CapabilityKind = "model"
	CapabilityTool      CapabilityKind = "tool"
	CapabilitySkill     CapabilityKind = "skill"
	CapabilitySession   CapabilityKind = "session"
	CapabilitySandbox   CapabilityKind = "sandbox"
	CapabilityStorage   CapabilityKind = "storage"
	CapabilityLoop      CapabilityKind = "loop"
	CapabilityScheduler CapabilityKind = "scheduler"
	CapabilityUI        CapabilityKind = "ui"
	CapabilityBridge    CapabilityKind = "bridge"
)

const PluginAPIVersion = "xiaoyu.plugin.v1"

type Manifest struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Version      string           `json:"version"`
	APIVersion   string           `json:"apiVersion"`
	Source       string           `json:"source"`
	Kinds        []CapabilityKind `json:"kinds"`
	Requires     []string         `json:"requires,omitempty"`
	Provides     []string         `json:"provides,omitempty"`
	Permissions  []string         `json:"permissions,omitempty"`
	ConfigSchema map[string]any   `json:"configSchema,omitempty"`
	Description  string           `json:"description,omitempty"`
	Experimental bool             `json:"experimental,omitempty"`
}

type PluginSnapshot struct {
	Manifest Manifest    `json:"manifest"`
	State    PluginState `json:"state"`
	Error    string      `json:"error,omitempty"`
}

type CapabilitySnapshot struct {
	Plugins  []PluginSnapshot          `json:"plugins"`
	Services []string                  `json:"services"`
	Tools    []xiaoyucontract.ToolSpec `json:"tools"`
}

type DisposeFunc func(context.Context) error

type Plugin interface {
	Manifest() Manifest
	Apply(context.Context, *Context) (DisposeFunc, error)
}

type PluginFunc struct {
	Meta    Manifest
	ApplyFn func(context.Context, *Context) (DisposeFunc, error)
}

func (p PluginFunc) Manifest() Manifest { return p.Meta }
func (p PluginFunc) Apply(ctx context.Context, host *Context) (DisposeFunc, error) {
	if p.ApplyFn == nil {
		return nil, nil
	}
	return p.ApplyFn(ctx, host)
}

var pluginIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

var (
	ErrPluginIDRequired     = errors.New("xiaoyu plugin id is required")
	ErrPluginIDInvalid      = errors.New("xiaoyu plugin id is invalid")
	ErrPluginAPIUnsupported = errors.New("xiaoyu plugin api version is unsupported")
	ErrCapabilityInvalid    = errors.New("xiaoyu capability kind is invalid")
	ErrPluginExists         = errors.New("xiaoyu plugin already exists")
	ErrPluginNotFound       = errors.New("xiaoyu plugin not found")
	ErrPluginDependency     = errors.New("xiaoyu plugin dependency is unavailable")
	ErrServiceExists        = errors.New("xiaoyu service already exists")
	ErrServiceNameRequired  = errors.New("xiaoyu service name is required")
	ErrUnsupportedPlugin    = errors.New("xiaoyu plugin is not supported by this bridge")
	ErrAgentStepBudget      = errors.New("xiaoyu agent step budget exhausted")
	ErrAgentToolBudget      = errors.New("xiaoyu agent tool-call budget exhausted")
	ErrAgentFailureBudget   = errors.New("xiaoyu agent failure budget exhausted")
	ErrAgentDoomLoop        = errors.New("xiaoyu agent repeated the same action too many times")
	ErrAgentInvalidDecision = errors.New("xiaoyu agent returned an invalid decision")
)

func normalizeManifest(value Manifest) (Manifest, error) {
	value.ID = strings.TrimSpace(value.ID)
	value.Name = strings.TrimSpace(value.Name)
	value.Version = strings.TrimSpace(value.Version)
	value.Source = strings.TrimSpace(value.Source)
	if value.ID == "" {
		return Manifest{}, ErrPluginIDRequired
	}
	if !pluginIDPattern.MatchString(value.ID) {
		return Manifest{}, ErrPluginIDInvalid
	}
	if value.Name == "" {
		value.Name = value.ID
	}
	if value.Version == "" {
		value.Version = "0.0.0"
	}
	if strings.TrimSpace(value.APIVersion) == "" {
		value.APIVersion = PluginAPIVersion
	}
	if value.APIVersion != PluginAPIVersion {
		return Manifest{}, fmt.Errorf("%w: %s", ErrPluginAPIUnsupported, value.APIVersion)
	}
	if err := validateCapabilityKinds(value.Kinds); err != nil {
		return Manifest{}, err
	}
	if value.Source == "" {
		value.Source = "agmp"
	}
	value.Requires = normalizeNames(value.Requires)
	value.Provides = normalizeNames(value.Provides)
	value.Permissions = normalizeNames(value.Permissions)
	return value, nil
}

func validateCapabilityKinds(values []CapabilityKind) error {
	seen := map[CapabilityKind]struct{}{}
	for _, value := range values {
		switch value {
		case CapabilityModel, CapabilityTool, CapabilitySkill, CapabilitySession, CapabilitySandbox, CapabilityStorage, CapabilityLoop, CapabilityScheduler, CapabilityUI, CapabilityBridge:
		default:
			return fmt.Errorf("%w: %s", ErrCapabilityInvalid, value)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
	}
	return nil
}

func normalizeNames(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

type Event struct {
	Sequence uint64         `json:"sequence"`
	Time     time.Time      `json:"time"`
	Type     string         `json:"type"`
	PluginID string         `json:"pluginId,omitempty"`
	RunID    string         `json:"runId,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

type EventHandler func(Event)

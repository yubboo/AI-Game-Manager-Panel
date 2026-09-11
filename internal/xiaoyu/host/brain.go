package host

import (
	"context"
	"errors"
	"strings"
)

const PrimaryBrainService = "brain.primary"

var ErrBrainUnavailable = errors.New("xiaoyu primary brain provider is unavailable")

// BrainInfo is intentionally provider-neutral. XiaoYu Kernel only needs to know
// which reasoning provider is active; API keys and provider-private settings
// never enter the capability graph or Harness trace.
type BrainInfo struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Model        string             `json:"model,omitempty"`
	Source       string             `json:"source,omitempty"`
	Ready        bool               `json:"ready"`
	Message      string             `json:"message,omitempty"`
	Capabilities *ModelCapabilities `json:"capabilities,omitempty"`
}

// BrainProvider is the replaceable reasoning seam used by the autonomous loop.
// A provider may call XiaoYu Rust Core plus any model backend, but it receives
// only a Frame and returns a structured Decision. It never receives direct OS
// handles or bypass access to AGMP Domain Tools.
type BrainProvider interface {
	Brain
	Info(context.Context) BrainInfo
}

// RegisterBrain exposes one primary reasoning provider through the shared
// service context. A second provider cannot silently overwrite it.
func (c *Context) RegisterBrain(provider BrainProvider) error {
	if provider == nil {
		return ErrBrainUnavailable
	}
	return c.RegisterService(PrimaryBrainService, provider)
}

func (k *Kernel) Brain() (BrainProvider, bool) {
	value, ok := k.service(PrimaryBrainService)
	if !ok {
		return nil, false
	}
	provider, ok := value.(BrainProvider)
	return provider, ok && provider != nil
}

// BrainPlugin is a small adapter for built-in or compatibility providers. It
// keeps model-specific implementation out of the Kernel itself.
type BrainPlugin struct {
	Meta     Manifest
	Provider BrainProvider
}

func (p BrainPlugin) Manifest() Manifest {
	meta := p.Meta
	if len(meta.Kinds) == 0 {
		meta.Kinds = []CapabilityKind{CapabilityModel}
	}
	hasPrimary := false
	for _, name := range meta.Provides {
		if strings.TrimSpace(name) == PrimaryBrainService {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary {
		meta.Provides = append(meta.Provides, PrimaryBrainService)
	}
	return meta
}

func (p BrainPlugin) Apply(_ context.Context, host *Context) (DisposeFunc, error) {
	if p.Provider == nil {
		return nil, ErrBrainUnavailable
	}
	return nil, host.RegisterBrain(p.Provider)
}

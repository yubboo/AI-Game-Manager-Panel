package host

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

type serviceEntry struct {
	owner string
	value any
}

type pluginEntry struct {
	plugin  Plugin
	meta    Manifest
	state   PluginState
	lastErr error
	dispose DisposeFunc
}

type Kernel struct {
	mu       sync.RWMutex
	tools    *xiaoyucontract.Registry
	plugins  map[string]*pluginEntry
	services map[string]serviceEntry
	events   *EventBus
	trace    *Trace
}

func New(tools *xiaoyucontract.Registry) *Kernel {
	if tools == nil {
		tools = xiaoyucontract.New()
	}
	return &Kernel{
		tools: tools, plugins: make(map[string]*pluginEntry), services: make(map[string]serviceEntry),
		events: NewEventBus(), trace: NewTrace(),
	}
}

func (k *Kernel) Tools() *xiaoyucontract.Registry { return k.tools }
func (k *Kernel) Events() *EventBus               { return k.events }
func (k *Kernel) Trace() *Trace                   { return k.trace }

// Observe appends a sanitized host/runtime event to XiaoYu's append-only trace.
// Callers must not put secrets, raw credentials or full shell commands in Event.Data.
func (k *Kernel) Observe(event Event) { k.record(event) }

func (k *Kernel) Add(plugin Plugin) error {
	if plugin == nil {
		return ErrPluginNotFound
	}
	meta, err := normalizeManifest(plugin.Manifest())
	if err != nil {
		return err
	}
	k.mu.Lock()
	if _, exists := k.plugins[meta.ID]; exists {
		k.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPluginExists, meta.ID)
	}
	k.plugins[meta.ID] = &pluginEntry{plugin: plugin, meta: meta, state: StatePending}
	k.mu.Unlock()
	k.record(Event{Type: "plugin/added", PluginID: meta.ID, Summary: meta.Name})
	return nil
}

func (k *Kernel) Mount(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	k.mu.Lock()
	entry, ok := k.plugins[id]
	if !ok {
		k.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}
	if entry.state == StateActive {
		k.mu.Unlock()
		return nil
	}
	missing := k.missingServicesLocked(entry.meta.Requires)
	if len(missing) > 0 {
		entry.state = StatePending
		k.mu.Unlock()
		return fmt.Errorf("%w: %s requires %s", ErrPluginDependency, id, strings.Join(missing, ", "))
	}
	entry.state = StateLoading
	entry.lastErr = nil
	plugin := entry.plugin
	k.mu.Unlock()

	k.record(Event{Type: "plugin/loading", PluginID: id})
	dispose, err := plugin.Apply(ctx, &Context{kernel: k, owner: id})

	k.mu.Lock()
	if err != nil {
		entry.state = StateFailed
		entry.lastErr = err
		k.mu.Unlock()
		k.removeOwnerResources(id)
		k.record(Event{Type: "plugin/failed", PluginID: id, Summary: err.Error()})
		return err
	}
	entry.dispose = dispose
	entry.state = StateActive
	k.mu.Unlock()
	k.record(Event{Type: "plugin/active", PluginID: id})
	return nil
}

// MountAll resolves required services until no more plugins can make progress.
// Pending plugins remain visible in CapabilitySnapshot instead of being hidden.
func (k *Kernel) MountAll(ctx context.Context) error {
	var firstErr error
	for {
		progressed := false
		for _, snap := range k.Plugins() {
			if snap.State == StateActive || snap.State == StateFailed || snap.State == StateDisposed {
				continue
			}
			if err := k.Mount(ctx, snap.Manifest.ID); err == nil {
				progressed = true
			} else if !errors.Is(err, ErrPluginDependency) && firstErr == nil {
				firstErr = err
			}
		}
		if !progressed {
			break
		}
	}
	return firstErr
}

func (k *Kernel) Unmount(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	// Required-service consumers unload before their provider.
	for _, dependent := range k.dependentsOf(id) {
		if err := k.Unmount(ctx, dependent); err != nil {
			return err
		}
	}

	k.mu.Lock()
	entry, ok := k.plugins[id]
	if !ok {
		k.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}
	if entry.state == StateDisposed || entry.state == StatePending {
		entry.state = StateDisposed
		k.mu.Unlock()
		k.removeOwnerResources(id)
		return nil
	}
	entry.state = StateUnloading
	dispose := entry.dispose
	k.mu.Unlock()
	k.record(Event{Type: "plugin/unloading", PluginID: id})

	var err error
	if dispose != nil {
		err = dispose(ctx)
	}
	k.removeOwnerResources(id)
	k.mu.Lock()
	entry.dispose = nil
	entry.lastErr = err
	entry.state = StateDisposed
	k.mu.Unlock()
	k.record(Event{Type: "plugin/disposed", PluginID: id})
	return err
}

// Forget removes an inactive plugin entry after all reversible resources have
// been released. This lets a compatibility plugin be reloaded with a newer
// version or a different AGMP policy without retaining stale handlers.
func (k *Kernel) Forget(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}
	k.mu.Lock()
	entry, ok := k.plugins[id]
	if !ok {
		k.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
	}
	if entry.state == StateActive || entry.state == StateLoading || entry.state == StateUnloading {
		k.mu.Unlock()
		return fmt.Errorf("xiaoyu plugin must be inactive before forget: %s", id)
	}
	delete(k.plugins, id)
	k.mu.Unlock()
	k.removeOwnerResources(id)
	k.record(Event{Type: "plugin/forgotten", PluginID: id})
	return nil
}

func (k *Kernel) Plugins() []PluginSnapshot {
	k.mu.RLock()
	result := make([]PluginSnapshot, 0, len(k.plugins))
	for _, entry := range k.plugins {
		snap := PluginSnapshot{Manifest: entry.meta, State: entry.state}
		if entry.lastErr != nil {
			snap.Error = entry.lastErr.Error()
		}
		result = append(result, snap)
	}
	k.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].Manifest.ID < result[j].Manifest.ID })
	return result
}

func (k *Kernel) Capabilities() CapabilitySnapshot {
	k.mu.RLock()
	services := make([]string, 0, len(k.services))
	for name := range k.services {
		services = append(services, name)
	}
	k.mu.RUnlock()
	sort.Strings(services)
	return CapabilitySnapshot{Plugins: k.Plugins(), Services: services, Tools: k.tools.List()}
}

func (k *Kernel) registerService(owner, name string, value any) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrServiceNameRequired
	}
	k.mu.Lock()
	if existing, ok := k.services[name]; ok && existing.owner != owner {
		k.mu.Unlock()
		return fmt.Errorf("%w: %s (owner=%s)", ErrServiceExists, name, existing.owner)
	}
	k.services[name] = serviceEntry{owner: owner, value: value}
	k.mu.Unlock()
	k.record(Event{Type: "service/registered", PluginID: owner, Summary: name})
	return nil
}

func (k *Kernel) service(name string) (any, bool) {
	k.mu.RLock()
	entry, ok := k.services[strings.TrimSpace(name)]
	k.mu.RUnlock()
	return entry.value, ok
}

func (k *Kernel) missingServicesLocked(names []string) []string {
	var missing []string
	for _, name := range names {
		if _, ok := k.services[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

func (k *Kernel) dependentsOf(owner string) []string {
	k.mu.RLock()
	provided := map[string]struct{}{}
	for name, service := range k.services {
		if service.owner == owner {
			provided[name] = struct{}{}
		}
	}
	var result []string
	for id, entry := range k.plugins {
		if id == owner || entry.state != StateActive {
			continue
		}
		for _, required := range entry.meta.Requires {
			if _, ok := provided[required]; ok {
				result = append(result, id)
				break
			}
		}
	}
	k.mu.RUnlock()
	sort.Strings(result)
	return result
}

func (k *Kernel) removeOwnerResources(owner string) {
	k.tools.UnregisterOwner(owner)
	k.events.RemoveOwner(owner)
	k.mu.Lock()
	for name, entry := range k.services {
		if entry.owner == owner {
			delete(k.services, name)
		}
	}
	k.mu.Unlock()
}

func (k *Kernel) record(event Event) {
	if event.Type == "" {
		return
	}
	event = k.trace.Append(event)
	k.events.Emit(event)
}

type Context struct {
	kernel *Kernel
	owner  string
}

func (c *Context) PluginID() string { return c.owner }
func (c *Context) RegisterTool(spec xiaoyucontract.ToolSpec, handler xiaoyucontract.ToolHandler) error {
	return c.kernel.tools.RegisterOwned(c.owner, spec, handler)
}
func (c *Context) RegisterService(name string, value any) error {
	return c.kernel.registerService(c.owner, name, value)
}
func (c *Context) Service(name string) (any, bool) { return c.kernel.service(name) }
func (c *Context) On(eventType string, handler EventHandler) func() {
	return c.kernel.events.On(c.owner, eventType, handler)
}
func (c *Context) Emit(event Event) {
	event.PluginID = c.owner
	c.kernel.record(event)
}

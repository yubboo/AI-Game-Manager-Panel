package game

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	ErrInvalidProvider   = errors.New("invalid game provider")
	ErrDuplicateProvider = errors.New("game provider already registered")
)

// Registry is the single catalog of game modules known to AI Game Manager Panel.
// Core/application code queries this registry instead of hard-coding DST, Palworld or Minecraft branches.
type Registry struct {
	mu        sync.RWMutex
	providers map[ID]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[ID]Provider)}
}

func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return ErrInvalidProvider
	}
	metadata := provider.Metadata()
	if metadata.ID == "" || metadata.Name == "" || metadata.Family == "" {
		return fmt.Errorf("%w: missing id, name or family", ErrInvalidProvider)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[metadata.ID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateProvider, metadata.ID)
	}
	r.providers[metadata.ID] = provider
	return nil
}

func (r *Registry) Get(id ID) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[id]
	return provider, ok
}

func (r *Registry) List() []Metadata {
	r.mu.RLock()
	result := make([]Metadata, 0, len(r.providers))
	for _, provider := range r.providers {
		result = append(result, provider.Metadata())
	}
	r.mu.RUnlock()

	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}

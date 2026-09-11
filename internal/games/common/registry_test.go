package game

import (
	"errors"
	"testing"
)

type testProvider struct{ metadata Metadata }

func (p testProvider) Metadata() Metadata { return p.metadata }

func TestRegistryRegisterAndList(t *testing.T) {
	registry := NewRegistry()
	provider := testProvider{metadata: Metadata{
		ID:           "steam.example",
		Name:         "Example Server",
		Family:       FamilySteam,
		Capabilities: []Capability{CapabilityInstall, CapabilityRuntime},
	}}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("register: %v", err)
	}
	if registry.Count() != 1 {
		t.Fatalf("expected 1 provider, got %d", registry.Count())
	}
	items := registry.List()
	if len(items) != 1 || items[0].ID != "steam.example" {
		t.Fatalf("unexpected registry items: %#v", items)
	}
	if !items[0].Supports(CapabilityRuntime) {
		t.Fatal("runtime capability should be reported")
	}
}

func TestRegistryRejectsDuplicate(t *testing.T) {
	registry := NewRegistry()
	provider := testProvider{metadata: Metadata{ID: "steam.example", Name: "Example", Family: FamilySteam}}
	if err := registry.Register(provider); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := registry.Register(provider); !errors.Is(err, ErrDuplicateProvider) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

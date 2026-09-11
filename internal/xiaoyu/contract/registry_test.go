package contract

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRegistryListStableOrder(t *testing.T) {
	r := New()
	if err := r.Register(ToolSpec{Name: "steam.validate", Risk: RiskOperate}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(ToolSpec{Name: "games.list", Risk: RiskRead}); err != nil {
		t.Fatal(err)
	}
	items := r.List()
	if len(items) != 2 || items[0].Name != "games.list" || items[1].Name != "steam.validate" {
		t.Fatalf("unexpected tool order: %#v", items)
	}
}

func TestRegistryRejectsDuplicateAndInvalidTools(t *testing.T) {
	r := New()
	spec := ToolSpec{Name: "files.read", Risk: RiskRead}
	if err := r.Register(spec); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(spec); !errors.Is(err, ErrToolExists) {
		t.Fatalf("duplicate tool should be rejected, got %v", err)
	}
	if err := r.Register(ToolSpec{Name: "Bad Tool", Risk: RiskRead}); !errors.Is(err, ErrToolNameInvalid) {
		t.Fatalf("invalid name should be rejected, got %v", err)
	}
	if err := r.Register(ToolSpec{Name: "files.write", Risk: RiskLevel("unknown")}); !errors.Is(err, ErrToolRiskInvalid) {
		t.Fatalf("invalid risk should be rejected, got %v", err)
	}
}

func TestRegistryExecutesOnlyRegisteredHandler(t *testing.T) {
	r := New()
	if err := r.RegisterHandler(ToolSpec{Name: "system.info", Risk: RiskRead}, func(_ context.Context, args map[string]any) (ToolExecution, error) {
		return ToolExecution{Summary: "ok", Data: args["value"]}, nil
	}); err != nil {
		t.Fatal(err)
	}
	value, err := r.Execute(context.Background(), "system.info", map[string]any{"value": 7})
	if err != nil || value.Summary != "ok" || value.Data != 7 {
		t.Fatalf("unexpected execution: %+v err=%v", value, err)
	}
	if _, err := r.Execute(context.Background(), "missing.tool", nil); !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("missing tool should fail, got %v", err)
	}
}

func TestRegistryOwnerCanBeRevoked(t *testing.T) {
	r := New()
	if err := r.RegisterOwned("plugin.demo", ToolSpec{Name: "demo.read", Risk: RiskRead}, func(context.Context, map[string]any) (ToolExecution, error) {
		return ToolExecution{Summary: "ok"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if owner, ok := r.Owner("demo.read"); !ok || owner != "plugin.demo" {
		t.Fatalf("unexpected owner %q %v", owner, ok)
	}
	r.UnregisterOwner("plugin.demo")
	if _, ok := r.Get("demo.read"); ok {
		t.Fatal("tool should be removed with plugin")
	}
}

func TestRegistryValidatesArgumentsBeforeHandler(t *testing.T) {
	called := false
	registry := New()
	err := registry.RegisterHandler(ToolSpec{
		Name: "demo.write", Risk: RiskModify,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":  map[string]any{"type": "string"},
				"force": map[string]any{"type": "boolean"},
			},
			"required":             []string{"path"},
			"additionalProperties": false,
		},
	}, func(context.Context, map[string]any) (ToolExecution, error) {
		called = true
		return ToolExecution{Summary: "ok"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Execute(context.Background(), "demo.write", map[string]any{"force": true}); !errors.Is(err, ErrToolArgumentsInvalid) {
		t.Fatalf("expected required-field failure, got %v", err)
	}
	if called {
		t.Fatal("handler must not run after schema rejection")
	}
	if _, err := registry.Execute(context.Background(), "demo.write", map[string]any{"path": "a", "extra": true}); !errors.Is(err, ErrToolArgumentsInvalid) {
		t.Fatalf("expected additional-property failure, got %v", err)
	}
	if _, err := registry.Execute(context.Background(), "demo.write", map[string]any{"path": "a", "force": true}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("valid arguments should reach handler")
	}
}

func TestRegistryRejectsOversizedOrNonObjectToolSchemas(t *testing.T) {
	registry := New()
	if err := registry.RegisterHandler(ToolSpec{Name: "demo.badroot", Risk: RiskRead, Parameters: map[string]any{"type": "string"}}, func(context.Context, map[string]any) (ToolExecution, error) { return ToolExecution{}, nil }); !errors.Is(err, ErrToolSchemaInvalid) {
		t.Fatalf("non-object root error=%v", err)
	}
	if err := registry.RegisterHandler(ToolSpec{Name: "demo.big", Description: strings.Repeat("x", 4097), Risk: RiskRead}, func(context.Context, map[string]any) (ToolExecution, error) { return ToolExecution{}, nil }); !errors.Is(err, ErrToolMetadataTooBig) {
		t.Fatalf("oversized metadata error=%v", err)
	}
}

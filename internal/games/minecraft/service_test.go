package minecraft

import (
	"context"
	"net"
	"testing"
)

func TestPlanRejectsMissingOnlineModeBeforeFactLookup(t *testing.T) {
	s := New(Options{})
	_, err := s.Plan(context.Background(), PlanRequest{Name: "test", Software: SoftwarePaper})
	if err == nil {
		t.Fatal("expected missing online-mode to fail closed")
	}
}

func TestPlanRejectsOfflineWithoutWhitelistBeforeFactLookup(t *testing.T) {
	offline := false
	s := New(Options{})
	_, err := s.Plan(context.Background(), PlanRequest{Name: "test", Software: SoftwarePaper, OnlineMode: &offline})
	if err == nil {
		t.Fatal("expected offline mode without whitelist to fail closed")
	}
}

func TestEnsurePortAvailableRejectsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := ensurePortAvailable(port); err == nil {
		t.Fatal("expected occupied port to be rejected")
	}
}

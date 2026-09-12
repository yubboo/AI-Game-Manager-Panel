package control

import (
	"errors"
	"testing"
	"time"
)

func TestCapabilityLeaseIsBoundSingleUseAndNotReplayable(t *testing.T) {
	store := NewCapabilityLeaseStore(30 * time.Second)
	lease, err := store.Issue("native-terminal", "shell.exec", "RUN-1", "ORG-1:USR-1", "hash-a", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-1", "ORG-1:USR-1", "hash-b"); !errors.Is(err, ErrLeaseMismatch) {
		t.Fatalf("different payload must not consume lease: %v", err)
	}
	if err := store.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-1", "ORG-1:USR-1", "hash-a"); err != nil {
		t.Fatalf("exact approved action should consume lease: %v", err)
	}
	if err := store.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-1", "ORG-1:USR-1", "hash-a"); !errors.Is(err, ErrLeaseNotFound) && !errors.Is(err, ErrLeaseConsumed) {
		t.Fatalf("spent lease must not replay: %v", err)
	}
}

func TestCapabilityLeaseExpiresAndDoesNotPersistAuthority(t *testing.T) {
	store := NewCapabilityLeaseStore(time.Second)
	now := time.Unix(1_700_000_000, 0)
	store.now = func() time.Time { return now }
	lease, err := store.Issue("native-terminal", "shell.exec", "RUN-2", "ORG-1:USR-1", "hash", 1)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if err := store.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-2", "ORG-1:USR-1", "hash"); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired lease must fail closed: %v", err)
	}
	freshStore := NewCapabilityLeaseStore(time.Second)
	if err := freshStore.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-2", "ORG-1:USR-1", "hash"); !errors.Is(err, ErrLeaseNotFound) {
		t.Fatalf("host restart/new store must revoke old lease: %v", err)
	}
}

func TestCapabilityLeaseBindsRunAndPrincipal(t *testing.T) {
	store := NewCapabilityLeaseStore(30 * time.Second)
	lease, err := store.Issue("native-terminal", "shell.exec", "RUN-3", "ORG-1:USR-1", "hash", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-other", "ORG-1:USR-1", "hash"); !errors.Is(err, ErrLeaseMismatch) {
		t.Fatalf("lease must not cross Runs: %v", err)
	}
	if err := store.Consume(lease.ID, "native-terminal", "shell.exec", "RUN-3", "ORG-1:USR-other", "hash"); !errors.Is(err, ErrLeaseMismatch) {
		t.Fatalf("lease must not cross principals: %v", err)
	}
}

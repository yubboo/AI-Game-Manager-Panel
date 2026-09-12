package minecraft

import "testing"

func TestRuntimeReadyRequiresDoneAndHelpMarker(t *testing.T) {
	p := &runtimeProcess{id: "mc", state: "running", logs: []LogLine{}}
	p.consume(`[Server thread/INFO]: Done (1.234s)!`)
	if p.snapshot().Ready {
		t.Fatal("Done without help marker must not mark server ready")
	}
	p.consume(`[Server thread/INFO]: For help, type "help"`)
	if p.snapshot().Ready {
		t.Fatal("markers split across different lines must not mark server ready")
	}
	p.consume(`[Server thread/INFO]: Done (1.234s)! For help, type "help"`)
	if !p.snapshot().Ready {
		t.Fatal("canonical Done + help marker should mark server ready")
	}
}

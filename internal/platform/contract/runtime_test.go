package platformcontract

import "testing"

func TestForSeparatesControlSurfacesFromExecutionNode(t *testing.T) {
	value := For("linux", "amd64")
	if value.HostOS != "linux" || value.NodeKind != "linux-native" || !value.NativeExecution {
		t.Fatalf("unexpected runtime contract: %+v", value)
	}
	var web, linuxRuntime *Surface
	for i := range value.Surfaces {
		s := &value.Surfaces[i]
		if s.ID == "web" {
			web = s
		}
		if s.ID == "linux-runtime" {
			linuxRuntime = s
		}
	}
	if web == nil || web.ExecutesOnNode || !web.ControlsNodes || !web.Browser {
		t.Fatalf("web surface must stay control-only: %+v", web)
	}
	if linuxRuntime == nil || !linuxRuntime.ExecutesOnNode || linuxRuntime.ControlsNodes {
		t.Fatalf("linux runtime must be native execution surface: %+v", linuxRuntime)
	}
}

func TestDesktopSurfacesAreNativeOSSpecific(t *testing.T) {
	value := For("windows", "amd64")
	seen := map[string][]string{}
	for _, surface := range value.Surfaces {
		seen[surface.ID] = surface.SupportedOS
	}
	if len(seen["windows-desktop"]) != 1 || seen["windows-desktop"][0] != "windows" {
		t.Fatalf("windows desktop contract drifted: %+v", seen["windows-desktop"])
	}
	if len(seen["macos-desktop"]) != 1 || seen["macos-desktop"][0] != "darwin" {
		t.Fatalf("macOS desktop contract drifted: %+v", seen["macos-desktop"])
	}
}

func TestPlatformContractDoesNotClaimMacOSImplementedYet(t *testing.T) {
	value := For("linux", "amd64")
	states := map[string]string{}
	for _, surface := range value.Surfaces {
		states[surface.ID] = surface.State
	}
	if states["web"] != "supported" || states["windows-desktop"] != "supported" || states["linux-runtime"] != "supported" {
		t.Fatalf("current supported surface contract drifted: %+v", states)
	}
	if states["macos-desktop"] != "planned" || states["macos-runtime"] != "planned" {
		t.Fatalf("macOS must stay honest until native CI/build exists: %+v", states)
	}
}

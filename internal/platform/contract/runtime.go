package platformcontract

import "runtime"

// SurfaceKind separates a user-facing control surface from the native node that
// actually owns files, processes and game runtimes.
type SurfaceKind string

const (
	SurfaceWeb     SurfaceKind = "web-client"
	SurfaceDesktop SurfaceKind = "desktop-client"
	SurfaceRuntime SurfaceKind = "node-runtime"
)

// Surface describes where an AGMP surface runs. A Web client is always a
// control plane: it never changes the operating system of the execution node.
type Surface struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Kind            SurfaceKind `json:"kind"`
	State           string      `json:"state"`
	SupportedOS     []string    `json:"supportedOs,omitempty"`
	InstallRequired bool        `json:"installRequired"`
	Browser         bool        `json:"browser"`
	ControlsNodes   bool        `json:"controlsNodes"`
	ExecutesOnNode  bool        `json:"executesOnNode"`
}

// RuntimeContract is the platform boundary exposed to Desktop and Web. HostOS
// and HostArch always describe the machine where AGMP Core is running, not the
// browser or remote controller used to access it.
type RuntimeContract struct {
	HostOS          string    `json:"hostOs"`
	HostArch        string    `json:"hostArch"`
	NodeKind        string    `json:"nodeKind"`
	NativeExecution bool      `json:"nativeExecution"`
	RemoteControl   bool      `json:"remoteControl"`
	Surfaces        []Surface `json:"surfaces"`
}

func Current() RuntimeContract { return For(runtime.GOOS, runtime.GOARCH) }

func For(goos, goarch string) RuntimeContract {
	return RuntimeContract{
		HostOS:          goos,
		HostArch:        goarch,
		NodeKind:        nativeNodeKind(goos),
		NativeExecution: true,
		RemoteControl:   true,
		Surfaces: []Surface{
			{ID: "web", Name: "Web", Kind: SurfaceWeb, State: "supported", InstallRequired: false, Browser: true, ControlsNodes: true, ExecutesOnNode: false},
			{ID: "windows-desktop", Name: "Windows Desktop", Kind: SurfaceDesktop, State: "supported", SupportedOS: []string{"windows"}, InstallRequired: true, ControlsNodes: true, ExecutesOnNode: false},
			{ID: "macos-desktop", Name: "macOS Desktop", Kind: SurfaceDesktop, State: "planned", SupportedOS: []string{"darwin"}, InstallRequired: true, ControlsNodes: true, ExecutesOnNode: false},
			{ID: "windows-runtime", Name: "Windows Runtime", Kind: SurfaceRuntime, State: "supported", SupportedOS: []string{"windows"}, InstallRequired: true, ControlsNodes: false, ExecutesOnNode: true},
			{ID: "macos-runtime", Name: "macOS Runtime", Kind: SurfaceRuntime, State: "planned", SupportedOS: []string{"darwin"}, InstallRequired: true, ControlsNodes: false, ExecutesOnNode: true},
			{ID: "linux-runtime", Name: "Linux Runtime", Kind: SurfaceRuntime, State: "supported", SupportedOS: []string{"linux"}, InstallRequired: true, ControlsNodes: false, ExecutesOnNode: true},
		},
	}
}

func nativeNodeKind(goos string) string {
	switch goos {
	case "windows":
		return "windows-native"
	case "darwin":
		return "macos-native"
	case "linux":
		return "linux-native"
	default:
		return goos + "-native"
	}
}

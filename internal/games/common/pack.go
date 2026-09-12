package game

// PackState describes whether a Game Pack may perform real deployment today.
// Planned packs are discoverable to the UI/XiaoYu but must never be presented
// as executable capability.
type PackState string

const (
	PackSupported PackState = "supported"
	PackPlanned   PackState = "planned"
)

// Pack is the cross-client contract for a game integration. Both the visual
// game library and XiaoYu consume this shape; neither is allowed to invent a
// separate list of supported games or UI capabilities.
type Pack struct {
	ID              ID        `json:"id"`
	Family          Family    `json:"family"`
	NameZh          string    `json:"nameZh"`
	NameEn          string    `json:"nameEn"`
	State           PackState `json:"state"`
	SupportedOS     []string  `json:"supportedOs,omitempty"`
	Capabilities    []string  `json:"capabilities,omitempty"`
	UIPanels        []string  `json:"uiPanels,omitempty"`
	InstallStrategy string    `json:"installStrategy,omitempty"`
	FactSources     []string  `json:"factSources,omitempty"`
}

func (p Pack) Executable() bool { return p.State == PackSupported }

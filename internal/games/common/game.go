package game

import "slices"

// ID is a stable machine-readable game identifier, for example "steam.dst".
type ID string

// Family groups games by their delivery/runtime ecosystem rather than by display name.
type Family string

const (
	FamilySteam     Family = "steam"
	FamilyMinecraft Family = "minecraft"
)

// Capability describes an optional feature supported by a game module.
// AI Game Manager Panel does not require every game to implement every capability.
type Capability string

const (
	CapabilityDetect  Capability = "detect"
	CapabilityInstall Capability = "install"
	CapabilityUpdate  Capability = "update"
	CapabilityRuntime Capability = "runtime"
	CapabilityConfig  Capability = "config"
	CapabilityBackup  Capability = "backup"
	CapabilityConsole Capability = "console"
	CapabilityPlayers Capability = "players"
	CapabilityMods    Capability = "mods"
)

// Metadata contains generic information that AI Game Manager Panel can understand without knowing game-specific details.
type Metadata struct {
	ID           ID           `json:"id"`
	Name         string       `json:"name"`
	Family       Family       `json:"family"`
	Capabilities []Capability `json:"capabilities"`
}

func (m Metadata) Supports(capability Capability) bool {
	return slices.Contains(m.Capabilities, capability)
}

// Provider is intentionally small. Game-specific operations are exposed through optional capability interfaces later.
type Provider interface {
	Metadata() Metadata
}

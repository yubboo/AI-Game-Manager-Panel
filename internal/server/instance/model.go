package instance

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Origin string

const (
	OriginDiscovered Origin = "discovered"
	OriginVisual     Origin = "visual"
	OriginAgent      Origin = "agent"
)

// Instance is the shared resource rendered by the visual control plane and
// addressed by XiaoYu. Creation origin does not change the management model.
type Instance struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	GameID       string   `json:"gameId"`
	Origin       Origin   `json:"origin"`
	NodeOS       string   `json:"nodeOs"`
	NodeArch     string   `json:"nodeArch"`
	InstallPath  string   `json:"installPath"`
	RuntimeState string   `json:"runtimeState"`
	Capabilities []string `json:"capabilities,omitempty"`
	Managed      bool     `json:"managed"`
}

// StableID gives discovered/imported servers a repeatable identity without
// leaking the full local path into URLs or UI keys.
func StableID(gameID, path string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(gameID) + "\x00" + strings.TrimSpace(path)))
	return "inst_" + hex.EncodeToString(sum[:8])
}

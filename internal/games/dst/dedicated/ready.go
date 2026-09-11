package dedicated

import "strings"

var realStartMarkers = []string{
	"about to start a shard with these settings",
	"about to start a server with the following settings",
}

var masterReadyMarkers = []string{
	"sim paused",
	"sim unpaused",
	"dst_master_ready",
}

var secondaryReadyMarkers = []string{
	"is now ready!",
}

// AdvanceWorldReadyMarker is a direct behavior port of DSTCamp's
// advance_world_ready_marker(). Reset() returning is intentionally *not* a
// Master ready marker because the temporary worldgen Lua environment can emit
// it before the real simulation is ready.
func AdvanceWorldReadyMarker(line string, isMaster bool, realStartSeen bool) (bool, bool) {
	lowered := strings.ToLower(line)
	if !realStartSeen {
		for _, marker := range realStartMarkers {
			if strings.Contains(lowered, marker) {
				return true, false
			}
		}
		return false, false
	}
	markers := secondaryReadyMarkers
	if isMaster {
		markers = masterReadyMarkers
	}
	for _, marker := range markers {
		if strings.Contains(lowered, marker) {
			return realStartSeen, true
		}
	}
	return realStartSeen, false
}

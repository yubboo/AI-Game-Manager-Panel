package platformnet

import "strings"

type Protocol string

const (
	ProtocolUDP Protocol = "udp"
	ProtocolTCP Protocol = "tcp"
)

type Owner struct {
	Port        int      `json:"port"`
	Protocol    Protocol `json:"protocol"`
	PID         int      `json:"pid"`
	ProcessPath string   `json:"processPath"`
	ProcessName string   `json:"processName"`
}

type Status struct {
	Port     int     `json:"port"`
	Occupied bool    `json:"occupied"`
	Owners   []Owner `json:"owners"`
}

func UniquePorts(values []int) []int {
	seen := map[int]struct{}{}
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 || value > 65535 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func SameExecutable(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

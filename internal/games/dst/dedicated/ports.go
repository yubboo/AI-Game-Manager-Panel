package dedicated

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type PortPurpose string

const (
	PortPurposeServer      PortPurpose = "server"
	PortPurposeSteamMaster PortPurpose = "steam_master"
	PortPurposeSteamAuth   PortPurpose = "steam_auth"
	PortPurposeShardMaster PortPurpose = "shard_master"
)

type PortUse struct {
	Port      int         `json:"port"`
	ShardName string      `json:"shardName"`
	Purpose   PortPurpose `json:"purpose"`
	Source    string      `json:"source"`
}

type PortCollision struct {
	Port int       `json:"port"`
	Uses []PortUse `json:"uses"`
}

type PortPlan struct {
	ClusterPath string          `json:"clusterPath"`
	Uses        []PortUse       `json:"uses"`
	Collisions  []PortCollision `json:"collisions"`
	Warnings    []string        `json:"warnings"`
	Missing     []string        `json:"missing"`
}

func (p PortPlan) Ports() []int {
	seen := map[int]struct{}{}
	result := make([]int, 0, len(p.Uses))
	for _, use := range p.Uses {
		if use.Port <= 0 || use.Port > 65535 {
			continue
		}
		if _, ok := seen[use.Port]; ok {
			continue
		}
		seen[use.Port] = struct{}{}
		result = append(result, use.Port)
	}
	sort.Ints(result)
	return result
}

func ReadPortPlan(clusterPath string) (PortPlan, error) {
	clusterPath = filepath.Clean(strings.TrimSpace(clusterPath))
	if clusterPath == "." || clusterPath == "" {
		return PortPlan{}, fmt.Errorf("cluster path is required")
	}
	plan := PortPlan{ClusterPath: clusterPath, Uses: []PortUse{}, Collisions: []PortCollision{}, Warnings: []string{}, Missing: []string{}}

	clusterINI := filepath.Join(clusterPath, "cluster.ini")
	clusterValues, err := readINI(clusterINI)
	if err != nil {
		return plan, err
	}
	if enabled, ok := iniBool(clusterValues, "SHARD", "shard_enabled"); ok && enabled {
		if port, ok := iniPort(clusterValues, "SHARD", "master_port"); ok {
			plan.Uses = append(plan.Uses, PortUse{Port: port, ShardName: "Master", Purpose: PortPurposeShardMaster, Source: "cluster.ini [SHARD] master_port"})
		} else {
			plan.Missing = append(plan.Missing, "cluster.ini [SHARD] master_port")
			plan.Warnings = append(plan.Warnings, "cluster.ini 已启用 Shard，但未显式配置 [SHARD] master_port；无法在启动前验证该端口")
		}
	}

	for _, shardName := range []string{"Master", "Caves"} {
		serverINI := filepath.Join(clusterPath, shardName, "server.ini")
		if info, statErr := os.Stat(serverINI); statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		values, readErr := readINI(serverINI)
		if readErr != nil {
			return plan, readErr
		}
		appendPort := func(section, key string, purpose PortPurpose) {
			source := shardName + "/server.ini [" + section + "] " + key
			if port, ok := iniPort(values, section, key); ok {
				plan.Uses = append(plan.Uses, PortUse{Port: port, ShardName: shardName, Purpose: purpose, Source: source})
			} else {
				plan.Missing = append(plan.Missing, source)
			}
		}
		appendPort("NETWORK", "server_port", PortPurposeServer)
		appendPort("STEAM", "master_server_port", PortPurposeSteamMaster)
		appendPort("STEAM", "authentication_port", PortPurposeSteamAuth)
	}

	byPort := map[int][]PortUse{}
	for _, use := range plan.Uses {
		if use.Port <= 0 || use.Port > 65535 {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s 配置了无效端口 %d", use.Source, use.Port))
			continue
		}
		byPort[use.Port] = append(byPort[use.Port], use)
	}
	ports := make([]int, 0, len(byPort))
	for port := range byPort {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	for _, port := range ports {
		uses := byPort[port]
		if len(uses) > 1 {
			plan.Collisions = append(plan.Collisions, PortCollision{Port: port, Uses: uses})
		}
	}
	return plan, nil
}

type iniMap map[string]map[string]string

func readINI(path string) (iniMap, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	result := iniMap{}
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			end := strings.IndexByte(line, ']')
			section = strings.ToUpper(strings.TrimSpace(line[1:end]))
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if section == "" || key == "" {
			continue
		}
		if result[section] == nil {
			result[section] = map[string]string{}
		}
		result[section][key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func iniPort(values iniMap, section, key string) (int, bool) {
	sectionValues := values[strings.ToUpper(section)]
	if sectionValues == nil {
		return 0, false
	}
	raw, ok := sectionValues[strings.ToLower(key)]
	if !ok || strings.TrimSpace(raw) == "" {
		return 0, false
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return value, true
}

func iniBool(values iniMap, section, key string) (bool, bool) {
	sectionValues := values[strings.ToUpper(section)]
	if sectionValues == nil {
		return false, false
	}
	raw, ok := sectionValues[strings.ToLower(key)]
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

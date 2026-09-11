package dedicated

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

const (
	RecommendedShardMasterPort   = 10888
	RecommendedMasterServerPort  = 10999
	RecommendedMasterSteamMaster = 27016
	RecommendedMasterSteamAuth   = 8766
	RecommendedCavesServerPort   = 11000
	RecommendedCavesSteamMaster  = 27017
	RecommendedCavesSteamAuth    = 8767
)

type PortSettings struct {
	ShardMasterPort       int `json:"shardMasterPort"`
	MasterServerPort      int `json:"masterServerPort"`
	MasterSteamMasterPort int `json:"masterSteamMasterPort"`
	MasterSteamAuthPort   int `json:"masterSteamAuthPort"`
	CavesServerPort       int `json:"cavesServerPort"`
	CavesSteamMasterPort  int `json:"cavesSteamMasterPort"`
	CavesSteamAuthPort    int `json:"cavesSteamAuthPort"`
}

type PortConfiguration struct {
	ClusterPath string          `json:"clusterPath"`
	HasCaves    bool            `json:"hasCaves"`
	Configured  PortSettings    `json:"configured"`
	Effective   PortSettings    `json:"effective"`
	Recommended PortSettings    `json:"recommended"`
	Missing     []string        `json:"missing"`
	Collisions  []PortCollision `json:"collisions"`
	Valid       bool            `json:"valid"`
}

type ApplyPortSettingsResult struct {
	Configuration PortConfiguration `json:"configuration"`
	BackupPath    string            `json:"backupPath"`
}

func RecommendedPortSettings(hasCaves bool) PortSettings {
	value := PortSettings{
		ShardMasterPort:       RecommendedShardMasterPort,
		MasterServerPort:      RecommendedMasterServerPort,
		MasterSteamMasterPort: RecommendedMasterSteamMaster,
		MasterSteamAuthPort:   RecommendedMasterSteamAuth,
	}
	if hasCaves {
		value.CavesServerPort = RecommendedCavesServerPort
		value.CavesSteamMasterPort = RecommendedCavesSteamMaster
		value.CavesSteamAuthPort = RecommendedCavesSteamAuth
	}
	return value
}

func ReadPortConfiguration(clusterPath string) (PortConfiguration, error) {
	clusterPath = filepath.Clean(strings.TrimSpace(clusterPath))
	if clusterPath == "." || clusterPath == "" {
		return PortConfiguration{}, errors.New("cluster path is required")
	}
	masterINI := filepath.Join(clusterPath, "Master", "server.ini")
	if !isRegularFile(masterINI) {
		return PortConfiguration{}, errors.New("Master/server.ini 不存在")
	}
	hasCaves := isRegularFile(filepath.Join(clusterPath, "Caves", "server.ini"))
	recommended := RecommendedPortSettings(hasCaves)
	configured := PortSettings{}
	missing := []string{}

	clusterValues, err := readINI(filepath.Join(clusterPath, "cluster.ini"))
	if err != nil {
		return PortConfiguration{}, err
	}
	masterValues, err := readINI(masterINI)
	if err != nil {
		return PortConfiguration{}, err
	}
	var cavesValues iniMap
	if hasCaves {
		cavesValues, err = readINI(filepath.Join(clusterPath, "Caves", "server.ini"))
		if err != nil {
			return PortConfiguration{}, err
		}
	}

	read := func(values iniMap, section, key, label string, dst *int) {
		if value, ok := iniPort(values, section, key); ok {
			*dst = value
		} else {
			missing = append(missing, label)
		}
	}
	if hasCaves {
		read(clusterValues, "SHARD", "master_port", "cluster.ini [SHARD] master_port", &configured.ShardMasterPort)
	}
	read(masterValues, "NETWORK", "server_port", "Master/server.ini [NETWORK] server_port", &configured.MasterServerPort)
	read(masterValues, "STEAM", "master_server_port", "Master/server.ini [STEAM] master_server_port", &configured.MasterSteamMasterPort)
	read(masterValues, "STEAM", "authentication_port", "Master/server.ini [STEAM] authentication_port", &configured.MasterSteamAuthPort)
	if hasCaves {
		read(cavesValues, "NETWORK", "server_port", "Caves/server.ini [NETWORK] server_port", &configured.CavesServerPort)
		read(cavesValues, "STEAM", "master_server_port", "Caves/server.ini [STEAM] master_server_port", &configured.CavesSteamMasterPort)
		read(cavesValues, "STEAM", "authentication_port", "Caves/server.ini [STEAM] authentication_port", &configured.CavesSteamAuthPort)
	}

	effective := configured
	fillMissing(&effective, recommended, hasCaves)
	collisions := portSettingsCollisions(effective, hasCaves)
	return PortConfiguration{
		ClusterPath: clusterPath,
		HasCaves:    hasCaves,
		Configured:  configured,
		Effective:   effective,
		Recommended: recommended,
		Missing:     missing,
		Collisions:  collisions,
		Valid:       len(missing) == 0 && len(collisions) == 0,
	}, nil
}

func ApplyPortSettings(clusterPath string, settings PortSettings) (ApplyPortSettingsResult, error) {
	current, err := ReadPortConfiguration(clusterPath)
	if err != nil {
		return ApplyPortSettingsResult{}, err
	}
	if err := ValidatePortSettings(settings, current.HasCaves); err != nil {
		return ApplyPortSettingsResult{}, err
	}

	backupPath, err := backupNetworkFiles(current.ClusterPath, current.HasCaves)
	if err != nil {
		return ApplyPortSettingsResult{}, fmt.Errorf("备份当前网络配置失败: %w", err)
	}

	if current.HasCaves {
		if err := updateINIValues(filepath.Join(current.ClusterPath, "cluster.ini"), map[string]map[string]string{
			"SHARD": {"master_port": strconv.Itoa(settings.ShardMasterPort)},
		}); err != nil {
			return ApplyPortSettingsResult{}, fmt.Errorf("写入 cluster.ini 失败: %w", err)
		}
	}
	if err := updateINIValues(filepath.Join(current.ClusterPath, "Master", "server.ini"), map[string]map[string]string{
		"NETWORK": {"server_port": strconv.Itoa(settings.MasterServerPort)},
		"STEAM": {
			"master_server_port":  strconv.Itoa(settings.MasterSteamMasterPort),
			"authentication_port": strconv.Itoa(settings.MasterSteamAuthPort),
		},
	}); err != nil {
		return ApplyPortSettingsResult{}, fmt.Errorf("写入 Master/server.ini 失败: %w", err)
	}
	if current.HasCaves {
		if err := updateINIValues(filepath.Join(current.ClusterPath, "Caves", "server.ini"), map[string]map[string]string{
			"NETWORK": {"server_port": strconv.Itoa(settings.CavesServerPort)},
			"STEAM": {
				"master_server_port":  strconv.Itoa(settings.CavesSteamMasterPort),
				"authentication_port": strconv.Itoa(settings.CavesSteamAuthPort),
			},
		}); err != nil {
			return ApplyPortSettingsResult{}, fmt.Errorf("写入 Caves/server.ini 失败: %w", err)
		}
	}
	updated, err := ReadPortConfiguration(current.ClusterPath)
	if err != nil {
		return ApplyPortSettingsResult{}, err
	}
	return ApplyPortSettingsResult{Configuration: updated, BackupPath: backupPath}, nil
}

func ValidatePortSettings(settings PortSettings, hasCaves bool) error {
	values := []struct {
		label string
		value int
	}{
		{"Master 玩家端口", settings.MasterServerPort},
		{"Master Steam 主端口", settings.MasterSteamMasterPort},
		{"Master Steam 认证端口", settings.MasterSteamAuthPort},
	}
	if hasCaves {
		values = append(values,
			struct {
				label string
				value int
			}{"Shard 内部端口", settings.ShardMasterPort},
			struct {
				label string
				value int
			}{"Caves 玩家端口", settings.CavesServerPort},
			struct {
				label string
				value int
			}{"Caves Steam 主端口", settings.CavesSteamMasterPort},
			struct {
				label string
				value int
			}{"Caves Steam 认证端口", settings.CavesSteamAuthPort},
		)
	}
	for _, item := range values {
		if item.value < 1024 || item.value > 65535 {
			return fmt.Errorf("%s必须在 1024-65535 之间", item.label)
		}
	}
	collisions := portSettingsCollisions(settings, hasCaves)
	if len(collisions) > 0 {
		parts := make([]string, 0, len(collisions))
		for _, collision := range collisions {
			names := make([]string, 0, len(collision.Uses))
			for _, use := range collision.Uses {
				names = append(names, use.ShardName+"/"+string(use.Purpose))
			}
			parts = append(parts, fmt.Sprintf("%d (%s)", collision.Port, strings.Join(names, ", ")))
		}
		return fmt.Errorf("端口不能重复：%s", strings.Join(parts, "；"))
	}
	return nil
}

func fillMissing(value *PortSettings, recommended PortSettings, hasCaves bool) {
	if value.MasterServerPort == 0 {
		value.MasterServerPort = recommended.MasterServerPort
	}
	if value.MasterSteamMasterPort == 0 {
		value.MasterSteamMasterPort = recommended.MasterSteamMasterPort
	}
	if value.MasterSteamAuthPort == 0 {
		value.MasterSteamAuthPort = recommended.MasterSteamAuthPort
	}
	if hasCaves {
		if value.ShardMasterPort == 0 {
			value.ShardMasterPort = recommended.ShardMasterPort
		}
		if value.CavesServerPort == 0 {
			value.CavesServerPort = recommended.CavesServerPort
		}
		if value.CavesSteamMasterPort == 0 {
			value.CavesSteamMasterPort = recommended.CavesSteamMasterPort
		}
		if value.CavesSteamAuthPort == 0 {
			value.CavesSteamAuthPort = recommended.CavesSteamAuthPort
		}
	}
}

func portSettingsCollisions(settings PortSettings, hasCaves bool) []PortCollision {
	uses := []PortUse{
		{Port: settings.MasterServerPort, ShardName: "Master", Purpose: PortPurposeServer, Source: "Master/server.ini [NETWORK] server_port"},
		{Port: settings.MasterSteamMasterPort, ShardName: "Master", Purpose: PortPurposeSteamMaster, Source: "Master/server.ini [STEAM] master_server_port"},
		{Port: settings.MasterSteamAuthPort, ShardName: "Master", Purpose: PortPurposeSteamAuth, Source: "Master/server.ini [STEAM] authentication_port"},
	}
	if hasCaves {
		uses = append(uses,
			PortUse{Port: settings.ShardMasterPort, ShardName: "Master", Purpose: PortPurposeShardMaster, Source: "cluster.ini [SHARD] master_port"},
			PortUse{Port: settings.CavesServerPort, ShardName: "Caves", Purpose: PortPurposeServer, Source: "Caves/server.ini [NETWORK] server_port"},
			PortUse{Port: settings.CavesSteamMasterPort, ShardName: "Caves", Purpose: PortPurposeSteamMaster, Source: "Caves/server.ini [STEAM] master_server_port"},
			PortUse{Port: settings.CavesSteamAuthPort, ShardName: "Caves", Purpose: PortPurposeSteamAuth, Source: "Caves/server.ini [STEAM] authentication_port"},
		)
	}
	byPort := map[int][]PortUse{}
	for _, use := range uses {
		if use.Port > 0 {
			byPort[use.Port] = append(byPort[use.Port], use)
		}
	}
	ports := make([]int, 0, len(byPort))
	for port := range byPort {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	out := []PortCollision{}
	for _, port := range ports {
		if len(byPort[port]) > 1 {
			out = append(out, PortCollision{Port: port, Uses: byPort[port]})
		}
	}
	return out
}

func updateINIValues(path string, updates map[string]map[string]string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("配置文件不可用: %s", path)
	}
	if info.Size() > maxClusterINIBytes {
		return errors.New("配置文件异常过大")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newline := "\n"
	if bytes.Contains(body, []byte("\r\n")) {
		newline = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")

	normalized := map[string]map[string]string{}
	for section, values := range updates {
		upper := strings.ToUpper(strings.TrimSpace(section))
		normalized[upper] = map[string]string{}
		for key, value := range values {
			normalized[upper][strings.ToLower(strings.TrimSpace(key))] = value
		}
	}

	applied := map[string]map[string]bool{}
	section := ""
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToUpper(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
			continue
		}
		values := normalized[section]
		if values == nil {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value, ok := values[key]
		if !ok {
			continue
		}
		indent := raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
		lines[i] = indent + key + " = " + value
		if applied[section] == nil {
			applied[section] = map[string]bool{}
		}
		applied[section][key] = true
	}

	sections := make([]string, 0, len(normalized))
	for sec := range normalized {
		sections = append(sections, sec)
	}
	sort.Strings(sections)
	for _, sec := range sections {
		missingKeys := []string{}
		for key := range normalized[sec] {
			if !applied[sec][key] {
				missingKeys = append(missingKeys, key)
			}
		}
		if len(missingKeys) == 0 {
			continue
		}
		sort.Strings(missingKeys)
		start, end := findINISection(lines, sec)
		additions := make([]string, 0, len(missingKeys))
		for _, key := range missingKeys {
			additions = append(additions, key+" = "+normalized[sec][key])
		}
		if start >= 0 {
			lines = insertLines(lines, end, additions)
		} else {
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
				lines = append(lines, "")
			}
			lines = append(lines, "["+sec+"]")
			lines = append(lines, additions...)
		}
	}

	updated := strings.Join(lines, newline)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bonfire-ini-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(updated); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return platformfiles.AtomicReplace(tmpName, path)
}

func findINISection(lines []string, wanted string) (int, int) {
	start := -1
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
			continue
		}
		sec := strings.ToUpper(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
		if start >= 0 {
			return start, i
		}
		if sec == wanted {
			start = i
		}
	}
	if start >= 0 {
		return start, len(lines)
	}
	return -1, -1
}

func insertLines(lines []string, at int, values []string) []string {
	out := make([]string, 0, len(lines)+len(values))
	out = append(out, lines[:at]...)
	out = append(out, values...)
	out = append(out, lines[at:]...)
	return out
}

func backupNetworkFiles(clusterPath string, hasCaves bool) (string, error) {
	stamp := time.Now().Format("20060102-150405")
	backupRoot := filepath.Join(clusterPath, ".bonfire", "backups", "network", stamp)
	files := []string{"cluster.ini", filepath.Join("Master", "server.ini")}
	if hasCaves {
		files = append(files, filepath.Join("Caves", "server.ini"))
	}
	for _, rel := range files {
		source := filepath.Join(clusterPath, rel)
		body, err := os.ReadFile(source)
		if err != nil {
			return "", err
		}
		target := filepath.Join(backupRoot, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, body, 0o600); err != nil {
			return "", err
		}
	}
	return backupRoot, nil
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

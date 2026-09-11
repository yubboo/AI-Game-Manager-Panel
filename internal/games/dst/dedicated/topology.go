package dedicated

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ShardTopology struct {
	HasCaves             bool     `json:"hasCaves"`
	ShardEnabled         bool     `json:"shardEnabled"`
	MasterPort           int      `json:"masterPort"`
	BindIPConfigured     bool     `json:"bindIpConfigured"`
	MasterIPConfigured   bool     `json:"masterIpConfigured"`
	ClusterKeyConfigured bool     `json:"clusterKeyConfigured"`
	MasterIsMaster       bool     `json:"masterIsMaster"`
	CavesIsSecondary     bool     `json:"cavesIsSecondary"`
	Ready                bool     `json:"ready"`
	Issues               []string `json:"issues"`
}

func InspectShardTopology(clusterPath string) (ShardTopology, error) {
	clusterPath = filepath.Clean(strings.TrimSpace(clusterPath))
	result := ShardTopology{Issues: []string{}}
	cavesINI := filepath.Join(clusterPath, "Caves", "server.ini")
	if info, err := os.Stat(cavesINI); err != nil || !info.Mode().IsRegular() {
		result.Ready = true
		return result, nil
	}
	result.HasCaves = true

	clusterINI, err := readINI(filepath.Join(clusterPath, "cluster.ini"))
	if err != nil {
		return result, err
	}
	masterINI, err := readINI(filepath.Join(clusterPath, "Master", "server.ini"))
	if err != nil {
		return result, err
	}
	cavesValues, err := readINI(cavesINI)
	if err != nil {
		return result, err
	}

	if enabled, ok := iniBool(clusterINI, "SHARD", "shard_enabled"); ok && enabled {
		result.ShardEnabled = true
	} else {
		result.Issues = append(result.Issues, "cluster.ini [SHARD] shard_enabled 必须为 true")
	}
	if port, ok := iniPort(clusterINI, "SHARD", "master_port"); ok && port > 0 && port <= 65535 {
		result.MasterPort = port
	} else {
		result.Issues = append(result.Issues, "cluster.ini [SHARD] master_port 未配置或无效")
	}
	if value, ok := iniValue(clusterINI, "SHARD", "bind_ip"); ok && strings.TrimSpace(value) != "" {
		result.BindIPConfigured = true
	} else {
		result.Issues = append(result.Issues, "cluster.ini [SHARD] bind_ip 未配置")
	}
	if value, ok := iniValue(clusterINI, "SHARD", "master_ip"); ok && strings.TrimSpace(value) != "" {
		result.MasterIPConfigured = true
	} else {
		result.Issues = append(result.Issues, "cluster.ini [SHARD] master_ip 未配置")
	}
	if value, ok := iniValue(clusterINI, "SHARD", "cluster_key"); ok && strings.TrimSpace(value) != "" {
		result.ClusterKeyConfigured = true
	} else {
		result.Issues = append(result.Issues, "cluster.ini [SHARD] cluster_key 未配置")
	}
	if value, ok := iniBool(masterINI, "SHARD", "is_master"); ok && value {
		result.MasterIsMaster = true
	} else {
		result.Issues = append(result.Issues, "Master/server.ini [SHARD] is_master 必须为 true")
	}
	if value, ok := iniBool(cavesValues, "SHARD", "is_master"); ok && !value {
		result.CavesIsSecondary = true
	} else {
		result.Issues = append(result.Issues, "Caves/server.ini [SHARD] is_master 必须为 false")
	}
	result.Ready = len(result.Issues) == 0
	return result, nil
}

func iniValue(values iniMap, section, key string) (string, bool) {
	sectionValues := values[strings.ToUpper(section)]
	if sectionValues == nil {
		return "", false
	}
	value, ok := sectionValues[strings.ToLower(key)]
	return value, ok
}

func (t ShardTopology) Summary() string {
	if !t.HasCaves {
		return "当前 Cluster 为 Master 单 Shard"
	}
	if t.Ready {
		return fmt.Sprintf("Master/Caves Shard 联动配置完整 · master_port %d", t.MasterPort)
	}
	return strings.Join(t.Issues, "；")
}

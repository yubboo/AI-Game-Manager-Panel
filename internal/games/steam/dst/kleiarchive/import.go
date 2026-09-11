package kleiarchive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

func Apply(request ImportRequest) (ImportResult, error) {
	clusterPath := filepath.Clean(strings.TrimSpace(request.ClusterPath))
	if clusterPath == "." || clusterPath == "" {
		return ImportResult{}, errors.New("缺少目标 Cluster")
	}
	info, err := os.Stat(clusterPath)
	if err != nil || !info.IsDir() {
		return ImportResult{}, errors.New("目标 Cluster 目录不可用")
	}
	data, err := DecodeRequest(request.ArchiveName, request.ArchiveBase64)
	if err != nil {
		return ImportResult{}, err
	}
	preview, tokenBytes, configs, err := Parse(request.ArchiveName, data)
	if err != nil {
		return ImportResult{}, err
	}

	if err := token.ValidateBytes(tokenBytes); err != nil {
		return ImportResult{}, fmt.Errorf("ZIP 内服务器令牌无效: %w", err)
	}

	mode := strings.TrimSpace(request.Mode)
	if mode == "" {
		mode = ModeTokenOnly
	}
	if mode != ModeTokenOnly && mode != ModeFullConfig {
		return ImportResult{}, errors.New("未知的 Klei 配置包导入模式")
	}

	result := ImportResult{Mode: mode, ClusterPath: clusterPath, Preview: preview, Warnings: []string{}}
	if mode == ModeFullConfig {
		if !preview.HasClusterINI || !preview.HasMasterServerINI {
			return ImportResult{}, errors.New("完整配置导入需要 cluster.ini 和 Master/server.ini")
		}
		backup, copied, err := applyConfigs(clusterPath, configs)
		if err != nil {
			return ImportResult{}, err
		}
		result.BackupPath = backup
		result.ConfigFilesCopied = copied
		if !preview.HasCavesServerINI {
			result.Warnings = append(result.Warnings, "配置包未包含 Caves/server.ini；现有洞穴配置不会被删除")
		}
	}
	status, err := token.ImportBytes(clusterPath, tokenBytes)
	if err != nil {
		return ImportResult{}, err
	}
	result.TokenConfigured = status.Configured
	return result, nil
}

func applyConfigs(clusterPath string, configs map[string][]byte) (string, int, error) {
	stamp := time.Now().Format("20060102-150405")
	backupRoot := filepath.Join(clusterPath, ".bonfire-backup", "klei-import-"+stamp)
	copied := 0
	backedUp := false
	ordered := []string{"cluster.ini", filepath.Join("Master", "server.ini"), filepath.Join("Caves", "server.ini")}
	for _, rel := range ordered {
		body, ok := configs[filepath.ToSlash(rel)]
		if !ok || len(body) == 0 {
			continue
		}
		target := filepath.Join(clusterPath, rel)
		if existing, err := os.ReadFile(target); err == nil {
			backup := filepath.Join(backupRoot, rel)
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return "", copied, err
			}
			if err := os.WriteFile(backup, existing, 0o600); err != nil {
				return "", copied, err
			}
			backedUp = true
		}
		if err := writeAtomic(target, body); err != nil {
			return "", copied, fmt.Errorf("写入 %s 失败: %w", rel, err)
		}
		copied++
	}
	if !backedUp {
		backupRoot = ""
	}
	return backupRoot, copied, nil
}

func writeAtomic(target string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".bonfire-klei-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
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
	return platformfiles.AtomicReplace(tmpName, target)
}

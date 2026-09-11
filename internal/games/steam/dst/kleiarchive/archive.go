package kleiarchive

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	maxArchiveBytes      = 8 * 1024 * 1024
	maxEntries           = 64
	maxEntryBytes        = 1024 * 1024
	maxUncompressedBytes = 4 * 1024 * 1024
)

type candidate struct {
	name string
	data []byte
}

type packageFiles struct {
	preview Preview
	token   []byte
	cluster []byte
	master  []byte
	caves   []byte
}

func DecodeRequest(name, encoded string) ([]byte, error) {
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), ".zip") {
		return nil, errors.New("请选择 Klei 官方下载的 .zip 配置包")
	}
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, errors.New("配置包内容为空")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("配置包编码无效")
	}
	if len(data) == 0 {
		return nil, errors.New("配置包为空")
	}
	if len(data) > maxArchiveBytes {
		return nil, fmt.Errorf("Klei 配置包超过 %d MiB 安全上限", maxArchiveBytes/(1024*1024))
	}
	return data, nil
}

func Inspect(name string, data []byte) (Preview, error) {
	files, err := parse(name, data)
	if err != nil {
		return Preview{}, err
	}
	return files.preview, nil
}

func Parse(name string, data []byte) (Preview, []byte, map[string][]byte, error) {
	files, err := parse(name, data)
	if err != nil {
		return Preview{}, nil, nil, err
	}
	configs := map[string][]byte{}
	if len(files.cluster) > 0 {
		configs["cluster.ini"] = files.cluster
	}
	if len(files.master) > 0 {
		configs["Master/server.ini"] = files.master
	}
	if len(files.caves) > 0 {
		configs["Caves/server.ini"] = files.caves
	}
	return files.preview, append([]byte(nil), files.token...), configs, nil
}

func parse(name string, data []byte) (packageFiles, error) {
	if len(data) > maxArchiveBytes {
		return packageFiles{}, fmt.Errorf("Klei 配置包超过 %d MiB 安全上限", maxArchiveBytes/(1024*1024))
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return packageFiles{}, errors.New("无法读取 ZIP；请确认文件来自 Klei 官方“下载设置”")
	}
	if len(reader.File) == 0 || len(reader.File) > maxEntries {
		return packageFiles{}, errors.New("ZIP 文件数量异常，已拒绝导入")
	}

	entries := make(map[string]candidate)
	var total int64
	for _, file := range reader.File {
		clean, err := safeName(file.Name)
		if err != nil {
			return packageFiles{}, err
		}
		if clean == "" || file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return packageFiles{}, errors.New("ZIP 包含符号链接，已拒绝导入")
		}
		if file.UncompressedSize64 > maxEntryBytes {
			return packageFiles{}, fmt.Errorf("ZIP 内文件 %s 超过安全大小限制", clean)
		}
		total += int64(file.UncompressedSize64)
		if total > maxUncompressedBytes {
			return packageFiles{}, errors.New("ZIP 解压后体积异常，已拒绝导入")
		}
		base := strings.ToLower(path.Base(clean))
		if base != "cluster_token.txt" && base != "cluster.ini" && base != "server.ini" {
			continue
		}
		body, err := readEntry(file)
		if err != nil {
			return packageFiles{}, err
		}
		entries[strings.ToLower(clean)] = candidate{name: clean, data: body}
	}

	root, clusterEntry, err := chooseRoot(entries)
	if err != nil {
		return packageFiles{}, err
	}
	if root == "" {
		return packageFiles{}, errors.New("没有在 ZIP 中识别到 Klei Dedicated Server 配置")
	}
	lookup := func(rel string) []byte {
		key := strings.ToLower(joinRoot(root, rel))
		if item, ok := entries[key]; ok {
			return append([]byte(nil), item.data...)
		}
		return nil
	}
	files := packageFiles{}
	files.token = lookup("cluster_token.txt")
	files.cluster = lookup("cluster.ini")
	files.master = lookup("Master/server.ini")
	files.caves = lookup("Caves/server.ini")
	if len(files.cluster) == 0 && clusterEntry != nil {
		files.cluster = append([]byte(nil), clusterEntry.data...)
	}
	if len(files.token) == 0 {
		return packageFiles{}, errors.New("ZIP 中没有找到 cluster_token.txt；请确认下载的是 Klei Dedicated Server 设置包")
	}
	files.preview = Preview{
		ArchiveName: strings.TrimSpace(name), RootPrefix: normalizeRoot(root),
		HasToken: len(files.token) > 0, HasClusterINI: len(files.cluster) > 0,
		HasMasterServerINI: len(files.master) > 0, HasCavesServerINI: len(files.caves) > 0,
		EntryCount: len(reader.File), UncompressedBytes: total,
	}
	if len(files.cluster) > 0 {
		meta := parseClusterINI(files.cluster)
		files.preview.ServerName = previewText(meta["cluster_name"], 160)
		files.preview.MaxPlayers = previewText(meta["max_players"], 32)
		files.preview.GameMode = previewText(meta["game_mode"], 64)
		files.preview.Description = previewText(meta["cluster_description"], 320)
		files.preview.Passworded = strings.TrimSpace(meta["cluster_password"]) != ""
	}
	return files, nil
}

func safeName(value string) (string, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return "", errors.New("ZIP 包含绝对路径，已拒绝导入")
	}
	clean := path.Clean(value)
	if clean == "." {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", errors.New("ZIP 包含目录穿越路径，已拒绝导入")
	}
	return clean, nil
}

func readEntry(file *zip.File) ([]byte, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	limited := io.LimitReader(r, maxEntryBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > maxEntryBytes {
		return nil, errors.New("ZIP 内文件超过安全大小限制")
	}
	return body, nil
}

func chooseRoot(entries map[string]candidate) (string, *candidate, error) {
	clusterRoots := make(map[string]candidate)
	tokenRoots := make(map[string]struct{})
	for key, item := range entries {
		switch path.Base(key) {
		case "cluster.ini":
			clusterRoots[path.Dir(item.name)] = item
		case "cluster_token.txt":
			tokenRoots[path.Dir(item.name)] = struct{}{}
		}
	}
	matching := make([]string, 0)
	for root := range clusterRoots {
		if _, ok := entries[strings.ToLower(joinRoot(root, "cluster_token.txt"))]; ok {
			matching = append(matching, root)
		}
	}
	sort.Strings(matching)
	if len(matching) > 1 {
		return "", nil, errors.New("ZIP 中检测到多个独立的 Klei Cluster，无法安全判断要导入哪一个")
	}
	if len(matching) == 1 {
		item := clusterRoots[matching[0]]
		return matching[0], &item, nil
	}
	if len(clusterRoots) == 1 {
		for root, item := range clusterRoots {
			return root, &item, nil
		}
	}
	if len(clusterRoots) > 1 {
		return "", nil, errors.New("ZIP 中检测到多个 cluster.ini，已拒绝猜测导入目标")
	}
	if len(tokenRoots) == 1 {
		for root := range tokenRoots {
			return root, nil, nil
		}
	}
	if len(tokenRoots) > 1 {
		return "", nil, errors.New("ZIP 中检测到多个 cluster_token.txt，已拒绝猜测导入目标")
	}
	return "", nil, nil
}

func normalizeRoot(root string) string {
	if root == "." || root == "/" {
		return ""
	}
	return strings.Trim(root, "/")
}
func joinRoot(root, rel string) string {
	if root == "" {
		return rel
	}
	return path.Join(root, rel)
}

func parseClusterINI(data []byte) map[string]string {
	out := map[string]string{}
	section := ""
	for _, raw := range strings.Split(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if section == "gameplay" || section == "network" || section == "misc" || section == "" {
			out[key] = value
		}
	}
	return out
}

func previewText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

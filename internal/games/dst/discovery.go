package dst

import (
	"os"
	"path/filepath"
	"sort"
	"unicode"

	platformos "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/os"
)

const (
	steamKleiFolder  = "DoNotStarveTogether"
	wegameKleiFolder = "DoNotStarveTogetherRail"
)

var extraSearchDriveSubpaths = []string{
	filepath.FromSlash("系统目录/文档/Klei"),
	filepath.FromSlash("文档/Klei"),
	filepath.FromSlash("Documents/Klei"),
}

// DiscoverEnvironment is the Go parity port of dstools/shared/discovery.py.
// It scans Steam and WeGame roots independently and preserves SERVER vs LOCAL
// clusters instead of deduplicating by cluster name.
func DiscoverEnvironment() Environment {
	documents := platformos.DocumentsDir()
	env := Environment{
		DocumentsDir: documents,
		Clusters:     []Cluster{},
	}

	if root := findKleiRoot(documents, steamKleiFolder); root != "" {
		env.KleiRoot = root
		scanPlatformRoot(&env, root, DistributionSteam)
	}
	if root := findKleiRoot(documents, wegameKleiFolder); root != "" {
		env.WeGameKleiRoot = root
		scanPlatformRoot(&env, root, DistributionWeGame)
	}
	return env
}

func findKleiRoot(documentsDir, folderName string) string {
	primary := filepath.Join(documentsDir, "Klei", folderName)
	if isDir(primary) {
		return filepath.Clean(primary)
	}
	for _, drive := range []string{"D:", "E:", "F:", "G:"} {
		for _, subpath := range extraSearchDriveSubpaths {
			candidate := filepath.Join(drive+string(os.PathSeparator), subpath, folderName)
			if isDir(candidate) {
				return filepath.Clean(candidate)
			}
		}
	}
	return ""
}

func scanPlatformRoot(env *Environment, root string, distribution Distribution) {
	userDir := findUserDir(root)
	if userDir != "" {
		if distribution == DistributionSteam {
			env.UserID = filepath.Base(userDir)
			clientIni := filepath.Join(userDir, "client.ini")
			if isFile(clientIni) {
				env.ClientConfig = clientIni
			}
		} else {
			env.WeGameUserID = filepath.Base(userDir)
		}
	}

	for _, clusterPath := range listClusters(root) {
		env.Clusters = append(env.Clusters, buildCluster(clusterPath, SaveSourceServer, distribution))
	}
	if userDir != "" {
		for _, clusterPath := range listClusters(userDir) {
			env.Clusters = append(env.Clusters, buildCluster(clusterPath, SaveSourceLocal, distribution))
		}
	}
}

func findUserDir(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() && isDigits(entry.Name()) {
			return filepath.Join(root, entry.Name())
		}
	}
	return ""
}

func listClusters(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if isFile(filepath.Join(path, "cluster.ini")) && len(listShards(path)) > 0 {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func listShards(clusterPath string) []string {
	entries, err := os.ReadDir(clusterPath)
	if err != nil {
		return nil
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(clusterPath, entry.Name())
		if isFile(filepath.Join(path, "server.ini")) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func buildCluster(path string, source SaveSource, distribution Distribution) Cluster {
	cluster := Cluster{
		Name:         filepath.Base(path),
		Path:         path,
		Source:       source,
		Distribution: distribution,
		Shards:       []Shard{},
	}
	cluster.ModOverridesPath = existingFile(filepath.Join(path, "modoverrides.lua"))
	cluster.AdminListPath = existingFile(filepath.Join(path, "adminlist.txt"))
	cluster.BlockListPath = existingFile(filepath.Join(path, "blocklist.txt"))
	cluster.TokenPath = existingFile(filepath.Join(path, "cluster_token.txt"))
	for _, shardPath := range listShards(path) {
		cluster.Shards = append(cluster.Shards, buildShard(shardPath))
	}
	return cluster
}

func buildShard(path string) Shard {
	return Shard{
		Name:             filepath.Base(path),
		Path:             path,
		ModOverridesPath: existingFile(filepath.Join(path, "modoverrides.lua")),
		LevelDataPath:    existingFile(filepath.Join(path, "leveldataoverride.lua")),
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func existingFile(path string) string {
	if isFile(path) {
		return path
	}
	return ""
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

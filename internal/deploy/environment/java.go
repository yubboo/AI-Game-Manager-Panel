package environment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

type JavaInstallRequest struct {
	Major      int    `json:"major"`
	TargetRoot string `json:"targetRoot,omitempty"`
	SetDefault bool   `json:"setDefault,omitempty"`
}

type javaInfo struct {
	Major   int
	Version string
}

type adoptiumAsset struct {
	Binary struct {
		Package struct {
			Link     string `json:"link"`
			Checksum string `json:"checksum"`
			Name     string `json:"name"`
		} `json:"package"`
	} `json:"binary"`
	Version struct {
		Semver string `json:"semver"`
	} `json:"version"`
}

var javaVersionPattern = regexp.MustCompile(`(?i)version\s+"([0-9]+)(?:\.([0-9]+))?[^\"]*"`)

func (s *Service) InstallJava(ctx context.Context, request JavaInstallRequest) (RuntimeRecord, error) {
	s.installMu.Lock()
	defer s.installMu.Unlock()
	if !supportedJavaMajor(request.Major) {
		return RuntimeRecord{}, errors.New("Java 版本只支持 8、17、21、25")
	}
	osName, archName, err := adoptiumPlatform()
	if err != nil {
		return RuntimeRecord{}, err
	}
	endpoint := fmt.Sprintf("https://api.adoptium.net/v3/assets/latest/%d/hotspot?architecture=%s&image_type=jre&os=%s&vendor=eclipse", request.Major, archName, osName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RuntimeRecord{}, err
	}
	resp, err := s.options.HTTPClient.Do(req)
	if err != nil {
		return RuntimeRecord{}, fmt.Errorf("查询 Java %d 下载信息失败: %w", request.Major, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return RuntimeRecord{}, fmt.Errorf("查询 Java %d 下载信息失败: HTTP %d", request.Major, resp.StatusCode)
	}
	var assets []adoptiumAsset
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&assets); err != nil || len(assets) == 0 {
		return RuntimeRecord{}, errors.New("Adoptium 返回了无效的 Java 下载信息")
	}
	asset := assets[0]
	if asset.Binary.Package.Link == "" || asset.Binary.Package.Checksum == "" {
		return RuntimeRecord{}, errors.New("Java 下载信息缺少下载地址或 SHA256")
	}

	target := strings.TrimSpace(request.TargetRoot)
	if target == "" {
		target = filepath.Join(s.runtimeRoot(), "java", strconv.Itoa(request.Major))
	} else {
		target = s.absolutePath(target)
	}
	stage := target + ".stage-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return RuntimeRecord{}, err
	}
	defer os.RemoveAll(stage)
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	archivePath := filepath.Join(s.options.TempDir, fmt.Sprintf("java-%d-%d%s", request.Major, time.Now().UnixNano(), ext))
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return RuntimeRecord{}, err
	}
	defer os.Remove(archivePath)
	checksum, err := downloadWithSHA256(ctx, s.options.HTTPClient, asset.Binary.Package.Link, archivePath, 512<<20)
	if err != nil {
		return RuntimeRecord{}, err
	}
	if !strings.EqualFold(checksum, strings.TrimSpace(asset.Binary.Package.Checksum)) {
		return RuntimeRecord{}, errors.New("Java 下载包 SHA256 校验失败，拒绝安装")
	}
	if runtime.GOOS == "windows" {
		err = extractZIP(archivePath, stage)
	} else {
		err = extractTarGZ(archivePath, stage)
	}
	if err != nil {
		return RuntimeRecord{}, fmt.Errorf("解压 Java 失败: %w", err)
	}
	executable := findJavaExecutable(stage)
	if executable == "" {
		return RuntimeRecord{}, errors.New("Java 下载完成，但没有找到 java 可执行文件")
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(executable, 0o755)
	}
	info, err := s.inspectJava(executable)
	if err != nil {
		return RuntimeRecord{}, fmt.Errorf("Java 安装后验证失败: %w", err)
	}
	if info.Major != request.Major {
		return RuntimeRecord{}, fmt.Errorf("Java 下载包实际主版本为 %d，期望 %d", info.Major, request.Major)
	}
	actualRoot := javaRootFromExecutable(executable)
	if _, err := filepath.Rel(stage, actualRoot); err != nil {
		return RuntimeRecord{}, err
	}
	if err := replaceManagedTree(stage, target, RuntimeRecord{Kind: RuntimeJava, Major: request.Major}); err != nil {
		return RuntimeRecord{}, err
	}
	finalExecutable := filepath.Join(target, mustRel(stage, executable))
	record := RuntimeRecord{ID: fmt.Sprintf("java-%d-managed", request.Major), Kind: RuntimeJava, Name: fmt.Sprintf("Java %d", request.Major), Version: firstNonEmpty(asset.Version.Semver, info.Version), Major: request.Major, OS: runtime.GOOS, Arch: runtime.GOARCH, Root: target, Executable: finalExecutable, Source: "adoptium", Managed: true, Checksum: checksum, InstalledAt: time.Now().Unix(), LastVerified: time.Now().Unix()}
	return s.upsertRuntime(record, request.SetDefault)
}

func (s *Service) inspectJava(executable string) (javaInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := platformruntime.Run(ctx, platformruntime.RunSpec{Spec: platformruntime.Spec{Executable: executable, Arguments: []string{"-version"}}, MaxOutputBytes: 64 << 10})
	text := strings.TrimSpace(result.Stderr + "\n" + result.Stdout)
	if err != nil && text == "" {
		return javaInfo{}, fmt.Errorf("执行 java -version 失败: %w", err)
	}
	return parseJavaVersionOutput(text)
}

func parseJavaVersionOutput(text string) (javaInfo, error) {
	match := javaVersionPattern.FindStringSubmatch(strings.TrimSpace(text))
	if len(match) < 2 {
		return javaInfo{}, errors.New("无法识别 java -version 输出")
	}
	first, _ := strconv.Atoi(match[1])
	major := first
	if first == 1 && len(match) > 2 {
		major, _ = strconv.Atoi(match[2])
	}
	if major <= 0 {
		return javaInfo{}, errors.New("无法识别 Java 主版本")
	}
	return javaInfo{Major: major, Version: strings.Trim(strings.TrimSpace(match[0]), `"`)}, nil
}

func findJavaExecutable(root string) string {
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.exe"
	}
	direct := filepath.Join(root, "bin", name)
	if regularFileExists(direct) {
		return direct
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(entry.Name(), name) && strings.EqualFold(filepath.Base(filepath.Dir(path)), "bin") {
			found = path
		}
		return nil
	})
	return found
}
func javaRootFromExecutable(executable string) string {
	return filepath.Dir(filepath.Dir(filepath.Clean(executable)))
}
func supportedJavaMajor(v int) bool { return v == 8 || v == 17 || v == 21 || v == 25 }
func adoptiumPlatform() (string, string, error) {
	osName := runtime.GOOS
	if osName != "windows" && osName != "linux" && osName != "darwin" {
		return "", "", fmt.Errorf("当前平台暂不支持自动安装 Java: %s", osName)
	}
	arch := map[string]string{"amd64": "x64", "arm64": "aarch64"}[runtime.GOARCH]
	if arch == "" {
		return "", "", fmt.Errorf("当前架构暂不支持自动安装 Java: %s", runtime.GOARCH)
	}
	if osName == "darwin" {
		osName = "mac"
	}
	return osName, arch, nil
}

func downloadWithSHA256(ctx context.Context, client *http.Client, url, target string, max int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > max && resp.ContentLength >= 0 {
		return "", errors.New("下载文件超过大小限制")
	}
	f, err := os.Create(target)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, max+1))
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n > max {
		return "", errors.New("下载文件超过大小限制")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func replaceManagedTree(stage, target string, marker RuntimeRecord) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	backup := target + ".old-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	hadOld := false
	if _, err := os.Stat(target); err == nil {
		if err := os.Rename(target, backup); err != nil {
			return err
		}
		hadOld = true
	}
	if err := os.Rename(stage, target); err != nil {
		if hadOld {
			_ = os.Rename(backup, target)
		}
		return err
	}
	markerBytes, _ := json.MarshalIndent(map[string]any{"managed": true, "kind": marker.Kind, "major": marker.Major, "createdAt": time.Now().Unix()}, "", "  ")
	if err := os.WriteFile(filepath.Join(target, ".agmp-runtime.json"), append(markerBytes, '\n'), 0o600); err != nil {
		_ = os.RemoveAll(target)
		if hadOld {
			_ = os.Rename(backup, target)
		}
		return err
	}
	if hadOld {
		_ = os.RemoveAll(backup)
	}
	return nil
}
func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

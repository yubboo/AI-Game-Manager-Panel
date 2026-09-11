package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

const maxInstallerBytes int64 = 512 << 20

type Config struct {
	Enabled                     bool   `json:"enabled"`
	Provider                    string `json:"provider"`
	Repository                  string `json:"repository"`
	APIBaseURL                  string `json:"apiBaseUrl"`
	Channel                     string `json:"channel"`
	CheckOnStartup              bool   `json:"checkOnStartup"`
	MinimumCheckIntervalMinutes int    `json:"minimumCheckIntervalMinutes"`
	AssetPattern                string `json:"assetPattern"`
	RequireSHA256               bool   `json:"requireSha256"`
	InstallMode                 string `json:"installMode"`
	PreserveRuntimeData         bool   `json:"preserveRuntimeData"`
}

type Options struct {
	CurrentVersion string
	CacheDir       string
	Config         Config
	HTTPClient     *http.Client
	Now            func() time.Time
}

type Status struct {
	Enabled         bool   `json:"enabled"`
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Channel         string `json:"channel"`
	ReleaseName     string `json:"releaseName"`
	ReleaseNotes    string `json:"releaseNotes"`
	ReleaseURL      string `json:"releaseUrl"`
	PublishedAt     string `json:"publishedAt"`
	InstallerName   string `json:"installerName"`
	InstallerSize   int64  `json:"installerSize"`
	InstallerSHA256 string `json:"installerSha256"`
	CanInstall      bool   `json:"canInstall"`
	CheckedAt       int64  `json:"checkedAt"`
	Message         string `json:"message"`
}

type PreparedUpdate struct {
	Version       string `json:"version"`
	InstallerPath string `json:"installerPath"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

type Service struct {
	currentVersion string
	cacheDir       string
	config         Config
	client         *http.Client
	now            func() time.Time

	mu          sync.Mutex
	lastChecked time.Time
	lastStatus  Status
}

func New(options Options) *Service {
	cfg := options.Config
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		cfg.APIBaseURL = "https://api.github.com"
	}
	if strings.TrimSpace(cfg.Channel) == "" {
		cfg.Channel = "stable"
	}
	if cfg.MinimumCheckIntervalMinutes <= 0 {
		cfg.MinimumCheckIntervalMinutes = 360
	}
	if strings.TrimSpace(cfg.AssetPattern) == "" {
		cfg.AssetPattern = "AI-Game-Manager-Panel-{version}-Windows-x64-Setup.exe"
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		currentVersion: strings.TrimSpace(options.CurrentVersion),
		cacheDir:       filepath.Clean(options.CacheDir),
		config:         cfg,
		client:         client,
		now:            now,
	}
}

func (s *Service) Check(ctx context.Context, force bool) (Status, error) {
	s.mu.Lock()
	if !force && !s.lastChecked.IsZero() && s.now().Sub(s.lastChecked) < time.Duration(s.config.MinimumCheckIntervalMinutes)*time.Minute {
		status := s.lastStatus
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()

	status, err := s.checkGitHub(ctx)
	if err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	s.lastChecked = s.now()
	s.lastStatus = status
	s.mu.Unlock()
	return status, nil
}

func (s *Service) checkGitHub(ctx context.Context) (Status, error) {
	base := Status{
		Enabled:        s.config.Enabled,
		CurrentVersion: s.currentVersion,
		Channel:        s.config.Channel,
		CheckedAt:      s.now().Unix(),
		Message:        "当前已是最新版本。",
	}
	if !s.config.Enabled {
		base.Message = "自动更新已关闭。"
		return base, nil
	}
	if strings.TrimSpace(s.config.Provider) != "github-releases" {
		return Status{}, fmt.Errorf("不支持的更新提供方：%s", s.config.Provider)
	}
	owner, repo, err := splitRepository(s.config.Repository)
	if err != nil {
		return Status{}, err
	}
	endpoint := strings.TrimRight(s.config.APIBaseURL, "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Status{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AI-Game-Manager-Panel/"+s.currentVersion)
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	resp, err := s.client.Do(req)
	if err != nil {
		return Status{}, fmt.Errorf("检查新版本失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		base.Message = "尚未发布可用的 GitHub Release。"
		return base, nil
	}
	if resp.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("检查新版本失败：GitHub HTTP %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return Status{}, fmt.Errorf("解析 GitHub Release 失败：%w", err)
	}
	if release.Draft || release.Prerelease {
		base.Message = "最新 Release 不是稳定版本。"
		return base, nil
	}
	latest, err := normalizeVersion(release.TagName)
	if err != nil {
		return Status{}, fmt.Errorf("无法识别 Release 版本 %q：%w", release.TagName, err)
	}
	base.LatestVersion = latest
	base.ReleaseName = strings.TrimSpace(release.Name)
	if base.ReleaseName == "" {
		base.ReleaseName = "AI Game Manager Panel " + latest
	}
	base.ReleaseNotes = strings.TrimSpace(release.Body)
	base.ReleaseURL = release.HTMLURL
	if !release.PublishedAt.IsZero() {
		base.PublishedAt = release.PublishedAt.Format(time.RFC3339)
	}

	cmp, err := compareVersions(latest, s.currentVersion)
	if err != nil {
		return Status{}, err
	}
	if cmp <= 0 {
		return base, nil
	}
	base.UpdateAvailable = true
	base.Message = "发现新版本 v" + latest

	expectedName := strings.ReplaceAll(s.config.AssetPattern, "{version}", latest)
	installer, ok := findAsset(release.Assets, expectedName)
	if !ok {
		base.Message = "发现新版本，但 Release 中缺少 Windows 安装器。"
		return base, nil
	}
	if err := validateGitHubAsset(installer); err != nil {
		return Status{}, err
	}
	base.InstallerName = installer.Name
	base.InstallerSize = installer.Size
	base.InstallerSHA256 = digestSHA256(installer.Digest)
	base.CanInstall = runtime.GOOS == "windows"
	if s.config.RequireSHA256 && base.InstallerSHA256 == "" {
		if checksum, ok := findAsset(release.Assets, installer.Name+".sha256"); ok {
			hash, err := s.fetchChecksum(ctx, checksum)
			if err != nil {
				return Status{}, err
			}
			base.InstallerSHA256 = hash
		} else {
			base.CanInstall = false
			base.Message = "发现新版本，但缺少 SHA256 校验信息，已阻止自动安装。"
		}
	}
	return base, nil
}

func (s *Service) PrepareLatest(ctx context.Context) (PreparedUpdate, error) {
	if runtime.GOOS != "windows" {
		return PreparedUpdate{}, errors.New("自动安装目前仅支持 Windows 桌面版")
	}
	status, err := s.Check(ctx, true)
	if err != nil {
		return PreparedUpdate{}, err
	}
	if !status.UpdateAvailable {
		return PreparedUpdate{}, errors.New("当前没有可安装的新版本")
	}
	if !status.CanInstall || status.InstallerName == "" {
		return PreparedUpdate{}, errors.New(status.Message)
	}
	release, installer, err := s.resolveLatestInstaller(ctx, status.LatestVersion, status.InstallerName)
	if err != nil {
		return PreparedUpdate{}, err
	}
	expectedHash := status.InstallerSHA256
	if expectedHash == "" && s.config.RequireSHA256 {
		return PreparedUpdate{}, errors.New("缺少安装器 SHA256，拒绝自动更新")
	}
	dir := filepath.Join(s.cacheDir, "updates", status.LatestVersion)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return PreparedUpdate{}, fmt.Errorf("创建更新缓存目录失败：%w", err)
	}
	final := filepath.Join(dir, filepath.Base(installer.Name))
	if info, err := os.Stat(final); err == nil && info.Size() > 0 {
		hash, err := fileSHA256(final)
		if err == nil && (expectedHash == "" || strings.EqualFold(hash, expectedHash)) {
			return PreparedUpdate{Version: status.LatestVersion, InstallerPath: final, SHA256: hash, Size: info.Size()}, nil
		}
		_ = os.Remove(final)
	}
	if installer.Size <= 0 || installer.Size > maxInstallerBytes {
		return PreparedUpdate{}, fmt.Errorf("安装器大小异常：%d bytes", installer.Size)
	}
	if err := downloadFile(ctx, s.client, installer.BrowserDownloadURL, final, installer.Size, expectedHash); err != nil {
		return PreparedUpdate{}, err
	}
	hash, err := fileSHA256(final)
	if err != nil {
		return PreparedUpdate{}, err
	}
	_ = release
	info, _ := os.Stat(final)
	return PreparedUpdate{Version: status.LatestVersion, InstallerPath: final, SHA256: hash, Size: info.Size()}, nil
}

func (s *Service) LaunchInstaller(update PreparedUpdate) error {
	if runtime.GOOS != "windows" {
		return errors.New("自动安装目前仅支持 Windows")
	}
	if strings.TrimSpace(update.InstallerPath) == "" {
		return errors.New("安装器路径为空")
	}
	path, err := filepath.Abs(update.InstallerPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("更新安装器不存在：%w", err)
	}
	_, err = platformruntime.StartDetached(platformruntime.Spec{
		Executable:       path,
		Arguments:        []string{"/SP-", "/AGMPUPDATE=1"},
		WorkingDirectory: filepath.Dir(path),
		ShowWindow:       true,
	})
	if err != nil {
		return fmt.Errorf("启动更新安装器失败：%w", err)
	}
	return nil
}

func (s *Service) resolveLatestInstaller(ctx context.Context, version, name string) (githubRelease, githubAsset, error) {
	owner, repo, err := splitRepository(s.config.Repository)
	if err != nil {
		return githubRelease{}, githubAsset{}, err
	}
	endpoint := strings.TrimRight(s.config.APIBaseURL, "/") + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases/latest"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AI-Game-Manager-Panel/"+s.currentVersion)
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	resp, err := s.client.Do(req)
	if err != nil {
		return githubRelease{}, githubAsset{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, githubAsset{}, fmt.Errorf("GitHub HTTP %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return githubRelease{}, githubAsset{}, err
	}
	normalized, err := normalizeVersion(release.TagName)
	if err != nil || normalized != version {
		return githubRelease{}, githubAsset{}, errors.New("Release 在下载前发生变化，请重新检查更新")
	}
	asset, ok := findAsset(release.Assets, name)
	if !ok {
		return githubRelease{}, githubAsset{}, errors.New("Release 安装器已不存在")
	}
	if err := validateGitHubAsset(asset); err != nil {
		return githubRelease{}, githubAsset{}, err
	}
	return release, asset, nil
}

func (s *Service) fetchChecksum(ctx context.Context, asset githubAsset) (string, error) {
	if err := validateGitHubAsset(asset); err != nil {
		return "", err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	req.Header.Set("User-Agent", "AI-Game-Manager-Panel/"+s.currentVersion)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载 SHA256 失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 SHA256 失败：HTTP %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 32<<10))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		hash := strings.TrimPrefix(strings.ToLower(fields[0]), "sha256:")
		if isSHA256(hash) {
			return hash, nil
		}
	}
	return "", errors.New("SHA256 文件内容无效")
}

func splitRepository(value string) (string, string, error) {
	parts := strings.Split(strings.Trim(value, "/ "), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("update repository 必须为 owner/repo")
	}
	return parts[0], parts[1], nil
}

func normalizeVersion(value string) (string, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if strings.HasPrefix(value, "AI-Game-Manager-Panel-") {
		value = strings.TrimPrefix(value, "AI-Game-Manager-Panel-")
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return "", errors.New("版本必须为 x.y.z")
	}
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return "", errors.New("版本包含非数字段")
		}
	}
	return value, nil
}

func compareVersions(a, b string) (int, error) {
	av, err := versionNumbers(a)
	if err != nil {
		return 0, err
	}
	bv, err := versionNumbers(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return 1, nil
		}
		if av[i] < bv[i] {
			return -1, nil
		}
	}
	return 0, nil
}

func versionNumbers(v string) ([3]int, error) {
	var out [3]int
	normalized, err := normalizeVersion(v)
	if err != nil {
		return out, err
	}
	parts := strings.Split(normalized, ".")
	for i := range out {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out, nil
}

func findAsset(assets []githubAsset, name string) (githubAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func validateGitHubAsset(asset githubAsset) error {
	u, err := url.Parse(asset.BrowserDownloadURL)
	if err != nil || u.Scheme != "https" {
		return errors.New("Release asset 下载地址不是有效 HTTPS")
	}
	if !strings.EqualFold(u.Hostname(), "github.com") {
		return fmt.Errorf("Release asset 下载地址不是 github.com：%s", u.Hostname())
	}
	return nil
}

func digestSHA256(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "sha256:")
	if isSHA256(value) {
		return value
	}
	return ""
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadFile(ctx context.Context, client *http.Client, sourceURL, destination string, expectedSize int64, expectedHash string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	req.Header.Set("User-Agent", "AI-Game-Manager-Panel-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载安装器失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载安装器失败：HTTP %d", resp.StatusCode)
	}
	temp := destination + ".part"
	_ = os.Remove(temp)
	f, err := os.OpenFile(temp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	h := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxInstallerBytes+1))
	syncErr := f.Sync()
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(temp)
		return copyErr
	}
	if syncErr != nil {
		_ = os.Remove(temp)
		return syncErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return closeErr
	}
	if written > maxInstallerBytes {
		_ = os.Remove(temp)
		return errors.New("安装器超过安全大小上限")
	}
	if expectedSize > 0 && written != expectedSize {
		_ = os.Remove(temp)
		return fmt.Errorf("安装器大小校验失败：期望 %d，实际 %d", expectedSize, written)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))
	if expectedHash != "" && !strings.EqualFold(actualHash, expectedHash) {
		_ = os.Remove(temp)
		return errors.New("安装器 SHA256 校验失败，已拒绝运行")
	}
	if err := os.Rename(temp, destination); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

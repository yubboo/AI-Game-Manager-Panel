package environment

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

type RuntimeKind string

const (
	RuntimeJava     RuntimeKind = "java"
	RuntimeSteamCMD RuntimeKind = "steamcmd"
)

type RuntimeRecord struct {
	ID           string            `json:"id"`
	Kind         RuntimeKind       `json:"kind"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Major        int               `json:"major,omitempty"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	Root         string            `json:"root"`
	Executable   string            `json:"executable"`
	Source       string            `json:"source"`
	Managed      bool              `json:"managed"`
	Default      bool              `json:"default"`
	Checksum     string            `json:"checksum,omitempty"`
	InstalledAt  int64             `json:"installedAt,omitempty"`
	LastVerified int64             `json:"lastVerified,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type RuntimeCatalog struct {
	Platform     string          `json:"platform"`
	Architecture string          `json:"architecture"`
	RuntimeRoot  string          `json:"runtimeRoot"`
	JavaMajors   []int           `json:"javaMajors"`
	Runtimes     []RuntimeRecord `json:"runtimes"`
	Warnings     []string        `json:"warnings"`
}

type RegisterRuntimeRequest struct {
	Kind       RuntimeKind `json:"kind"`
	Executable string      `json:"executable"`
	Major      int         `json:"major,omitempty"`
	SetDefault bool        `json:"setDefault,omitempty"`
}

type ResolveRuntimeRequest struct {
	Kind  RuntimeKind `json:"kind"`
	Major int         `json:"major,omitempty"`
}

type SetDefaultRuntimeRequest struct {
	ID string `json:"id"`
}

type RemoveRuntimeRequest struct {
	ID string `json:"id"`
}

type runtimeRegistryFile struct {
	Version  int             `json:"version"`
	Runtimes []RuntimeRecord `json:"runtimes"`
}

func (s *Service) RuntimeCatalog() RuntimeCatalog {
	s.registryMu.RLock()
	values := append([]RuntimeRecord{}, s.runtimes...)
	s.registryMu.RUnlock()
	warnings := make([]string, 0)
	for i := range values {
		if !regularFileExists(values[i].Executable) {
			warnings = append(warnings, fmt.Sprintf("Runtime %s 的可执行文件已失效：%s", values[i].ID, values[i].Executable))
		}
	}
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Kind != values[j].Kind {
			return values[i].Kind < values[j].Kind
		}
		if values[i].Major != values[j].Major {
			return values[i].Major < values[j].Major
		}
		return values[i].ID < values[j].ID
	})
	return RuntimeCatalog{Platform: runtime.GOOS, Architecture: runtime.GOARCH, RuntimeRoot: s.runtimeRoot(), JavaMajors: []int{8, 17, 21, 25}, Runtimes: values, Warnings: warnings}
}

func (s *Service) ResolveRuntime(request ResolveRuntimeRequest) (RuntimeRecord, error) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()
	var fallback *RuntimeRecord
	for i := range s.runtimes {
		r := s.runtimes[i]
		if r.Kind != request.Kind || !regularFileExists(r.Executable) {
			continue
		}
		if request.Major > 0 && r.Major != request.Major {
			continue
		}
		if r.Default {
			return r, nil
		}
		if fallback == nil {
			copy := r
			fallback = &copy
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return RuntimeRecord{}, fmt.Errorf("未找到可用 Runtime：kind=%s major=%d", request.Kind, request.Major)
}

func (s *Service) RegisterRuntime(request RegisterRuntimeRequest) (RuntimeRecord, error) {
	executable := s.absolutePath(request.Executable)
	if !regularFileExists(executable) {
		return RuntimeRecord{}, errors.New("Runtime 可执行文件不存在")
	}
	switch request.Kind {
	case RuntimeJava:
		info, err := s.inspectJava(executable)
		if err != nil {
			return RuntimeRecord{}, err
		}
		if request.Major > 0 && request.Major != info.Major {
			return RuntimeRecord{}, fmt.Errorf("Java 实际主版本为 %d，与指定版本 %d 不匹配", info.Major, request.Major)
		}
		record := RuntimeRecord{ID: fmt.Sprintf("java-%d-external-%x", info.Major, stablePathHash(executable)), Kind: RuntimeJava, Name: fmt.Sprintf("Java %d", info.Major), Version: info.Version, Major: info.Major, OS: runtime.GOOS, Arch: runtime.GOARCH, Root: javaRootFromExecutable(executable), Executable: executable, Source: "external", Managed: false, LastVerified: time.Now().Unix()}
		return s.upsertRuntime(record, request.SetDefault)
	case RuntimeSteamCMD:
		record := RuntimeRecord{ID: fmt.Sprintf("steamcmd-external-%x", stablePathHash(executable)), Kind: RuntimeSteamCMD, Name: "SteamCMD", Version: "external", OS: runtime.GOOS, Arch: runtime.GOARCH, Root: filepath.Dir(executable), Executable: executable, Source: "external", Managed: false, LastVerified: time.Now().Unix()}
		return s.upsertRuntime(record, request.SetDefault)
	default:
		return RuntimeRecord{}, errors.New("不支持的 Runtime 类型")
	}
}

func (s *Service) SetDefaultRuntime(request SetDefaultRuntimeRequest) (RuntimeRecord, error) {
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	idx := -1
	for i := range s.runtimes {
		if s.runtimes[i].ID == strings.TrimSpace(request.ID) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return RuntimeRecord{}, errors.New("Runtime 不存在")
	}
	if !regularFileExists(s.runtimes[idx].Executable) {
		return RuntimeRecord{}, errors.New("Runtime 可执行文件已失效")
	}
	kind := s.runtimes[idx].Kind
	for i := range s.runtimes {
		if s.runtimes[i].Kind == kind {
			s.runtimes[i].Default = false
		}
	}
	s.runtimes[idx].Default = true
	if err := s.saveRegistryLocked(); err != nil {
		return RuntimeRecord{}, err
	}
	return s.runtimes[idx], nil
}

func (s *Service) RemoveRuntime(request RemoveRuntimeRequest) error {
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	idx := -1
	for i := range s.runtimes {
		if s.runtimes[i].ID == strings.TrimSpace(request.ID) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("Runtime 不存在")
	}
	target := s.runtimes[idx]
	if target.Managed {
		root, err := filepath.Abs(target.Root)
		if err != nil {
			return err
		}
		managedRoot, err := filepath.Abs(s.runtimeRoot())
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(managedRoot, root)
		insideOwnedRoot := relErr == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
		if insideOwnedRoot {
			if _, err := os.Stat(filepath.Join(root, ".agmp-runtime.json")); err != nil {
				return errors.New("缺少 AGMP Runtime 管理标记，拒绝删除目录")
			}
			if err := os.RemoveAll(root); err != nil {
				return err
			}
		}
		// Custom managed paths outside AGMP's RuntimeRoot are deliberately only
		// unregistered. This avoids turning a user-supplied path into an arbitrary
		// recursive-delete primitive.
	}
	s.runtimes = append(s.runtimes[:idx], s.runtimes[idx+1:]...)
	return s.saveRegistryLocked()
}

func (s *Service) upsertRuntime(record RuntimeRecord, setDefault bool) (RuntimeRecord, error) {
	s.registryMu.Lock()
	defer s.registryMu.Unlock()
	idx := -1
	for i := range s.runtimes {
		if s.runtimes[i].ID == record.ID || samePath(s.runtimes[i].Executable, record.Executable) {
			idx = i
			break
		}
	}
	if idx >= 0 && !setDefault && s.runtimes[idx].Default {
		record.Default = true
	}
	if setDefault {
		for i := range s.runtimes {
			if s.runtimes[i].Kind == record.Kind {
				s.runtimes[i].Default = false
			}
		}
		record.Default = true
	}
	if idx >= 0 {
		s.runtimes[idx] = record
	} else {
		s.runtimes = append(s.runtimes, record)
	}
	if err := s.saveRegistryLocked(); err != nil {
		return RuntimeRecord{}, err
	}
	return record, nil
}

func (s *Service) runtimeRoot() string {
	return filepath.Join(filepath.Clean(s.options.Root), "runtime", "environments")
}
func (s *Service) registryPath() string {
	return filepath.Join(s.options.DataDir, "environment", "runtimes.json")
}

func (s *Service) loadRegistry() error {
	data, err := os.ReadFile(s.registryPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var file runtimeRegistryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	s.runtimes = file.Runtimes
	return nil
}
func (s *Service) saveRegistryLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.registryPath()), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(runtimeRegistryFile{Version: 1, Runtimes: s.runtimes}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.registryPath() + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(tmp, s.registryPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
func stablePathHash(path string) [6]byte {
	sum := sha256Bytes([]byte(strings.ToLower(filepath.Clean(path))))
	var out [6]byte
	copy(out[:], sum[:6])
	return out
}
func sha256Bytes(data []byte) [32]byte { return sha256.Sum256(data) }

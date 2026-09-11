package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	ErrPathRequired     = errors.New("path is required")
	ErrOutsideWorkspace = errors.New("path is outside AGMP workspace")
	ErrNotFile          = errors.New("target is not a file")
	ErrWorkspaceRoot    = errors.New("workspace root cannot be mutated by this operation")
)

const (
	DefaultReadLimit  = 2 * 1024 * 1024
	DefaultWriteLimit = 4 * 1024 * 1024
)

type Entry struct {
	Name      string `json:"name"`
	Directory bool   `json:"directory"`
	Size      int64  `json:"size"`
}

type ListResult struct {
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
}

type ReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

type StatResult struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	Directory bool   `json:"directory"`
	Size      int64  `json:"size"`
}

type WriteResult struct {
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	Created bool   `json:"created"`
}

type ReplaceResult struct {
	Path         string `json:"path"`
	Replacements int    `json:"replacements"`
	Size         int    `json:"size"`
}

type MkdirResult struct {
	Path    string `json:"path"`
	Created bool   `json:"created"`
}

type RemoveResult struct {
	Path      string `json:"path"`
	Directory bool   `json:"directory"`
	Recursive bool   `json:"recursive"`
}

// Service owns workspace-scoped file capabilities exposed to XiaoYu. The
// service, not the model, is the filesystem security boundary: every path is
// resolved against one canonical root and symlink escapes are rejected. This
// lets XiaoYu have general file "hands" without granting unrestricted host FS.
type Service struct {
	root string
}

func New(root string) (*Service, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrPathRequired
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root symlinks: %w", err)
	}
	return &Service{root: filepath.Clean(resolved)}, nil
}

func (s *Service) Root() string { return s.root }

// Resolve returns an existing path inside the AGMP workspace. Empty input maps
// to the workspace root and symlink escapes are rejected.
func (s *Service) Resolve(path string) (string, error) { return s.resolveExisting(path, true) }

func (s *Service) List(path string) (ListResult, error) {
	resolved, err := s.resolveExisting(path, true)
	if err != nil {
		return ListResult{}, err
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return ListResult{}, fmt.Errorf("read directory: %w", err)
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return ListResult{}, fmt.Errorf("read entry %s: %w", entry.Name(), err)
		}
		result = append(result, Entry{
			Name:      entry.Name(),
			Directory: entry.IsDir(),
			Size: func() int64 {
				if info.Mode().IsRegular() {
					return info.Size()
				}
				return 0
			}(),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return ListResult{Path: resolved, Entries: result}, nil
}

func (s *Service) Read(path string, maxBytes int64) (ReadResult, error) {
	if strings.TrimSpace(path) == "" {
		return ReadResult{}, ErrPathRequired
	}
	resolved, err := s.resolveExisting(path, false)
	if err != nil {
		return ReadResult{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return ReadResult{}, err
	}
	if !info.Mode().IsRegular() {
		return ReadResult{}, ErrNotFile
	}
	if maxBytes <= 0 {
		maxBytes = DefaultReadLimit
	}
	if info.Size() > maxBytes {
		return ReadResult{}, fmt.Errorf("file exceeds read limit: %d > %d", info.Size(), maxBytes)
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return ReadResult{}, fmt.Errorf("read file: %w", err)
	}
	if !validUTF8(content) {
		return ReadResult{}, errors.New("file is not valid UTF-8 text")
	}
	return ReadResult{Path: resolved, Content: string(content), Size: info.Size()}, nil
}

func (s *Service) Stat(path string) (StatResult, error) {
	if strings.TrimSpace(path) == "" {
		return StatResult{}, ErrPathRequired
	}
	candidate, err := s.resolveTarget(path)
	if err != nil {
		return StatResult{}, err
	}
	info, err := os.Stat(candidate)
	if os.IsNotExist(err) {
		return StatResult{Path: candidate, Exists: false}, nil
	}
	if err != nil {
		return StatResult{}, err
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return StatResult{}, err
	}
	if err := s.ensureInside(resolved); err != nil {
		return StatResult{}, err
	}
	return StatResult{Path: filepath.Clean(resolved), Exists: true, Directory: info.IsDir(), Size: regularSize(info)}, nil
}

// Write stores UTF-8 text inside the managed workspace. createParents may be
// used by an agent for a missing directory tree, but the nearest existing
// ancestor is resolved first so a symlink cannot redirect creation outside the
// workspace.
func (s *Service) Write(path, content string, createParents bool) (WriteResult, error) {
	if strings.TrimSpace(path) == "" {
		return WriteResult{}, ErrPathRequired
	}
	data := []byte(content)
	if !validUTF8(data) {
		return WriteResult{}, errors.New("content is not valid UTF-8 text")
	}
	if len(data) > DefaultWriteLimit {
		return WriteResult{}, fmt.Errorf("content exceeds write limit: %d > %d", len(data), DefaultWriteLimit)
	}
	resolved, err := s.resolveTarget(path)
	if err != nil {
		return WriteResult{}, err
	}
	if filepath.Clean(resolved) == s.root {
		return WriteResult{}, ErrWorkspaceRoot
	}
	existed := false
	perm := os.FileMode(0o600)
	if info, statErr := os.Stat(resolved); statErr == nil {
		existed = true
		if info.IsDir() {
			return WriteResult{}, ErrNotFile
		}
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(statErr) {
		return WriteResult{}, statErr
	}
	parent := filepath.Dir(resolved)
	if createParents {
		if _, err := s.resolveTarget(parent); err != nil {
			return WriteResult{}, err
		}
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return WriteResult{}, fmt.Errorf("create parent directories: %w", err)
		}
	} else if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		if err != nil {
			return WriteResult{}, fmt.Errorf("parent directory unavailable: %w", err)
		}
		return WriteResult{}, errors.New("parent path is not a directory")
	}
	if err := os.WriteFile(resolved, data, perm); err != nil {
		return WriteResult{}, fmt.Errorf("write file: %w", err)
	}
	return WriteResult{Path: resolved, Bytes: len(data), Created: !existed}, nil
}

// Replace performs an exact text replacement. It is the preferred mutation for
// small config edits because the old text acts as a lightweight precondition;
// a stale plan fails instead of blindly overwriting newer content.
func (s *Service) Replace(path, oldText, newText string, replaceAll bool) (ReplaceResult, error) {
	if oldText == "" {
		return ReplaceResult{}, errors.New("old text is required")
	}
	read, err := s.Read(path, DefaultReadLimit)
	if err != nil {
		return ReplaceResult{}, err
	}
	count := strings.Count(read.Content, oldText)
	if count == 0 {
		return ReplaceResult{}, errors.New("old text was not found; file may have changed")
	}
	replacements := 1
	limit := 1
	if replaceAll {
		replacements = count
		limit = -1
	}
	updated := strings.Replace(read.Content, oldText, newText, limit)
	write, err := s.Write(path, updated, false)
	if err != nil {
		return ReplaceResult{}, err
	}
	return ReplaceResult{Path: write.Path, Replacements: replacements, Size: write.Bytes}, nil
}

func (s *Service) Mkdir(path string, parents bool) (MkdirResult, error) {
	if strings.TrimSpace(path) == "" {
		return MkdirResult{}, ErrPathRequired
	}
	resolved, err := s.resolveTarget(path)
	if err != nil {
		return MkdirResult{}, err
	}
	if filepath.Clean(resolved) == s.root {
		return MkdirResult{Path: s.root, Created: false}, nil
	}
	if info, statErr := os.Stat(resolved); statErr == nil {
		if !info.IsDir() {
			return MkdirResult{}, errors.New("target already exists and is not a directory")
		}
		return MkdirResult{Path: resolved, Created: false}, nil
	} else if !os.IsNotExist(statErr) {
		return MkdirResult{}, statErr
	}
	if parents {
		err = os.MkdirAll(resolved, 0o755)
	} else {
		err = os.Mkdir(resolved, 0o755)
	}
	if err != nil {
		return MkdirResult{}, fmt.Errorf("create directory: %w", err)
	}
	return MkdirResult{Path: resolved, Created: true}, nil
}

func (s *Service) Remove(path string, recursive bool) (RemoveResult, error) {
	if strings.TrimSpace(path) == "" {
		return RemoveResult{}, ErrPathRequired
	}
	resolved, err := s.resolveExisting(path, false)
	if err != nil {
		return RemoveResult{}, err
	}
	if filepath.Clean(resolved) == s.root {
		return RemoveResult{}, ErrWorkspaceRoot
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return RemoveResult{}, err
	}
	if info.IsDir() && recursive {
		err = os.RemoveAll(resolved)
	} else {
		err = os.Remove(resolved)
	}
	if err != nil {
		return RemoveResult{}, fmt.Errorf("remove path: %w", err)
	}
	return RemoveResult{Path: resolved, Directory: info.IsDir(), Recursive: recursive}, nil
}

func (s *Service) resolveExisting(value string, defaultRoot bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		if !defaultRoot {
			return "", ErrPathRequired
		}
		value = "."
	}
	candidate, err := s.lexicalCandidate(value)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	resolved = filepath.Clean(resolved)
	if err := s.ensureInside(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

// resolveTarget validates both existing and not-yet-existing targets. For a
// missing path it walks upward until an existing ancestor is found and resolves
// that ancestor's symlinks before accepting the target.
func (s *Service) resolveTarget(value string) (string, error) {
	candidate, err := s.lexicalCandidate(value)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(candidate); err == nil {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return "", err
		}
		if err := s.ensureInside(resolved); err != nil {
			return "", err
		}
		return filepath.Clean(resolved), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	ancestor := filepath.Dir(candidate)
	for {
		if _, err := os.Lstat(ancestor); err == nil {
			resolved, err := filepath.EvalSymlinks(ancestor)
			if err != nil {
				return "", err
			}
			if err := s.ensureInside(resolved); err != nil {
				return "", err
			}
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(ancestor)
		if next == ancestor {
			return "", ErrOutsideWorkspace
		}
		ancestor = next
	}
	return candidate, nil
}

func (s *Service) lexicalCandidate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrPathRequired
	}
	candidate := value
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(s.root, candidate)
	}
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	if err := s.ensureInside(absolute); err != nil {
		return "", err
	}
	return absolute, nil
}

func (s *Service) ensureInside(path string) error {
	path = filepath.Clean(path)
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return ErrOutsideWorkspace
	}
	return nil
}

func regularSize(info os.FileInfo) int64 {
	if info.Mode().IsRegular() {
		return info.Size()
	}
	return 0
}

func validUTF8(data []byte) bool { return utf8.Valid(data) }

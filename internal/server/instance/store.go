package instance

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

var ErrNotFound = errors.New("game instance not found")

type Store struct {
	mu   sync.RWMutex
	path string
	data map[string]Instance
}

type storeFile struct {
	Version int        `json:"version"`
	Items   []Instance `json:"items"`
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: filepath.Clean(path), data: map[string]Instance{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var f storeFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	for _, item := range f.Items {
		if strings.TrimSpace(item.ID) != "" {
			s.data[item.ID] = item
		}
	}
	return nil
}

func (s *Store) List() []Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Instance, 0, len(s.data))
	for _, item := range s.data {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GameID == out[j].GameID {
			return out[i].Name < out[j].Name
		}
		return out[i].GameID < out[j].GameID
	})
	return out
}

func (s *Store) Get(id string) (Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.data[strings.TrimSpace(id)]
	if !ok {
		return Instance{}, ErrNotFound
	}
	return item, nil
}

func (s *Store) Upsert(item Instance) (Instance, error) {
	if strings.TrimSpace(item.ID) == "" {
		return Instance{}, errors.New("game instance id is required")
	}
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.data[item.ID]; ok && item.CreatedAt == 0 {
		item.CreatedAt = old.CreatedAt
	}
	if item.CreatedAt == 0 {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	old, hadOld := s.data[item.ID]
	s.data[item.ID] = item
	if err := s.persistLocked(); err != nil {
		if hadOld {
			s.data[item.ID] = old
		} else {
			delete(s.data, item.ID)
		}
		return Instance{}, err
	}
	return item, nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	old, ok := s.data[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	if err := s.persistLocked(); err != nil {
		s.data[id] = old
		return err
	}
	return nil
}

func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	items := make([]Instance, 0, len(s.data))
	for _, item := range s.data {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	raw, err := json.MarshalIndent(storeFile{Version: 1, Items: items}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return platformfiles.AtomicReplace(tmp, s.path)
}

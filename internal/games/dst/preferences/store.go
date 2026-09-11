package preferences

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Value contains DST-specific AGMP preferences, separate from both generic
// AGMP UI settings and Klei server configuration files.
type Value struct {
	DedicatedServerPath      string `json:"dedicatedServerPath"`
	DedicatedServerExtraArgs string `json:"dedicatedServerExtraArgs"`
}

type Store struct{ path string }

func New(path string) *Store { return &Store{path: path} }

func (s *Store) Load() Value {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Value{}
	}
	var value Value
	if json.Unmarshal(data, &value) != nil {
		return Value{}
	}
	return value
}

func (s *Store) Save(value Value) error {
	value.DedicatedServerPath = strings.TrimSpace(value.DedicatedServerPath)
	value.DedicatedServerExtraArgs = strings.TrimSpace(value.DedicatedServerExtraArgs)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		// Windows does not reliably replace an existing destination with
		// os.Rename. Retry after removing the previous settings file so
		// repeated saves work on the primary AGMP platform.
		if removeErr := os.Remove(s.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmp)
			return removeErr
		}
		if retryErr := os.Rename(tmp, s.path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
}

func (s *Store) SetDedicatedServerPath(path string) error {
	value := s.Load()
	value.DedicatedServerPath = path
	return s.Save(value)
}

func (s *Store) SetExtraArgs(extra string) error {
	value := s.Load()
	value.DedicatedServerExtraArgs = extra
	return s.Save(value)
}

var ErrInvalidStore = errors.New("invalid DST preference store")

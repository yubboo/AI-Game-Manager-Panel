package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Settings struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Debug    bool   `json:"debug"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func Default() Settings {
	return Settings{
		Theme:    "dark",
		Language: "zh-CN",
		Debug:    false,
	}
}

func (s *Store) Load() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}

	value := Default()
	if err := json.Unmarshal(data, &value); err != nil {
		return Default(), err
	}
	return value, nil
}

func (s *Store) Save(value Settings) error {
	if value.Theme != "dark" && value.Theme != "light" && value.Theme != "system" {
		value.Theme = "dark"
	}
	if value.Language == "" {
		value.Language = "zh-CN"
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}

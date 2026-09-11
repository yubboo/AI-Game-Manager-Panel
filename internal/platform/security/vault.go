package platformsecurity

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

// Vault stores small local secrets outside ordinary AGMP settings files.
// Platform-specific seal/unseal functions protect the payload at rest.
type Vault struct {
	root string
}

func NewVault(root string) *Vault {
	return &Vault{root: filepath.Clean(root)}
}

func (v *Vault) Put(name, value string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("secret name is required")
	}
	if err := os.MkdirAll(v.root, 0o700); err != nil {
		return err
	}
	sealed, err := sealSecret([]byte(value), v.root)
	if err != nil {
		return err
	}
	data := []byte(base64.StdEncoding.EncodeToString(sealed))
	temp, err := os.CreateTemp(v.root, ".secret-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return platformfiles.AtomicReplace(tempPath, v.path(name))
}

func (v *Vault) Get(name string) (string, error) {
	data, err := os.ReadFile(v.path(name))
	if err != nil {
		return "", err
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	plain, err := unsealSecret(sealed, v.root)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (v *Vault) Delete(name string) error {
	err := os.Remove(v.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (v *Vault) Exists(name string) bool {
	_, err := os.Stat(v.path(name))
	return err == nil
}

func (v *Vault) path(name string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return filepath.Join(v.root, fmt.Sprintf("%x.secret", sum[:16]))
}

//go:build !windows

package platformsecurity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func masterKey(root string) ([]byte, error) {
	path := filepath.Join(root, "vault.key")
	if data, err := os.ReadFile(path); err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid vault key length: %d", len(data))
		}
		return data, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid vault key length: %d", len(data))
		}
		return data, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := file.Write(key); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return key, nil
}

func sealSecret(plain []byte, root string) ([]byte, error) {
	key, err := masterKey(root)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func unsealSecret(sealed []byte, root string) ([]byte, error) {
	key, err := masterKey(root)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, fmt.Errorf("invalid sealed secret")
	}
	nonce, payload := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

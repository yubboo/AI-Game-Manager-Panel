package license

import (
	"crypto/ed25519"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
)

// VendorKeyEntry 描述可以由客户端信任的发行公钥。
// status=active 的密钥用于签发新许可证；legacy/retired 仅用于验证历史许可证。
type VendorKeyEntry struct {
	KeyID       string `json:"keyId"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"createdAt"`
	Note        string `json:"note,omitempty"`
}

type VendorKeyRing struct {
	SchemaVersion int              `json:"schemaVersion"`
	ActiveKeyID   string           `json:"activeKeyId"`
	Keys          []VendorKeyEntry `json:"keys"`
}

//go:embed vendor_public_keys.json
var vendorKeyRingJSON []byte

var (
	vendorKeyRingOnce sync.Once
	vendorKeyRingData VendorKeyRing
)

// DefaultVendorKeyRing 返回编译进客户端的公开发行密钥环副本。
// 私钥永远不应进入本包、源码仓库或客户端发行物。
func DefaultVendorKeyRing() VendorKeyRing {
	vendorKeyRingOnce.Do(func() {
		_ = json.Unmarshal(vendorKeyRingJSON, &vendorKeyRingData)
	})
	copyValue := vendorKeyRingData
	copyValue.Keys = append([]VendorKeyEntry(nil), vendorKeyRingData.Keys...)
	return copyValue
}

func ActiveVendorKey() (VendorKeyEntry, bool) {
	ring := DefaultVendorKeyRing()
	activeID := strings.TrimSpace(ring.ActiveKeyID)
	if activeID == "" {
		return VendorKeyEntry{}, false
	}
	for _, entry := range ring.Keys {
		if strings.EqualFold(strings.TrimSpace(entry.KeyID), activeID) && strings.EqualFold(strings.TrimSpace(entry.Status), "active") {
			return entry, true
		}
	}
	return VendorKeyEntry{}, false
}

func ActiveVendorPublicKey() string {
	entry, ok := ActiveVendorKey()
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.PublicKey)
}

func ActiveVendorKeyID() string {
	entry, ok := ActiveVendorKey()
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.KeyID)
}

func TrustedVendorPublicKeys() []string {
	ring := DefaultVendorKeyRing()
	values := make([]string, 0, len(ring.Keys))
	seen := map[string]struct{}{}
	for _, entry := range ring.Keys {
		value := strings.TrimSpace(entry.PublicKey)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		if _, err := DecodeEd25519PublicKey(value); err != nil {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func DecodeEd25519PublicKey(value string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, ErrPublicKeyMissing
	}
	return ed25519.PublicKey(raw), nil
}

func PublicKeyFingerprint(publicKey []byte) string {
	if len(publicKey) != ed25519.PublicKeySize {
		return ""
	}
	sum := sha256.Sum256(publicKey)
	return "SHA256:" + strings.ToUpper(hex.EncodeToString(sum[:]))
}

func PublicKeyID(publicKey []byte) string {
	if len(publicKey) != ed25519.PublicKeySize {
		return ""
	}
	sum := sha256.Sum256(publicKey)
	short := strings.ToUpper(hex.EncodeToString(sum[:16]))
	return "AGMP-KID-" + groupCode(short, 8)
}

// legacyPublicKeyID preserves verification compatibility for development-era
// certificates issued before the product abbreviation was normalized to AGMP.
// New certificates must always use PublicKeyID.
func legacyPublicKeyID(publicKey []byte) string {
	if len(publicKey) != ed25519.PublicKeySize {
		return ""
	}
	sum := sha256.Sum256(publicKey)
	short := strings.ToUpper(hex.EncodeToString(sum[:16]))
	return "AI" + "GMP-KID-" + groupCode(short, 8)
}

func matchesPublicKeyID(publicKey []byte, declared string) bool {
	declared = strings.TrimSpace(declared)
	return strings.EqualFold(declared, PublicKeyID(publicKey)) || strings.EqualFold(declared, legacyPublicKeyID(publicKey))
}

func PublicKeyInfoFromBase64(value string) (keyID string, fingerprint string, ok bool) {
	pub, err := DecodeEd25519PublicKey(value)
	if err != nil {
		return "", "", false
	}
	return PublicKeyID(pub), PublicKeyFingerprint(pub), true
}

func (s *Service) verificationPublicKeys() []ed25519.PublicKey {
	values := make([]string, 0, 1+len(s.config.TrustedPublicKeys))
	if strings.TrimSpace(s.config.PublicKey) != "" {
		values = append(values, s.config.PublicKey)
	}
	values = append(values, s.config.TrustedPublicKeys...)
	seen := map[string]struct{}{}
	result := make([]ed25519.PublicKey, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		pub, err := DecodeEd25519PublicKey(value)
		if err != nil {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, pub)
	}
	return result
}

func (s *Service) activeVendorMetadata() (keyID, fingerprint string, trustedCount int) {
	keys := s.verificationPublicKeys()
	trustedCount = len(keys)
	if strings.TrimSpace(s.config.PublicKey) != "" {
		if pub, err := DecodeEd25519PublicKey(s.config.PublicKey); err == nil {
			return PublicKeyID(pub), PublicKeyFingerprint(pub), trustedCount
		}
	}
	if entry, ok := ActiveVendorKey(); ok {
		return entry.KeyID, entry.Fingerprint, trustedCount
	}
	return "", "", trustedCount
}

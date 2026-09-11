package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
)

var (
	ErrInvalidCertificate          = errors.New("许可证证书无效或签名校验失败")
	ErrMachineMismatch             = errors.New("许可证与当前机器码不匹配")
	ErrInstallMismatch             = errors.New("许可证与当前安装 ID 不匹配")
	ErrExpired                     = errors.New("许可证已过期")
	ErrPublicKeyMissing            = errors.New("许可证公钥未配置")
	ErrOnlineActivationUnavailable = errors.New("在线 CDK 激活服务尚未配置，请使用离线许可证证书或等待 License Server 上线")
	ErrFeatureNotEntitled          = errors.New("当前许可证未授权此功能")
)

var DefaultVendorPublicKey = ActiveVendorPublicKey()

const (
	StateUnlicensed        = "UNLICENSED"
	StateActive            = "ACTIVE"
	StateExpiring          = "EXPIRING"
	StateExpired           = "EXPIRED"
	StateDeviceMismatch    = "DEVICE_MISMATCH"
	StateRevoked           = "REVOKED"
	StateSeatLimit         = "SEAT_LIMIT"
	StateOfflineGrace      = "OFFLINE_GRACE"
	StateServerUnavailable = "SERVER_UNREACHABLE"
	StateDevelopment       = "DEVELOPMENT"
)

var StandardFeatures = []string{"server.basic"}
var ProFeatures = []string{"server.basic", "server.multi", "backup.advanced", "remote.agent", "web.remote", "automation", "ai.workbench", "plugin.extensions"}

func DefaultConfig() Config {
	return Config{Enabled: true, Product: "AI Game Manager Panel", PublicKey: DefaultVendorPublicKey, TrustedPublicKeys: TrustedVendorPublicKeys()}
}

type Config struct {
	Enabled           bool
	Product           string
	PublicKey         string
	TrustedPublicKeys []string
}

// CertificatePayload 是客户端离线可验证的许可证证书内容。
// 私钥只存在发行端；客户端仅持有公钥并验证 BFLC2 签名。
type CertificatePayload struct {
	Version      int      `json:"version"`
	LicenseID    string   `json:"licenseId"`
	ActivationID string   `json:"activationId"`
	Product      string   `json:"product"`
	MachineCode  string   `json:"machineCode"`
	InstallID    string   `json:"installId"`
	Edition      string   `json:"edition"`
	Features     []string `json:"features"`
	SeatLimit    int      `json:"seatLimit"`
	IssuedAt     int64    `json:"issuedAt"`
	ExpiresAt    int64    `json:"expiresAt"`
	IssuerKeyID  string   `json:"issuerKeyId,omitempty"`
}

// Payload 保留旧 BFCDK1/BFCDK2 读取兼容，仅用于迁移历史许可证。
type Payload struct {
	Version            int    `json:"version"`
	LicenseID          string `json:"licenseId"`
	Product            string `json:"product"`
	MachineCode        string `json:"machineCode"`
	IdentificationCode string `json:"identificationCode,omitempty"`
	Edition            string `json:"edition"`
	IssuedAt           int64  `json:"issuedAt"`
	ExpiresAt          int64  `json:"expiresAt"`
}

type Record struct {
	Certificate string `json:"certificate"`
	ActivatedAt int64  `json:"activatedAt"`
	LegacyCDK   string `json:"legacyCdk,omitempty"`
	CDK         string `json:"cdk,omitempty"` // 读取 0.1.62 activation.json；新写入不再使用。
}

type Status struct {
	Enabled              bool     `json:"enabled"`
	Activated            bool     `json:"activated"`
	Valid                bool     `json:"valid"`
	State                string   `json:"state"`
	MachineCode          string   `json:"machineCode"`
	InstallID            string   `json:"installId"`
	IdentificationCode   string   `json:"identificationCode"` // 兼容 0.1.62 UI，值与 InstallID 相同。
	LicenseID            string   `json:"licenseId"`
	ActivationID         string   `json:"activationId"`
	Edition              string   `json:"edition"`
	Features             []string `json:"features"`
	SeatLimit            int      `json:"seatLimit"`
	ActiveSeats          int      `json:"activeSeats"`
	IssuedAt             int64    `json:"issuedAt"`
	ExpiresAt            int64    `json:"expiresAt"`
	ActivatedAt          int64    `json:"activatedAt"`
	Legacy               bool     `json:"legacy"`
	IssuerKeyID          string   `json:"issuerKeyId"`
	VendorKeyID          string   `json:"vendorKeyId"`
	VendorKeyFingerprint string   `json:"vendorKeyFingerprint"`
	TrustedKeyCount      int      `json:"trustedKeyCount"`
	Message              string   `json:"message"`
}

type ActivateRequest struct {
	ActivationCode string `json:"activationCode,omitempty"`
	Certificate    string `json:"certificate,omitempty"`
	CDK            string `json:"cdk,omitempty"` // 兼容旧客户端；BFCDK 只作为历史证书导入。
}

type FeatureStatus struct {
	Feature string `json:"feature"`
	Allowed bool   `json:"allowed"`
	State   string `json:"state"`
	Message string `json:"message"`
}

type Service struct {
	config         Config
	recordPath     string
	installIDPath  string
	now            func() time.Time
	installIDOnce  sync.Once
	installIDValue string
}

func New(config Config, recordPath string) *Service {
	if strings.TrimSpace(config.Product) == "" {
		config.Product = "AI Game Manager Panel"
	}
	return &Service{config: config, recordPath: recordPath, installIDPath: filepath.Join(filepath.Dir(recordPath), "install.id"), now: time.Now}
}

var (
	machineCodeOnce  sync.Once
	machineCodeValue string
)

const machineCodeDerivationNamespace = "AI Game Manager Panel-License-v2|" // 已进入授权协议，禁止随品牌文案再次修改。

func MachineCode() string {
	machineCodeOnce.Do(func() {
		machineCodeValue = deriveMachineCode(machineIdentity())
	})
	return machineCodeValue
}

func deriveMachineCode(raw string) string {
	sum := sha256.Sum256([]byte(machineCodeDerivationNamespace + raw))
	value := strings.ToUpper(hex.EncodeToString(sum[:16]))
	return "BFM-" + groupCode(value, 8)
}

func groupCode(s string, size int) string {
	parts := make([]string, 0, (len(s)+size-1)/size)
	for len(s) > 0 {
		n := size
		if len(s) < n {
			n = len(s)
		}
		parts = append(parts, s[:n])
		s = s[n:]
	}
	return strings.Join(parts, "-")
}

func (s *Service) installID() string {
	s.installIDOnce.Do(func() { s.installIDValue = s.loadOrCreateInstallID() })
	return s.installIDValue
}

func (s *Service) loadOrCreateInstallID() string {
	if b, err := os.ReadFile(s.installIDPath); err == nil {
		value := strings.ToUpper(strings.TrimSpace(string(b)))
		if strings.HasPrefix(value, "BFID-") {
			return value
		}
	}
	// 0.1.62 把同一标识保存为 device.id。升级时必须沿用原 BFID，
	// 否则已经签发的 BFCDK2 会因为安装 ID 突然变化而失效。
	legacyPath := filepath.Join(filepath.Dir(s.recordPath), "device.id")
	if b, err := os.ReadFile(legacyPath); err == nil {
		value := strings.ToUpper(strings.TrimSpace(string(b)))
		if strings.HasPrefix(value, "BFID-") {
			if err := os.MkdirAll(filepath.Dir(s.installIDPath), 0o700); err == nil {
				_ = os.WriteFile(s.installIDPath, []byte(value+"\n"), 0o600)
			}
			return value
		}
	}
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		sum := sha256.Sum256([]byte(MachineCode() + "|" + s.recordPath + "|AI Game Manager Panel-Install-v2"))
		copy(raw, sum[:12])
	}
	value := "BFID-" + groupCode(strings.ToUpper(hex.EncodeToString(raw)), 8)
	if err := os.MkdirAll(filepath.Dir(s.installIDPath), 0o700); err == nil {
		_ = os.WriteFile(s.installIDPath, []byte(value+"\n"), 0o600)
	}
	return value
}

func (s *Service) Status() Status {
	machine := MachineCode()
	installID := s.installID()
	vendorKeyID, vendorKeyFingerprint, trustedKeyCount := s.activeVendorMetadata()
	if developmentEntitlementMode {
		return Status{
			Enabled: false, Activated: false, Valid: true, State: StateDevelopment,
			MachineCode: machine, InstallID: installID, IdentificationCode: installID,
			Edition: "development", Features: []string{"*"}, SeatLimit: 1,
			VendorKeyID: vendorKeyID, VendorKeyFingerprint: vendorKeyFingerprint, TrustedKeyCount: trustedKeyCount,
			Message: "源码开发模式：许可证门禁已关闭；正式 Release 不包含此模式",
		}
	}
	st := Status{Enabled: s.config.Enabled, State: StateUnlicensed, MachineCode: machine, InstallID: installID, IdentificationCode: installID, Features: []string{}, VendorKeyID: vendorKeyID, VendorKeyFingerprint: vendorKeyFingerprint, TrustedKeyCount: trustedKeyCount}
	if !s.config.Enabled {
		st.Message = "许可证服务已关闭"
		return st
	}
	b, err := os.ReadFile(s.recordPath)
	if err != nil {
		st.Message = "AI Game Manager Panel 尚未激活；登录、设置和基础诊断仍可使用"
		return st
	}
	var rec Record
	if json.Unmarshal(b, &rec) != nil {
		st.Message = "本地许可证记录损坏，请重新导入许可证"
		return st
	}
	st.ActivatedAt = rec.ActivatedAt
	certificate := strings.TrimSpace(rec.Certificate)
	if certificate == "" {
		certificate = strings.TrimSpace(rec.LegacyCDK)
	}
	if certificate == "" {
		certificate = strings.TrimSpace(rec.CDK)
	}
	if certificate == "" {
		st.Message = "本地许可证记录为空，请重新激活"
		return st
	}
	payload, legacy, err := s.verifyAny(certificate, machine, installID)
	st.Activated = true
	st.Legacy = legacy
	if err != nil {
		switch {
		case errors.Is(err, ErrMachineMismatch), errors.Is(err, ErrInstallMismatch):
			st.State = StateDeviceMismatch
		case errors.Is(err, ErrExpired):
			st.State = StateExpired
		default:
			st.State = StateUnlicensed
		}
		st.Message = err.Error()
		return st
	}
	st.Valid = true
	st.State = StateActive
	st.LicenseID = payload.LicenseID
	st.ActivationID = payload.ActivationID
	st.IssuerKeyID = payload.IssuerKeyID
	st.Edition = payload.Edition
	st.Features = append([]string(nil), payload.Features...)
	st.SeatLimit = payload.SeatLimit
	st.ActiveSeats = 1
	st.IssuedAt = payload.IssuedAt
	st.ExpiresAt = payload.ExpiresAt
	if payload.ExpiresAt > 0 && payload.ExpiresAt-s.now().Unix() <= 14*24*60*60 {
		st.State = StateExpiring
		st.Message = "许可证有效，但将在 14 天内到期"
	} else if legacy {
		st.Message = "历史 BFCDK 许可证有效；建议重新签发 BFLC2 许可证证书"
	} else {
		st.Message = "许可证已激活"
	}
	return st
}

func (s *Service) Activate(req ActivateRequest) (Status, error) {
	code := strings.TrimSpace(req.ActivationCode)
	certificate := strings.TrimSpace(req.Certificate)
	if certificate == "" {
		certificate = strings.TrimSpace(req.CDK)
	}
	if certificate == "" && code != "" {
		if strings.HasPrefix(strings.ToUpper(code), "BF-") {
			return s.Status(), ErrOnlineActivationUnavailable
		}
		certificate = code // 允许用户把 BFLC2 直接粘贴到激活框，减少界面理解成本。
	}
	if certificate == "" {
		return s.Status(), errors.New("请输入激活码或导入许可证证书")
	}
	machine := MachineCode()
	installID := s.installID()
	_, _, err := s.verifyAny(certificate, machine, installID)
	if err != nil {
		return s.Status(), err
	}
	rec := Record{Certificate: certificate, ActivatedAt: s.now().Unix()}
	if strings.HasPrefix(certificate, "BFCDK") {
		rec.LegacyCDK = certificate
	}
	b, _ := json.MarshalIndent(rec, "", "  ")
	if err := s.writeActivationRecord(append(b, '\n')); err != nil {
		return s.Status(), err
	}
	return s.Status(), nil
}

// writeActivationRecord 使用同目录临时文件 + 原子替换保存激活记录。
// 这样即使程序在写盘过程中异常退出，也不会留下半截 activation.json。
func (s *Service) writeActivationRecord(data []byte) error {
	dir := filepath.Dir(s.recordPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".activation-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return platformfiles.AtomicReplace(tmpPath, s.recordPath)
}

func (s *Service) Unbind() (Status, error) {
	if err := os.Remove(s.recordPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return s.Status(), err
	}
	return s.Status(), nil
}

// VerifyCertificateForDevice verifies a BFLC2/BFCDK certificate against an explicit
// machine/install pair without mutating the local activation record. It is used by
// the offline issuer console to validate a certificate before it is handed to a user.
func (s *Service) VerifyCertificateForDevice(value, machine, installID string) (CertificatePayload, bool, error) {
	return s.verifyAny(strings.TrimSpace(value), strings.TrimSpace(machine), strings.TrimSpace(installID))
}

func (s *Service) HasFeature(feature string) FeatureStatus {
	feature = strings.TrimSpace(feature)
	st := s.Status()
	result := FeatureStatus{Feature: feature, State: st.State}
	if !st.Valid {
		result.Message = "当前设备没有有效许可证"
		return result
	}
	for _, value := range st.Features {
		if value == "*" || strings.EqualFold(strings.TrimSpace(value), feature) {
			result.Allowed = true
			result.Message = "许可证已包含此功能"
			return result
		}
	}
	result.Message = "当前许可证未包含此功能"
	return result
}

// RequireFeature is the backend enforcement point for licensed capabilities.
// UI route gates are only presentation; every future privileged application/API
// operation must call this method (or Application.RequireLicenseFeature) before
// executing business logic.
func (s *Service) RequireFeature(feature string) error {
	status := s.HasFeature(feature)
	if status.Allowed {
		return nil
	}
	feature = strings.TrimSpace(feature)
	if feature == "" {
		feature = "<empty>"
	}
	return fmt.Errorf("%w: %s (%s): %s", ErrFeatureNotEntitled, feature, status.State, status.Message)
}

func (s *Service) verifyAny(value, machine, installID string) (CertificatePayload, bool, error) {
	if strings.HasPrefix(value, "BFLC2.") {
		payload, err := s.verifyCertificate(value, machine, installID)
		return payload, false, err
	}
	if strings.HasPrefix(value, "BFCDK1.") || strings.HasPrefix(value, "BFCDK2.") {
		legacy, err := s.verifyLegacy(value, machine, installID)
		if err != nil {
			return CertificatePayload{}, true, err
		}
		features := StandardFeatures
		if strings.EqualFold(legacy.Edition, "pro") {
			features = ProFeatures
		}
		return CertificatePayload{Version: 2, LicenseID: legacy.LicenseID, ActivationID: "LEGACY-" + legacy.LicenseID, Product: legacy.Product, MachineCode: legacy.MachineCode, InstallID: installID, Edition: legacy.Edition, Features: append([]string(nil), features...), SeatLimit: 1, IssuedAt: legacy.IssuedAt, ExpiresAt: legacy.ExpiresAt}, true, nil
	}
	return CertificatePayload{}, false, ErrInvalidCertificate
}

func (s *Service) verifyCertificate(certificate, machine, installID string) (CertificatePayload, error) {
	var p CertificatePayload
	publicKeys := s.verificationPublicKeys()
	if len(publicKeys) == 0 {
		return p, ErrPublicKeyMissing
	}
	parts := strings.Split(certificate, ".")
	if len(parts) != 3 || parts[0] != "BFLC2" {
		return p, ErrInvalidCertificate
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return p, ErrInvalidCertificate
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return p, ErrInvalidCertificate
	}
	verified := false
	signingKeyID := ""
	var signingPublicKey ed25519.PublicKey
	for _, publicKey := range publicKeys {
		if ed25519.Verify(publicKey, payloadRaw, sig) {
			verified = true
			signingPublicKey = publicKey
			signingKeyID = PublicKeyID(publicKey)
			break
		}
	}
	if !verified {
		return p, ErrInvalidCertificate
	}
	if json.Unmarshal(payloadRaw, &p) != nil || p.Version != 2 || !strings.EqualFold(p.Product, s.config.Product) || strings.TrimSpace(p.LicenseID) == "" || strings.TrimSpace(p.ActivationID) == "" {
		return p, ErrInvalidCertificate
	}
	// 0.1.70 起 IssuerKeyID 不再只是显示字段：证书声明的 KeyID 必须与
	// 实际完成 Ed25519 验签的公钥一致。旧 BFLC2 若未携带该字段，则在
	// 本次验证结果中补齐真实 KeyID，以保持历史许可证兼容。
	if declared := strings.TrimSpace(p.IssuerKeyID); declared != "" && !matchesPublicKeyID(signingPublicKey, declared) {
		return CertificatePayload{}, ErrInvalidCertificate
	}
	p.IssuerKeyID = signingKeyID
	if !strings.EqualFold(strings.TrimSpace(p.MachineCode), strings.TrimSpace(machine)) {
		return p, ErrMachineMismatch
	}
	if !strings.EqualFold(strings.TrimSpace(p.InstallID), strings.TrimSpace(installID)) {
		return p, ErrInstallMismatch
	}
	if p.ExpiresAt > 0 && s.now().Unix() > p.ExpiresAt {
		return p, ErrExpired
	}
	return p, nil
}

func (s *Service) verifyLegacy(cdk, machine, installID string) (Payload, error) {
	var p Payload
	publicKeys := s.verificationPublicKeys()
	if len(publicKeys) == 0 {
		return p, ErrPublicKeyMissing
	}
	parts := strings.Split(cdk, ".")
	if len(parts) != 3 || (parts[0] != "BFCDK2" && parts[0] != "BFCDK1") {
		return p, ErrInvalidCertificate
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return p, ErrInvalidCertificate
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return p, ErrInvalidCertificate
	}
	verified := false
	for _, publicKey := range publicKeys {
		if ed25519.Verify(publicKey, payloadRaw, sig) {
			verified = true
			break
		}
	}
	if !verified {
		return p, ErrInvalidCertificate
	}
	if json.Unmarshal(payloadRaw, &p) != nil || (p.Version != 1 && p.Version != 2) || !strings.EqualFold(p.Product, s.config.Product) {
		return p, ErrInvalidCertificate
	}
	if !strings.EqualFold(strings.TrimSpace(p.MachineCode), strings.TrimSpace(machine)) {
		return p, ErrMachineMismatch
	}
	if parts[0] == "BFCDK2" || p.Version >= 2 {
		if strings.TrimSpace(p.IdentificationCode) == "" || !strings.EqualFold(strings.TrimSpace(p.IdentificationCode), strings.TrimSpace(installID)) {
			return p, ErrInstallMismatch
		}
	}
	if p.ExpiresAt > 0 && s.now().Unix() > p.ExpiresAt {
		return p, ErrExpired
	}
	return p, nil
}

func IssueCertificate(privateKey []byte, payload CertificatePayload) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("private key size invalid")
	}
	payload.Version = 2
	publicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	if strings.TrimSpace(payload.IssuerKeyID) == "" {
		payload.IssuerKeyID = PublicKeyID(publicKey)
	}
	if strings.TrimSpace(payload.LicenseID) == "" || strings.TrimSpace(payload.ActivationID) == "" || strings.TrimSpace(payload.MachineCode) == "" || strings.TrimSpace(payload.InstallID) == "" {
		return "", fmt.Errorf("licenseId, activationId, machineCode and installId are required")
	}
	if strings.TrimSpace(payload.Product) == "" {
		payload.Product = "AI Game Manager Panel"
	}
	if strings.TrimSpace(payload.Edition) == "" {
		payload.Edition = "standard"
	}
	if len(payload.Features) == 0 {
		payload.Features = append([]string(nil), StandardFeatures...)
		if strings.EqualFold(payload.Edition, "pro") {
			payload.Features = append([]string(nil), ProFeatures...)
		}
	}
	if payload.SeatLimit <= 0 {
		payload.SeatLimit = 1
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privateKey), b)
	return "BFLC2." + base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Issue 仅为旧签发工具兼容保留；新版本应使用 IssueCertificate。
func Issue(privateKey []byte, payload Payload) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("private key size invalid")
	}
	payload.Version = 2
	if strings.TrimSpace(payload.IdentificationCode) == "" {
		return "", fmt.Errorf("identification code required")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(ed25519.PrivateKey(privateKey), b)
	return "BFCDK2." + base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func NewTimeOrderedID(prefix string) string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	ms := uint64(time.Now().UnixMilli())
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	raw[6] = (raw[6] & 0x0f) | 0x70
	raw[8] = (raw[8] & 0x3f) | 0x80
	hexValue := strings.ToUpper(hex.EncodeToString(raw[:]))
	return strings.TrimSpace(prefix) + "-" + hexValue[0:8] + "-" + hexValue[8:12] + "-" + hexValue[12:16] + "-" + hexValue[16:20] + "-" + hexValue[20:32]
}

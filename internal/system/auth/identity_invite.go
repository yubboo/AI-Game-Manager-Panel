package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	platformsecurity "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/security"
)

const (
	DefaultOrganizationName = "AGMP 管理组织"
	DefaultManagementGroup  = "management"
	identitySecretName      = "auth:instance:ed25519"
	bootstrapSecretName     = "auth:bootstrap:complete"
	signedStoreSecretName   = "auth:accounts:signed-v3"
	inviteTokenPrefix       = "AGMPI1"
	defaultInviteLifetime   = 24 * time.Hour
	maxInviteLifetime       = 7 * 24 * time.Hour
)

var (
	ErrInvitationRequired   = errors.New("公开注册已关闭，请联系管理员邀请加入组织")
	ErrInvalidInvitation    = errors.New("邀请无效、已过期、已使用或不属于当前 AGMP 实例")
	ErrOrganizationRequired = errors.New("你尚未加入该组织，请联系管理员邀请加入组织")
)

type InstanceIdentity struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	CreatedAt   int64  `json:"createdAt"`
}
type identityFile struct {
	Version     int    `json:"version"`
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
	CreatedAt   int64  `json:"createdAt"`
}
type Organization struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DefaultGroupID string `json:"defaultGroupId"`
	CreatedAt      int64  `json:"createdAt"`
}

type Invitation struct {
	ID               string `json:"id"`
	OrganizationID   string `json:"organizationId"`
	GroupID          string `json:"groupId"`
	Role             string `json:"role"`
	TargetUsername   string `json:"targetUsername,omitempty"`
	TargetEmail      string `json:"targetEmail,omitempty"`
	SecretHash       string `json:"secretHash"`
	VerificationCode string `json:"verificationCode"`
	CreatedBy        string `json:"createdBy"`
	CreatedAt        int64  `json:"createdAt"`
	ExpiresAt        int64  `json:"expiresAt"`
	UsedAt           int64  `json:"usedAt,omitempty"`
	UsedBy           string `json:"usedBy,omitempty"`
	RevokedAt        int64  `json:"revokedAt,omitempty"`
}
type InvitationView struct {
	ID               string `json:"id"`
	Role             string `json:"role"`
	GroupID          string `json:"groupId"`
	TargetUsername   string `json:"targetUsername,omitempty"`
	TargetEmail      string `json:"targetEmail,omitempty"`
	VerificationCode string `json:"verificationCode"`
	CreatedBy        string `json:"createdBy"`
	CreatedAt        int64  `json:"createdAt"`
	ExpiresAt        int64  `json:"expiresAt"`
	UsedAt           int64  `json:"usedAt,omitempty"`
	UsedBy           string `json:"usedBy,omitempty"`
	RevokedAt        int64  `json:"revokedAt,omitempty"`
	Status           string `json:"status"`
}
type CreateInvitationRequest struct {
	Role           string `json:"role"`
	TargetUsername string `json:"targetUsername,omitempty"`
	TargetEmail    string `json:"targetEmail,omitempty"`
	ExpiresInHours int    `json:"expiresInHours,omitempty"`
}
type CreatedInvitation struct {
	InvitationView
	Token               string `json:"token"`
	InstanceID          string `json:"instanceId"`
	InstanceFingerprint string `json:"instanceFingerprint"`
	OrganizationName    string `json:"organizationName"`
}
type InvitationPreview struct {
	InvitationID        string `json:"invitationId"`
	OrganizationName    string `json:"organizationName"`
	Role                string `json:"role"`
	GroupID             string `json:"groupId"`
	TargetUsername      string `json:"targetUsername,omitempty"`
	TargetEmail         string `json:"targetEmail,omitempty"`
	ExpiresAt           int64  `json:"expiresAt"`
	VerificationCode    string `json:"verificationCode"`
	InstanceID          string `json:"instanceId"`
	InstanceFingerprint string `json:"instanceFingerprint"`
}
type RegisterInvitationRequest struct {
	InvitationToken    string `json:"invitationToken"`
	Username           string `json:"username"`
	DisplayName        string `json:"displayName"`
	Password           string `json:"password"`
	Email              string `json:"email,omitempty"`
	SecurityKey        string `json:"securityKey,omitempty"`
	RequireSecurityKey bool   `json:"requireSecurityKey,omitempty"`
}
type invitePayload struct {
	Version        int    `json:"v"`
	InviteID       string `json:"inviteId"`
	InstanceID     string `json:"instanceId"`
	Fingerprint    string `json:"fingerprint"`
	OrganizationID string `json:"organizationId"`
	GroupID        string `json:"groupId"`
	Role           string `json:"role"`
	TargetUsername string `json:"targetUsername,omitempty"`
	TargetEmail    string `json:"targetEmail,omitempty"`
	ExpiresAt      int64  `json:"expiresAt"`
	Nonce          string `json:"nonce"`
}

func loadOrCreateInstanceIdentity(path string, vault *platformsecurity.Vault, now time.Time) (InstanceIdentity, ed25519.PrivateKey, error) {
	var existing identityFile
	data, readErr := os.ReadFile(path)
	if readErr == nil {
		if err := json.Unmarshal(data, &existing); err != nil {
			return InstanceIdentity{}, nil, fmt.Errorf("AGMP 实例身份文件损坏: %w", err)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return InstanceIdentity{}, nil, readErr
	}
	if vault.Exists(identitySecretName) {
		encoded, err := vault.Get(identitySecretName)
		if err != nil {
			return InstanceIdentity{}, nil, fmt.Errorf("读取 AGMP 实例身份私钥失败: %w", err)
		}
		raw, err := base64.RawStdEncoding.DecodeString(encoded)
		if err != nil || len(raw) != ed25519.PrivateKeySize {
			return InstanceIdentity{}, nil, errors.New("AGMP 实例身份私钥无效，已安全锁定")
		}
		pk := ed25519.PrivateKey(raw)
		identity := identityFromPrivate(pk, existing.CreatedAt)
		if identity.CreatedAt == 0 {
			identity.CreatedAt = now.Unix()
		}
		if readErr == nil {
			if existing.ID != identity.ID || existing.Fingerprint != identity.Fingerprint || existing.PublicKey != identity.PublicKey {
				return InstanceIdentity{}, nil, errors.New("AGMP 实例身份指纹与本机私钥不匹配，已安全锁定")
			}
		} else if err := writeIdentityFile(path, identity); err != nil {
			return InstanceIdentity{}, nil, err
		}
		return identity, pk, nil
	}
	if readErr == nil {
		return InstanceIdentity{}, nil, errors.New("AGMP 实例身份私钥缺失，禁止自动生成新身份；请恢复原实例密钥或从备份恢复")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return InstanceIdentity{}, nil, err
	}
	identity := identityFromPublic(pub, now.Unix())
	if err := vault.Put(identitySecretName, base64.RawStdEncoding.EncodeToString(priv)); err != nil {
		return InstanceIdentity{}, nil, fmt.Errorf("保存 AGMP 实例身份私钥失败: %w", err)
	}
	if err := writeIdentityFile(path, identity); err != nil {
		return InstanceIdentity{}, nil, err
	}
	return identity, priv, nil
}
func identityFromPrivate(privateKey ed25519.PrivateKey, createdAt int64) InstanceIdentity {
	return identityFromPublic(privateKey.Public().(ed25519.PublicKey), createdAt)
}
func identityFromPublic(publicKey ed25519.PublicKey, createdAt int64) InstanceIdentity {
	sum := sha256.Sum256(publicKey)
	shortID := base64.RawURLEncoding.EncodeToString(sum[:16])
	encoded := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:16]))
	groups := []string{}
	for len(encoded) > 0 {
		n := 4
		if len(encoded) < n {
			n = len(encoded)
		}
		groups = append(groups, encoded[:n])
		encoded = encoded[n:]
	}
	return InstanceIdentity{ID: "inst_" + shortID, Fingerprint: "AGMP-" + strings.Join(groups, "-"), PublicKey: base64.RawStdEncoding.EncodeToString(publicKey), CreatedAt: createdAt}
}
func writeIdentityFile(path string, identity InstanceIdentity) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	value := identityFile{Version: 1, ID: identity.ID, Fingerprint: identity.Fingerprint, PublicKey: identity.PublicKey, CreatedAt: identity.CreatedAt}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}
func organizationForIdentity(identity InstanceIdentity, now time.Time) Organization {
	sum := sha256.Sum256([]byte(identity.ID + "|organization"))
	return Organization{ID: "org_" + base64.RawURLEncoding.EncodeToString(sum[:12]), Name: DefaultOrganizationName, DefaultGroupID: DefaultManagementGroup, CreatedAt: now.Unix()}
}
func (s *Service) InstanceIdentity() InstanceIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.identity
}
func (s *Service) Organization(token string) (Organization, error) {
	user, err := s.Validate(token)
	if err != nil {
		return Organization{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.organization.ID == "" || user.OrganizationID != s.organization.ID {
		return Organization{}, ErrOrganizationRequired
	}
	return s.organization, nil
}

func (s *Service) CreateInvitation(token string, request CreateInvitationRequest) (CreatedInvitation, error) {
	caller, err := s.RequireSensitiveAction(token, "签发组织成员邀请")
	if err != nil {
		return CreatedInvitation{}, err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return CreatedInvitation{}, ErrForbidden
	}
	role := strings.ToLower(strings.TrimSpace(request.Role))
	if role == "" {
		role = RoleOperator
	}
	if role != RoleOperator && role != RoleAdministrator {
		return CreatedInvitation{}, errors.New("邀请角色只允许 administrator 或 operator")
	}
	if role == RoleAdministrator && caller.Role != RoleOwner {
		return CreatedInvitation{}, fmt.Errorf("%w: 只有最高管理员可以邀请其他管理员", ErrForbidden)
	}
	target := normalizeUsername(request.TargetUsername)
	if request.TargetUsername != "" {
		if _, _, err := validateIdentity(request.TargetUsername, request.TargetUsername); err != nil {
			return CreatedInvitation{}, fmt.Errorf("目标用户名无效: %w", err)
		}
	}
	targetEmail := ""
	if strings.TrimSpace(request.TargetEmail) != "" {
		targetEmail, err = normalizeEmail(request.TargetEmail)
		if err != nil {
			return CreatedInvitation{}, fmt.Errorf("目标邮箱无效: %w", err)
		}
	}
	lifetime := defaultInviteLifetime
	if request.ExpiresInHours > 0 {
		lifetime = time.Duration(request.ExpiresInHours) * time.Hour
	}
	if lifetime < time.Hour || lifetime > maxInviteLifetime {
		return CreatedInvitation{}, errors.New("邀请有效期必须在 1 小时到 7 天之间")
	}
	inviteID, err := randomToken(16)
	if err != nil {
		return CreatedInvitation{}, err
	}
	nonceRaw := make([]byte, 32)
	if _, err := rand.Read(nonceRaw); err != nil {
		return CreatedInvitation{}, err
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceRaw)
	now := s.now()
	exp := now.Add(lifetime)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return CreatedInvitation{}, fmt.Errorf("账号数据不可用: %w", s.loadErr)
	}
	if caller.OrganizationID == "" || caller.OrganizationID != s.organization.ID {
		return CreatedInvitation{}, ErrOrganizationRequired
	}
	payload := invitePayload{Version: 1, InviteID: inviteID, InstanceID: s.identity.ID, Fingerprint: s.identity.Fingerprint, OrganizationID: s.organization.ID, GroupID: s.organization.DefaultGroupID, Role: role, TargetUsername: target, TargetEmail: targetEmail, ExpiresAt: exp.Unix(), Nonce: nonce}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return CreatedInvitation{}, err
	}
	signature := ed25519.Sign(s.identityPrivate, payloadBytes)
	tokenValue := inviteTokenPrefix + "." + base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + base64.RawURLEncoding.EncodeToString(signature)
	secretSum := sha256.Sum256([]byte(nonce))
	record := Invitation{ID: inviteID, OrganizationID: s.organization.ID, GroupID: s.organization.DefaultGroupID, Role: role, TargetUsername: target, TargetEmail: targetEmail, SecretHash: base64.RawStdEncoding.EncodeToString(secretSum[:]), VerificationCode: inviteVerificationCode(signature), CreatedBy: caller.ID, CreatedAt: now.Unix(), ExpiresAt: exp.Unix()}
	s.invitations = append(s.invitations, record)
	if err := s.saveLocked(); err != nil {
		s.invitations = s.invitations[:len(s.invitations)-1]
		return CreatedInvitation{}, err
	}
	return CreatedInvitation{InvitationView: s.invitationViewLocked(record), Token: tokenValue, InstanceID: s.identity.ID, InstanceFingerprint: s.identity.Fingerprint, OrganizationName: s.organization.Name}, nil
}
func (s *Service) ListInvitations(token string) ([]InvitationView, error) {
	caller, err := s.RequireCoreAccess(token)
	if err != nil {
		return nil, err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return nil, ErrForbidden
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if caller.OrganizationID != s.organization.ID {
		return nil, ErrOrganizationRequired
	}
	values := make([]InvitationView, 0, len(s.invitations))
	for i := len(s.invitations) - 1; i >= 0; i-- {
		values = append(values, s.invitationViewLocked(s.invitations[i]))
	}
	return values, nil
}
func (s *Service) RevokeInvitation(token, id string) (InvitationView, error) {
	caller, err := s.RequireSensitiveAction(token, "撤销组织成员邀请")
	if err != nil {
		return InvitationView{}, err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return InvitationView{}, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.invitations {
		inv := &s.invitations[i]
		if inv.ID != strings.TrimSpace(id) {
			continue
		}
		if inv.OrganizationID != s.organization.ID {
			return InvitationView{}, ErrOrganizationRequired
		}
		if inv.Role == RoleAdministrator && caller.Role != RoleOwner {
			return InvitationView{}, ErrForbidden
		}
		if inv.UsedAt != 0 {
			return InvitationView{}, errors.New("邀请已被使用，不能撤销")
		}
		if inv.RevokedAt == 0 {
			inv.RevokedAt = s.now().Unix()
			if err := s.saveLocked(); err != nil {
				inv.RevokedAt = 0
				return InvitationView{}, err
			}
		}
		return s.invitationViewLocked(*inv), nil
	}
	return InvitationView{}, ErrInvalidInvitation
}
func (s *Service) InspectInvitation(token string) (InvitationPreview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	payload, record, sig, err := s.verifyInvitationTokenLocked(token)
	if err != nil {
		return InvitationPreview{}, err
	}
	return InvitationPreview{InvitationID: record.ID, OrganizationName: s.organization.Name, Role: record.Role, GroupID: record.GroupID, TargetUsername: record.TargetUsername, TargetEmail: record.TargetEmail, ExpiresAt: record.ExpiresAt, VerificationCode: inviteVerificationCode(sig), InstanceID: payload.InstanceID, InstanceFingerprint: payload.Fingerprint}, nil
}
func (s *Service) RegisterWithInvitation(request RegisterInvitationRequest) (Session, error) {
	username, displayName, err := validateIdentity(request.Username, request.DisplayName)
	if err != nil {
		return Session{}, err
	}
	if err := validatePassword(request.Password); err != nil {
		return Session{}, err
	}
	email := ""
	if strings.TrimSpace(request.Email) != "" {
		email, err = normalizeEmail(request.Email)
		if err != nil {
			return Session{}, err
		}
	}
	securityKey := strings.TrimSpace(request.SecurityKey)
	if securityKey != "" {
		if err := validateSecurityKey(securityKey); err != nil {
			return Session{}, err
		}
	}
	if request.RequireSecurityKey && securityKey == "" {
		return Session{}, errors.New("启用登录安全密钥验证前必须先生成并保存安全密钥")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return Session{}, fmt.Errorf("账号数据不可用: %w", s.loadErr)
	}
	payload, record, _, err := s.verifyInvitationTokenLocked(request.InvitationToken)
	if err != nil {
		return Session{}, err
	}
	if record.TargetUsername != "" && normalizeUsername(username) != record.TargetUsername {
		return Session{}, errors.New("该邀请已绑定指定用户名，当前用户名不匹配")
	}
	if record.TargetEmail != "" && !strings.EqualFold(email, record.TargetEmail) {
		return Session{}, errors.New("该邀请已绑定指定邮箱，当前邮箱不匹配")
	}
	if s.usernameExistsLocked(username) {
		return Session{}, fmt.Errorf("用户名 %q 已存在", username)
	}
	if payload.Role != RoleOperator && payload.Role != RoleAdministrator {
		return Session{}, ErrInvalidInvitation
	}
	account, err := newAccount(username, displayName, request.Password, payload.Role, s.now())
	if err != nil {
		return Session{}, err
	}
	account.OrganizationID = s.organization.ID
	account.GroupID = payload.GroupID
	account.Email = email
	account.CoreAccess = CoreAccessPending
	account.RiskState = RiskStateNormal
	if securityKey != "" {
		account.SecurityKeyHash = hashSecurityKey(securityKey)
		account.RequireSecurityKey = request.RequireSecurityKey
	}
	inviteIndex := -1
	for i := range s.invitations {
		if s.invitations[i].ID == record.ID {
			inviteIndex = i
			break
		}
	}
	if inviteIndex < 0 {
		return Session{}, ErrInvalidInvitation
	}
	previous := s.invitations[inviteIndex]
	s.invitations[inviteIndex].UsedAt = s.now().Unix()
	s.invitations[inviteIndex].UsedBy = account.ID
	s.accounts = append(s.accounts, account)
	if err := s.saveLocked(); err != nil {
		s.accounts = s.accounts[:len(s.accounts)-1]
		s.invitations[inviteIndex] = previous
		return Session{}, err
	}
	return s.newSessionLocked(account.User)
}
func (s *Service) verifyInvitationTokenLocked(token string) (invitePayload, Invitation, []byte, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != inviteTokenPrefix {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payloadBytes) > 4096 {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) != ed25519.SignatureSize {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	pub, err := base64.RawStdEncoding.DecodeString(s.identity.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(pub), payloadBytes, sig) {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	var payload invitePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	if payload.Version != 1 || payload.InstanceID != s.identity.ID || payload.Fingerprint != s.identity.Fingerprint || payload.OrganizationID != s.organization.ID || payload.GroupID != s.organization.DefaultGroupID || payload.ExpiresAt <= s.now().Unix() {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	var record Invitation
	found := false
	for _, c := range s.invitations {
		if c.ID == payload.InviteID {
			record = c
			found = true
			break
		}
	}
	if !found || record.OrganizationID != s.organization.ID || record.GroupID != payload.GroupID || record.Role != payload.Role || record.ExpiresAt != payload.ExpiresAt || record.TargetUsername != payload.TargetUsername || record.TargetEmail != payload.TargetEmail || record.UsedAt != 0 || record.RevokedAt != 0 {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	sum := sha256.Sum256([]byte(payload.Nonce))
	expected, err := base64.RawStdEncoding.DecodeString(record.SecretHash)
	if err != nil || len(expected) != sha256.Size || subtle.ConstantTimeCompare(sum[:], expected) != 1 {
		return invitePayload{}, Invitation{}, nil, ErrInvalidInvitation
	}
	return payload, record, sig, nil
}
func (s *Service) invitationViewLocked(r Invitation) InvitationView {
	status := "active"
	now := s.now().Unix()
	if r.RevokedAt != 0 {
		status = "revoked"
	} else if r.UsedAt != 0 {
		status = "used"
	} else if r.ExpiresAt <= now {
		status = "expired"
	}
	return InvitationView{ID: r.ID, Role: r.Role, GroupID: r.GroupID, TargetUsername: r.TargetUsername, TargetEmail: r.TargetEmail, VerificationCode: r.VerificationCode, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, UsedAt: r.UsedAt, UsedBy: r.UsedBy, RevokedAt: r.RevokedAt, Status: status}
}
func inviteVerificationCode(signature []byte) string {
	sum := sha256.Sum256(signature)
	encoded := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:5]))
	if len(encoded) > 8 {
		encoded = encoded[:8]
	}
	if len(encoded) > 4 {
		return encoded[:4] + "-" + encoded[4:]
	}
	return encoded
}
func signAccountFile(file accountFile, privateKey ed25519.PrivateKey) (string, error) {
	file.Signature = ""
	data, err := json.Marshal(file)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, data)), nil
}
func verifyAccountFile(file accountFile, publicKey ed25519.PublicKey) bool {
	sig, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(file.Signature))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	file.Signature = ""
	data, err := json.Marshal(file)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, data, sig)
}

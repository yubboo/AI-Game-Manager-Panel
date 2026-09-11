package auth

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	platformsecurity "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/security"
)

const (
	RoleOwner         = "owner"
	RoleAdministrator = "administrator"
	RoleOperator      = "operator"

	CoreAccessPending    = "pending"
	CoreAccessAuthorized = "authorized"
	CoreAccessSuspended  = "suspended"

	RiskStateNormal    = "normal"
	RiskStateChallenge = "challenge"
	RiskStateLocked    = "locked"

	defaultIterations = 120000
	sessionLifetime   = 24 * time.Hour
	stepUpLifetime    = 15 * time.Minute
)

var (
	ErrBootstrapClosed           = errors.New("首个最高管理员已创建，公开注册入口已永久关闭")
	ErrInvalidLogin              = errors.New("用户名、密码或安全密钥错误")
	ErrSecurityKeyRequired       = errors.New("该账号已启用登录安全密钥验证，请输入安全密钥")
	ErrSecurityKeyNotConfigured  = errors.New("该账号尚未配置登录安全密钥")
	ErrUnauthorized              = errors.New("登录会话无效或已过期")
	ErrForbidden                 = errors.New("当前账号没有执行该操作的权限")
	ErrCoreAuthorizationRequired = errors.New("当前成员尚未获得超级管理员签发的核心功能授权码")
	ErrCoreAccessSuspended       = errors.New("当前成员的核心功能授权已被暂停，请联系超级管理员")
	ErrEmailStepUpRequired       = errors.New("检测到高风险操作，需要完成邮箱二次验证后才能继续")
	ErrCredentialStepUpRequired  = errors.New("检测到高风险操作，需要重新验证账号密码或安全密钥后才能继续")
	ErrRiskLocked                = errors.New("账号已被安全风控锁定，请完成安全验证或联系超级管理员解除风控")
)

type User struct {
	ID               string `json:"id"`
	Username         string `json:"username"`
	DisplayName      string `json:"displayName"`
	Role             string `json:"role"`
	OrganizationID   string `json:"organizationId"`
	GroupID          string `json:"groupId"`
	Email            string `json:"email,omitempty"`
	EmailVerifiedAt  int64  `json:"emailVerifiedAt,omitempty"`
	CoreAccess       string `json:"coreAccess"`
	CoreAuthorizedAt int64  `json:"coreAuthorizedAt,omitempty"`
	CoreAuthorizedBy string `json:"coreAuthorizedBy,omitempty"`
	RiskState        string `json:"riskState"`
	RiskReason       string `json:"riskReason,omitempty"`
	RiskUpdatedAt    int64  `json:"riskUpdatedAt,omitempty"`
	CreatedAt        int64  `json:"createdAt"`
}

type accountRecord struct {
	User
	PasswordSalt       string `json:"passwordSalt"`
	PasswordHash       string `json:"passwordHash"`
	Iterations         int    `json:"iterations"`
	SecurityKeyHash    string `json:"securityKeyHash,omitempty"`
	RequireSecurityKey bool   `json:"requireSecurityKey,omitempty"`
	AuthEpoch          uint64 `json:"authEpoch,omitempty"`
}

type accountFile struct {
	Version              int                   `json:"version"`
	Organization         Organization          `json:"organization"`
	Accounts             []accountRecord       `json:"accounts"`
	Invitations          []Invitation          `json:"invitations,omitempty"`
	MemberAuthorizations []MemberAuthorization `json:"memberAuthorizations,omitempty"`
	Signature            string                `json:"signature,omitempty"`
}

type BootstrapStatus struct {
	OwnerExists                bool   `json:"ownerExists"`
	RegistrationOpen           bool   `json:"registrationOpen"`
	InvitationRegistrationOnly bool   `json:"invitationRegistrationOnly"`
	BootstrapLocked            bool   `json:"bootstrapLocked"`
	StoreError                 string `json:"storeError"`
	UserCount                  int    `json:"userCount"`
	InstanceID                 string `json:"instanceId"`
	InstanceFingerprint        string `json:"instanceFingerprint"`
}

type CreateOwnerRequest struct {
	Username           string `json:"username"`
	DisplayName        string `json:"displayName"`
	Password           string `json:"password"`
	Email              string `json:"email,omitempty"`
	SecurityKey        string `json:"securityKey"`
	RequireSecurityKey bool   `json:"requireSecurityKey"`
}

type LoginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	SecurityKey string `json:"securityKey"`
}

type SecurityKeyMaterial struct {
	Key string `json:"key"`
}
type SecurityStatus struct {
	Configured          bool `json:"configured"`
	VerificationEnabled bool `json:"verificationEnabled"`
}
type RotateSecurityKeyRequest struct {
	Password string `json:"password"`
}
type SetSecurityKeyVerificationRequest struct {
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}
type UpdateDisplayNameRequest struct {
	DisplayName string `json:"displayName"`
}

// CreateUserRequest remains for source compatibility only. Direct member
// creation is intentionally disabled after 0.1.89; signed invitations are the
// only path into an initialized organization.
type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

type Session struct {
	Token     string `json:"token"`
	User      User   `json:"user"`
	ExpiresAt int64  `json:"expiresAt"`
}
type sessionRecord struct {
	Session
	expires     time.Time
	authEpoch   uint64
	stepUpUntil time.Time
}

type Service struct {
	path                      string
	lockPath                  string
	identityPath              string
	now                       func() time.Time
	mu                        sync.RWMutex
	accounts                  []accountRecord
	invitations               []Invitation
	memberAuthorizations      []MemberAuthorization
	organization              Organization
	identity                  InstanceIdentity
	identityPrivate           ed25519.PrivateKey
	vault                     *platformsecurity.Vault
	sessions                  map[string]sessionRecord
	emailChallenges           map[string]emailChallenge
	passwordResetChallenges   map[string]passwordResetChallenge
	loginFailures             map[string][]time.Time
	bootstrapLocked           bool
	loadErr                   error
	allowLegacyUnsignedStore  bool
	needsSignedStoreMigration bool
}

func New(path string) *Service {
	authRoot := filepath.Dir(path)
	s := &Service{
		path: path, lockPath: filepath.Join(authRoot, "bootstrap.lock"), identityPath: filepath.Join(authRoot, "instance-identity.json"),
		now: time.Now, vault: platformsecurity.NewVault(filepath.Join(authRoot, "secrets")), sessions: map[string]sessionRecord{},
		emailChallenges: map[string]emailChallenge{}, passwordResetChallenges: map[string]passwordResetChallenge{}, loginFailures: map[string][]time.Time{},
	}
	if _, err := os.Stat(s.lockPath); err == nil || s.vault.Exists(bootstrapSecretName) {
		s.bootstrapLocked = true
	}

	legacyVersion, legacyExists := peekAccountStoreVersion(path)
	identityWasPresent := s.vault.Exists(identitySecretName)
	if _, err := os.Stat(s.identityPath); err == nil {
		identityWasPresent = true
	}
	signedStoreWasRequired := s.vault.Exists(signedStoreSecretName)
	s.allowLegacyUnsignedStore = legacyExists && legacyVersion > 0 && legacyVersion < 3 && !identityWasPresent && !signedStoreWasRequired

	identity, privateKey, err := loadOrCreateInstanceIdentity(s.identityPath, s.vault, s.now())
	if err != nil {
		s.loadErr = err
		return s
	}
	s.identity, s.identityPrivate = identity, privateKey
	if err := s.load(); err != nil {
		s.loadErr = err
		return s
	}
	changed := s.ensureOrganizationMembershipLocked()
	if !s.bootstrapLocked {
		for _, account := range s.accounts {
			if account.Role == RoleOwner {
				s.bootstrapLocked = true
				_ = s.writeBootstrapLock()
				_ = s.vault.Put(bootstrapSecretName, "complete")
				break
			}
		}
	}
	if changed || s.needsSignedStoreMigration {
		if err := s.saveLocked(); err != nil {
			s.loadErr = fmt.Errorf("升级账号组织/签名数据失败: %w", err)
			return s
		}
	}
	if !s.vault.Exists(signedStoreSecretName) {
		if err := s.vault.Put(signedStoreSecretName, "required"); err != nil {
			s.loadErr = fmt.Errorf("保存账号签名降级保护标记失败: %w", err)
		}
	}
	return s
}

func (s *Service) BootstrapStatus() BootstrapStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	owner := false
	for _, account := range s.accounts {
		if account.Role == RoleOwner {
			owner = true
			break
		}
	}
	storeError := ""
	if s.loadErr != nil {
		storeError = s.loadErr.Error()
	}
	publicUserCount := 0
	if !owner && !s.bootstrapLocked {
		publicUserCount = len(s.accounts)
	}
	return BootstrapStatus{OwnerExists: owner, RegistrationOpen: !owner && !s.bootstrapLocked && s.loadErr == nil, InvitationRegistrationOnly: owner || s.bootstrapLocked, BootstrapLocked: s.bootstrapLocked, StoreError: storeError, UserCount: publicUserCount, InstanceID: s.identity.ID, InstanceFingerprint: s.identity.Fingerprint}
}

func (s *Service) CreateInitialOwner(request CreateOwnerRequest) (Session, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		displayName = "超级管理员"
	}
	username, displayName, err := validateIdentity(request.Username, displayName)
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
		return Session{}, fmt.Errorf("账号数据不可用，Bootstrap 已安全锁定: %w", s.loadErr)
	}
	if s.bootstrapLocked {
		return Session{}, ErrBootstrapClosed
	}
	for _, account := range s.accounts {
		if account.Role == RoleOwner {
			return Session{}, ErrBootstrapClosed
		}
	}
	if s.usernameExistsLocked(username) {
		return Session{}, fmt.Errorf("用户名 %q 已存在", username)
	}

	account, err := newAccount(username, displayName, request.Password, RoleOwner, s.now())
	if err != nil {
		return Session{}, err
	}
	if s.organization.ID == "" {
		s.organization = organizationForIdentity(s.identity, s.now())
	}
	account.OrganizationID = s.organization.ID
	account.GroupID = s.organization.DefaultGroupID
	account.Email = email
	account.CoreAccess = CoreAccessAuthorized
	account.CoreAuthorizedAt = s.now().Unix()
	account.CoreAuthorizedBy = account.ID
	account.RiskState = RiskStateNormal
	if securityKey != "" {
		account.SecurityKeyHash = hashSecurityKey(securityKey)
		account.RequireSecurityKey = request.RequireSecurityKey
	}
	s.accounts = append(s.accounts, account)
	if err := s.saveLocked(); err != nil {
		s.accounts = s.accounts[:len(s.accounts)-1]
		return Session{}, err
	}
	if err := s.writeBootstrapLock(); err != nil {
		return Session{}, fmt.Errorf("最高管理员已写入，但 Bootstrap 锁写入失败: %w", err)
	}
	if err := s.vault.Put(bootstrapSecretName, "complete"); err != nil {
		return Session{}, fmt.Errorf("最高管理员已写入，但实例 Bootstrap 安全标记写入失败: %w", err)
	}
	s.bootstrapLocked = true
	return s.newSessionLocked(account.User)
}

func (s *Service) Login(request LoginRequest) (Session, error) {
	username := normalizeUsername(request.Username)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return Session{}, fmt.Errorf("账号数据不可用: %w", s.loadErr)
	}
	s.pruneSessionsLocked()
	for i := range s.accounts {
		account := &s.accounts[i]
		if normalizeUsername(account.Username) != username {
			continue
		}
		if !verifyPassword(request.Password, *account) {
			s.recordLoginFailureLocked(i)
			return Session{}, ErrInvalidLogin
		}
		if account.RequireSecurityKey {
			if strings.TrimSpace(request.SecurityKey) == "" {
				s.recordLoginFailureLocked(i)
				return Session{}, ErrSecurityKeyRequired
			}
			if !verifySecurityKey(request.SecurityKey, account.SecurityKeyHash) {
				s.recordLoginFailureLocked(i)
				return Session{}, ErrInvalidLogin
			}
		}
		if account.OrganizationID == "" || account.OrganizationID != s.organization.ID {
			return Session{}, ErrOrganizationRequired
		}
		s.clearLoginFailuresLocked(account.ID)
		return s.newSessionLocked(account.User)
	}
	return Session{}, ErrInvalidLogin
}

func (s *Service) Validate(token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, ErrUnauthorized
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.sessions[token]
	if !ok || !record.expires.After(s.now()) {
		delete(s.sessions, token)
		return User{}, ErrUnauthorized
	}
	idx := s.accountIndexLocked(record.User.ID)
	if idx < 0 {
		delete(s.sessions, token)
		return User{}, ErrUnauthorized
	}
	account := s.accounts[idx]
	if account.OrganizationID == "" || account.OrganizationID != s.organization.ID {
		delete(s.sessions, token)
		return User{}, ErrOrganizationRequired
	}
	if record.authEpoch != account.AuthEpoch {
		delete(s.sessions, token)
		return User{}, ErrUnauthorized
	}
	if record.User != account.User {
		record.User = account.User
		s.sessions[token] = record
	}
	return account.User, nil
}

func (s *Service) Logout(token string) {
	s.mu.Lock()
	delete(s.sessions, strings.TrimSpace(token))
	s.mu.Unlock()
}

func (s *Service) ListUsers(token string) ([]User, error) {
	caller, err := s.Validate(token)
	if err != nil {
		return nil, err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return nil, ErrForbidden
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]User, 0, len(s.accounts))
	for _, account := range s.accounts {
		if account.OrganizationID == caller.OrganizationID {
			users = append(users, account.User)
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].CreatedAt < users[j].CreatedAt })
	return users, nil
}

func (s *Service) UpdateMyDisplayName(token string, request UpdateDisplayNameRequest) (User, error) {
	user, err := s.Validate(token)
	if err != nil {
		return User{}, err
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		return User{}, errors.New("显示名称不能为空")
	}
	if len([]rune(displayName)) > 40 {
		return User{}, errors.New("显示名称不能超过 40 个字符")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return User{}, ErrUnauthorized
	}
	previous := s.accounts[idx].DisplayName
	s.accounts[idx].DisplayName = displayName
	if err := s.saveLocked(); err != nil {
		s.accounts[idx].DisplayName = previous
		return User{}, err
	}
	s.refreshUserSessionsLocked(s.accounts[idx].User, s.accounts[idx].AuthEpoch, false)
	return s.accounts[idx].User, nil
}

func (s *Service) CreateUser(token string, request CreateUserRequest) (User, error) {
	if _, err := s.Validate(token); err != nil {
		return User{}, err
	}
	_ = request
	return User{}, ErrInvitationRequired
}

func GenerateSecurityKey() (SecurityKeyMaterial, error) {
	token, err := randomToken(32)
	if err != nil {
		return SecurityKeyMaterial{}, err
	}
	return SecurityKeyMaterial{Key: "BFK1-" + token}, nil
}

func (s *Service) MySecurityStatus(token string) (SecurityStatus, error) {
	user, err := s.Validate(token)
	if err != nil {
		return SecurityStatus{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return SecurityStatus{}, ErrUnauthorized
	}
	a := s.accounts[idx]
	return SecurityStatus{Configured: a.SecurityKeyHash != "", VerificationEnabled: a.RequireSecurityKey}, nil
}

func (s *Service) RotateMySecurityKey(token string, request RotateSecurityKeyRequest) (SecurityKeyMaterial, error) {
	user, err := s.Validate(token)
	if err != nil {
		return SecurityKeyMaterial{}, err
	}
	material, err := GenerateSecurityKey()
	if err != nil {
		return SecurityKeyMaterial{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return SecurityKeyMaterial{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	if !verifyPassword(request.Password, *a) {
		return SecurityKeyMaterial{}, ErrInvalidLogin
	}
	prev := a.SecurityKeyHash
	a.SecurityKeyHash = hashSecurityKey(material.Key)
	if err := s.saveLocked(); err != nil {
		a.SecurityKeyHash = prev
		return SecurityKeyMaterial{}, err
	}
	return material, nil
}

func (s *Service) SetMySecurityKeyVerification(token string, request SetSecurityKeyVerificationRequest) (SecurityStatus, error) {
	user, err := s.Validate(token)
	if err != nil {
		return SecurityStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return SecurityStatus{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	if !verifyPassword(request.Password, *a) {
		return SecurityStatus{}, ErrInvalidLogin
	}
	if request.Enabled && a.SecurityKeyHash == "" {
		return SecurityStatus{}, ErrSecurityKeyNotConfigured
	}
	prev := a.RequireSecurityKey
	a.RequireSecurityKey = request.Enabled
	if err := s.saveLocked(); err != nil {
		a.RequireSecurityKey = prev
		return SecurityStatus{}, err
	}
	return SecurityStatus{Configured: a.SecurityKeyHash != "", VerificationEnabled: a.RequireSecurityKey}, nil
}

func (s *Service) usernameExistsLocked(username string) bool {
	n := normalizeUsername(username)
	for _, a := range s.accounts {
		if normalizeUsername(a.Username) == n {
			return true
		}
	}
	return false
}

func (s *Service) newSessionLocked(user User) (Session, error) {
	token, err := randomToken(32)
	if err != nil {
		return Session{}, err
	}
	exp := s.now().Add(sessionLifetime)
	epoch := uint64(0)
	if idx := s.accountIndexLocked(user.ID); idx >= 0 {
		epoch = s.accounts[idx].AuthEpoch
		user = s.accounts[idx].User
	}
	session := Session{Token: token, User: user, ExpiresAt: exp.Unix()}
	s.sessions[token] = sessionRecord{Session: session, expires: exp, authEpoch: epoch}
	return session, nil
}
func (s *Service) pruneSessionsLocked() {
	now := s.now()
	for token, record := range s.sessions {
		if !record.expires.After(now) {
			delete(s.sessions, token)
		}
	}
}
func (s *Service) writeBootstrapLock() error {
	if err := os.MkdirAll(filepath.Dir(s.lockPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.lockPath, []byte("AI Game Manager Panel bootstrap registration is permanently closed.\n"), 0o600)
}

func peekAccountStoreVersion(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var h struct {
		Version int `json:"version"`
	}
	if json.Unmarshal(data, &h) != nil {
		return 0, true
	}
	return h.Version, true
}
func (s *Service) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var file accountFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("读取账号数据失败: %w", err)
	}
	if file.Version < 3 {
		if !s.allowLegacyUnsignedStore {
			return errors.New("检测到账号数据签名版本降级，已安全锁定；请恢复受信任备份")
		}
		s.needsSignedStoreMigration = true
	} else {
		pub, err := base64.RawStdEncoding.DecodeString(s.identity.PublicKey)
		if err != nil || len(pub) != ed25519.PublicKeySize || !verifyAccountFile(file, ed25519.PublicKey(pub)) {
			return errors.New("账号/组织数据完整性签名无效，已安全锁定；请恢复受信任备份")
		}
	}
	s.accounts = file.Accounts
	s.invitations = file.Invitations
	s.memberAuthorizations = file.MemberAuthorizations
	s.organization = file.Organization
	return nil
}
func (s *Service) saveLocked() error {
	if len(s.identityPrivate) != ed25519.PrivateKeySize {
		return errors.New("AGMP 实例身份私钥不可用，拒绝写入账号数据")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	file := accountFile{Version: 3, Organization: s.organization, Accounts: s.accounts, Invitations: s.invitations, MemberAuthorizations: s.memberAuthorizations}
	sig, err := signAccountFile(file, s.identityPrivate)
	if err != nil {
		return err
	}
	file.Signature = sig
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(temp, s.path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}
func (s *Service) ensureOrganizationMembershipLocked() bool {
	changed := false
	ownerExists := false
	for _, a := range s.accounts {
		if a.Role == RoleOwner {
			ownerExists = true
			break
		}
	}
	if ownerExists && s.organization.ID == "" {
		s.organization = organizationForIdentity(s.identity, s.now())
		changed = true
	}
	if s.organization.ID != "" {
		if s.organization.DefaultGroupID == "" {
			s.organization.DefaultGroupID = DefaultManagementGroup
			changed = true
		}
		if s.organization.Name == "" {
			s.organization.Name = DefaultOrganizationName
			changed = true
		}
		for i := range s.accounts {
			a := &s.accounts[i]
			if a.OrganizationID == "" {
				a.OrganizationID = s.organization.ID
				changed = true
			}
			if a.GroupID == "" {
				a.GroupID = s.organization.DefaultGroupID
				changed = true
			}
			if a.RiskState == "" {
				a.RiskState = RiskStateNormal
				changed = true
			}
			if a.AuthEpoch == 0 {
				a.AuthEpoch = 1
				changed = true
			}
			if a.CoreAccess == "" {
				if a.Role == RoleOwner {
					a.CoreAccess = CoreAccessAuthorized
					a.CoreAuthorizedAt = a.CreatedAt
					a.CoreAuthorizedBy = a.ID
				} else {
					a.CoreAccess = CoreAccessPending
				}
				changed = true
			}
		}
	}
	return changed
}

func newAccount(username, displayName, password, role string, now time.Time) (accountRecord, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return accountRecord{}, err
	}
	id, err := randomToken(16)
	if err != nil {
		return accountRecord{}, err
	}
	hash := derivePassword([]byte(password), salt, defaultIterations, 32)
	core := CoreAccessPending
	if role == RoleOwner {
		core = CoreAccessAuthorized
	}
	return accountRecord{User: User{ID: id, Username: username, DisplayName: displayName, Role: role, CoreAccess: core, RiskState: RiskStateNormal, CreatedAt: now.Unix()}, PasswordSalt: base64.RawStdEncoding.EncodeToString(salt), PasswordHash: base64.RawStdEncoding.EncodeToString(hash), Iterations: defaultIterations, AuthEpoch: 1}, nil
}
func verifyPassword(password string, account accountRecord) bool {
	salt, err := base64.RawStdEncoding.DecodeString(account.PasswordSalt)
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(account.PasswordHash)
	if err != nil || len(expected) == 0 {
		return false
	}
	iterations := account.Iterations
	if iterations < 10000 {
		return false
	}
	actual := derivePassword([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// derivePassword implements PBKDF2-HMAC-SHA256 locally to keep the credential format stable.
func derivePassword(password, salt []byte, iterations, keyLen int) []byte {
	hLen := sha256.Size
	blocks := (keyLen + hLen - 1) / hLen
	result := make([]byte, 0, blocks*hLen)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		var index [4]byte
		binary.BigEndian.PutUint32(index[:], uint32(block))
		_, _ = mac.Write(index[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:keyLen]
}
func validateSecurityKey(value string) error {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "BFK1-") {
		return errors.New("安全密钥格式无效，请使用 AI Game Manager Panel 生成的 BFK1 密钥")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "BFK1-"))
	if err != nil || len(raw) != 32 {
		return errors.New("安全密钥格式无效，请重新生成")
	}
	return nil
}
func hashSecurityKey(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}
func verifySecurityKey(value, expectedHash string) bool {
	if strings.TrimSpace(value) == "" || expectedHash == "" {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(expectedHash)
	if err != nil || len(expected) != sha256.Size {
		return false
	}
	actual := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return subtle.ConstantTimeCompare(actual[:], expected) == 1
}
func validateIdentity(username, displayName string) (string, string, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	if len(username) < 3 || len(username) > 32 {
		return "", "", errors.New("用户名长度必须为 3-32 个字符")
	}
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return "", "", errors.New("用户名只能包含字母、数字、点、下划线和短横线")
		}
	}
	if displayName == "" {
		displayName = username
	}
	if len([]rune(displayName)) > 40 {
		return "", "", errors.New("显示名称不能超过 40 个字符")
	}
	return username, displayName, nil
}
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码至少需要 8 个字符")
	}
	if len(password) > 128 {
		return errors.New("密码不能超过 128 个字符")
	}
	return nil
}
func normalizeUsername(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

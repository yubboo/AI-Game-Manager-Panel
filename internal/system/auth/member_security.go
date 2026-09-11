package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	memberAuthorizationPrefix = "AGMPU1"
	defaultMemberAuthLifetime = 24 * time.Hour
	maxMemberAuthLifetime     = 7 * 24 * time.Hour
)

type MemberAuthorization struct {
	ID               string `json:"id"`
	OrganizationID   string `json:"organizationId"`
	UserID           string `json:"userId"`
	SecretHash       string `json:"secretHash"`
	VerificationCode string `json:"verificationCode"`
	CreatedBy        string `json:"createdBy"`
	CreatedAt        int64  `json:"createdAt"`
	ExpiresAt        int64  `json:"expiresAt"`
	UsedAt           int64  `json:"usedAt,omitempty"`
	RevokedAt        int64  `json:"revokedAt,omitempty"`
}
type MemberAuthorizationView struct {
	ID               string `json:"id"`
	UserID           string `json:"userId"`
	Username         string `json:"username"`
	VerificationCode string `json:"verificationCode"`
	CreatedBy        string `json:"createdBy"`
	CreatedAt        int64  `json:"createdAt"`
	ExpiresAt        int64  `json:"expiresAt"`
	UsedAt           int64  `json:"usedAt,omitempty"`
	RevokedAt        int64  `json:"revokedAt,omitempty"`
	Status           string `json:"status"`
}
type CreateMemberAuthorizationRequest struct {
	UserID         string `json:"userId"`
	ExpiresInHours int    `json:"expiresInHours,omitempty"`
}
type CreatedMemberAuthorization struct {
	MemberAuthorizationView
	Token               string `json:"token"`
	InstanceID          string `json:"instanceId"`
	InstanceFingerprint string `json:"instanceFingerprint"`
}
type RedeemMemberAuthorizationRequest struct {
	Token string `json:"token"`
}
type memberAuthorizationPayload struct {
	Version         int    `json:"v"`
	AuthorizationID string `json:"authorizationId"`
	InstanceID      string `json:"instanceId"`
	Fingerprint     string `json:"fingerprint"`
	OrganizationID  string `json:"organizationId"`
	UserID          string `json:"userId"`
	ExpiresAt       int64  `json:"expiresAt"`
	Nonce           string `json:"nonce"`
}
type CredentialStepUpRequest struct {
	Password    string `json:"password"`
	SecurityKey string `json:"securityKey,omitempty"`
}

// CreateMemberAuthorization is deliberately independent from email. Email is an
// optional recovery/step-up provider, never a prerequisite for organization
// membership or core entitlement.
func (s *Service) CreateMemberAuthorization(token string, request CreateMemberAuthorizationRequest) (CreatedMemberAuthorization, error) {
	caller, err := s.RequireSensitiveAction(token, "签发成员核心功能授权码")
	if err != nil {
		return CreatedMemberAuthorization{}, err
	}
	if caller.Role != RoleOwner {
		return CreatedMemberAuthorization{}, ErrForbidden
	}
	userID := strings.TrimSpace(request.UserID)
	if userID == "" {
		return CreatedMemberAuthorization{}, errors.New("必须指定需要授权的成员")
	}
	lifetime := defaultMemberAuthLifetime
	if request.ExpiresInHours > 0 {
		lifetime = time.Duration(request.ExpiresInHours) * time.Hour
	}
	if lifetime < time.Hour || lifetime > maxMemberAuthLifetime {
		return CreatedMemberAuthorization{}, errors.New("成员核心授权码有效期必须在 1 小时到 7 天之间")
	}
	authID, err := randomToken(16)
	if err != nil {
		return CreatedMemberAuthorization{}, err
	}
	nonceRaw := make([]byte, 32)
	if _, err := rand.Read(nonceRaw); err != nil {
		return CreatedMemberAuthorization{}, err
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceRaw)
	now := s.now()
	exp := now.Add(lifetime)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadErr != nil {
		return CreatedMemberAuthorization{}, fmt.Errorf("账号数据不可用: %w", s.loadErr)
	}
	idx := s.accountIndexLocked(userID)
	if idx < 0 || s.accounts[idx].OrganizationID != s.organization.ID {
		return CreatedMemberAuthorization{}, ErrOrganizationRequired
	}
	target := s.accounts[idx]
	if target.Role == RoleOwner {
		return CreatedMemberAuthorization{}, errors.New("最高管理员不需要成员核心授权码")
	}
	previousAuthorizations := append([]MemberAuthorization(nil), s.memberAuthorizations...)
	for i := range s.memberAuthorizations {
		grant := &s.memberAuthorizations[i]
		if grant.UserID == target.ID && grant.UsedAt == 0 && grant.RevokedAt == 0 && grant.ExpiresAt > now.Unix() {
			grant.RevokedAt = now.Unix()
		}
	}
	payload := memberAuthorizationPayload{Version: 1, AuthorizationID: authID, InstanceID: s.identity.ID, Fingerprint: s.identity.Fingerprint, OrganizationID: s.organization.ID, UserID: target.ID, ExpiresAt: exp.Unix(), Nonce: nonce}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return CreatedMemberAuthorization{}, err
	}
	signature := ed25519.Sign(s.identityPrivate, payloadBytes)
	tokenValue := memberAuthorizationPrefix + "." + base64.RawURLEncoding.EncodeToString(payloadBytes) + "." + base64.RawURLEncoding.EncodeToString(signature)
	secretSum := sha256.Sum256([]byte(nonce))
	record := MemberAuthorization{ID: authID, OrganizationID: s.organization.ID, UserID: target.ID, SecretHash: base64.RawStdEncoding.EncodeToString(secretSum[:]), VerificationCode: inviteVerificationCode(signature), CreatedBy: caller.ID, CreatedAt: now.Unix(), ExpiresAt: exp.Unix()}
	s.memberAuthorizations = append(s.memberAuthorizations, record)
	if err := s.saveLocked(); err != nil {
		s.memberAuthorizations = previousAuthorizations
		return CreatedMemberAuthorization{}, err
	}
	return CreatedMemberAuthorization{MemberAuthorizationView: s.memberAuthorizationViewLocked(record), Token: tokenValue, InstanceID: s.identity.ID, InstanceFingerprint: s.identity.Fingerprint}, nil
}
func (s *Service) ListMemberAuthorizations(token string) ([]MemberAuthorizationView, error) {
	caller, err := s.RequireCoreAccess(token)
	if err != nil {
		return nil, err
	}
	if caller.Role != RoleOwner {
		return nil, ErrForbidden
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MemberAuthorizationView, 0, len(s.memberAuthorizations))
	for i := len(s.memberAuthorizations) - 1; i >= 0; i-- {
		out = append(out, s.memberAuthorizationViewLocked(s.memberAuthorizations[i]))
	}
	return out, nil
}
func (s *Service) RedeemMemberAuthorization(sessionToken string, request RedeemMemberAuthorizationRequest) (User, error) {
	caller, err := s.Validate(sessionToken)
	if err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, record, _, err := s.verifyMemberAuthorizationTokenLocked(request.Token)
	if err != nil {
		return User{}, err
	}
	if caller.ID != payload.UserID || record.UserID != caller.ID {
		return User{}, ErrForbidden
	}
	idx := s.accountIndexLocked(caller.ID)
	if idx < 0 {
		return User{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	grantIndex := -1
	for i := range s.memberAuthorizations {
		if s.memberAuthorizations[i].ID == record.ID {
			grantIndex = i
			break
		}
	}
	if grantIndex < 0 {
		return User{}, ErrCoreAuthorizationRequired
	}
	previousGrant := s.memberAuthorizations[grantIndex]
	previous := a.User
	now := s.now().Unix()
	s.memberAuthorizations[grantIndex].UsedAt = now
	a.CoreAccess = CoreAccessAuthorized
	a.CoreAuthorizedAt = now
	a.CoreAuthorizedBy = record.CreatedBy
	a.AuthEpoch++
	if err := s.saveLocked(); err != nil {
		s.memberAuthorizations[grantIndex] = previousGrant
		a.User = previous
		a.AuthEpoch--
		return User{}, err
	}
	s.refreshUserSessionsLocked(a.User, a.AuthEpoch, true)
	return a.User, nil
}
func (s *Service) RevokeMemberCoreAccess(token, userID string) (User, error) {
	caller, err := s.RequireSensitiveAction(token, "暂停成员核心功能授权")
	if err != nil {
		return User{}, err
	}
	if caller.Role != RoleOwner {
		return User{}, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(strings.TrimSpace(userID))
	if idx < 0 || s.accounts[idx].OrganizationID != s.organization.ID {
		return User{}, ErrOrganizationRequired
	}
	if s.accounts[idx].Role == RoleOwner {
		return User{}, errors.New("不能暂停最高管理员自己的核心访问")
	}
	previous := s.accounts[idx].User
	s.accounts[idx].CoreAccess = CoreAccessSuspended
	s.accounts[idx].CoreAuthorizedAt = 0
	s.accounts[idx].CoreAuthorizedBy = caller.ID
	s.accounts[idx].AuthEpoch++
	if err := s.saveLocked(); err != nil {
		s.accounts[idx].User = previous
		s.accounts[idx].AuthEpoch--
		return User{}, err
	}
	s.invalidateUserSessionsLocked(s.accounts[idx].ID)
	return s.accounts[idx].User, nil
}
func (s *Service) RequireCoreAccess(token string) (User, error) {
	user, err := s.Validate(token)
	if err != nil {
		return User{}, err
	}
	if user.Role == RoleOwner {
		return user, nil
	}
	switch user.CoreAccess {
	case CoreAccessAuthorized:
		return user, nil
	case CoreAccessSuspended:
		return User{}, ErrCoreAccessSuspended
	default:
		return User{}, ErrCoreAuthorizationRequired
	}
}

// RequireSensitiveAction selects the strongest configured step-up provider.
// Verified email is preferred when SMTP is configured. Otherwise an existing
// security key is used; if neither exists, current-password re-authentication is
// required. Email therefore improves safety without becoming mandatory.
func (s *Service) RequireSensitiveAction(token, reason string) (User, error) {
	user, err := s.RequireCoreAccess(token)
	if err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return User{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	if a.RiskState == RiskStateLocked {
		return User{}, ErrRiskLocked
	}
	record, ok := s.sessions[strings.TrimSpace(token)]
	if ok && record.stepUpUntil.After(s.now()) {
		return a.User, nil
	}
	if a.RiskState != RiskStateChallenge || strings.TrimSpace(a.RiskReason) != strings.TrimSpace(reason) {
		a.RiskState = RiskStateChallenge
		a.RiskReason = strings.TrimSpace(reason)
		a.RiskUpdatedAt = s.now().Unix()
		if err := s.saveLocked(); err != nil {
			return User{}, err
		}
		s.refreshUserSessionsLocked(a.User, a.AuthEpoch, false)
	}
	if a.EmailVerifiedAt != 0 && s.vault.Exists(smtpConfigSecretName) {
		return User{}, ErrEmailStepUpRequired
	}
	return User{}, ErrCredentialStepUpRequired
}

func (s *Service) ConfirmCredentialStepUp(token string, request CredentialStepUpRequest) (User, error) {
	user, err := s.Validate(token)
	if err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return User{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	if a.RiskState == RiskStateLocked {
		return User{}, ErrRiskLocked
	}
	if !verifyPassword(request.Password, *a) {
		return User{}, ErrInvalidLogin
	}
	if a.SecurityKeyHash != "" && !verifySecurityKey(request.SecurityKey, a.SecurityKeyHash) {
		return User{}, ErrSecurityKeyRequired
	}
	a.RiskState = RiskStateNormal
	a.RiskReason = ""
	a.RiskUpdatedAt = s.now().Unix()
	if err := s.saveLocked(); err != nil {
		return User{}, err
	}
	record := s.sessions[token]
	record.User = a.User
	record.stepUpUntil = s.now().Add(stepUpLifetime)
	s.sessions[token] = record
	return a.User, nil
}
func (s *Service) ClearUserRisk(token, userID string) (User, error) {
	caller, err := s.RequireSensitiveAction(token, "超级管理员解除成员风控")
	if err != nil {
		return User{}, err
	}
	if caller.Role != RoleOwner {
		return User{}, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(strings.TrimSpace(userID))
	if idx < 0 || s.accounts[idx].OrganizationID != s.organization.ID {
		return User{}, ErrOrganizationRequired
	}
	a := &s.accounts[idx]
	a.RiskState = RiskStateNormal
	a.RiskReason = ""
	a.RiskUpdatedAt = s.now().Unix()
	if err := s.saveLocked(); err != nil {
		return User{}, err
	}
	s.refreshUserSessionsLocked(a.User, a.AuthEpoch, true)
	return a.User, nil
}
func (s *Service) RemoveMember(token, userID string) error {
	caller, err := s.RequireSensitiveAction(token, "移除组织成员")
	if err != nil {
		return err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(strings.TrimSpace(userID))
	if idx < 0 || s.accounts[idx].OrganizationID != s.organization.ID {
		return ErrOrganizationRequired
	}
	target := s.accounts[idx]
	if target.Role == RoleOwner || (target.Role == RoleAdministrator && caller.Role != RoleOwner) {
		return ErrForbidden
	}
	previous := append([]accountRecord(nil), s.accounts...)
	s.accounts = append(s.accounts[:idx], s.accounts[idx+1:]...)
	if err := s.saveLocked(); err != nil {
		s.accounts = previous
		return err
	}
	s.invalidateUserSessionsLocked(target.ID)
	delete(s.emailChallenges, target.ID)
	delete(s.passwordResetChallenges, target.ID)
	delete(s.loginFailures, target.ID)
	return nil
}
func (s *Service) CoreSeatCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, a := range s.accounts {
		if a.OrganizationID == s.organization.ID && (a.Role == RoleOwner || a.CoreAccess == CoreAccessAuthorized) {
			count++
		}
	}
	return count
}
func (s *Service) RevokeMemberAuthorization(token, authorizationID string) (MemberAuthorizationView, error) {
	caller, err := s.RequireSensitiveAction(token, "撤销成员核心功能授权码")
	if err != nil {
		return MemberAuthorizationView{}, err
	}
	if caller.Role != RoleOwner {
		return MemberAuthorizationView{}, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.memberAuthorizations {
		r := &s.memberAuthorizations[i]
		if r.ID != strings.TrimSpace(authorizationID) {
			continue
		}
		if r.OrganizationID != s.organization.ID {
			return MemberAuthorizationView{}, ErrOrganizationRequired
		}
		if r.UsedAt != 0 {
			return MemberAuthorizationView{}, errors.New("授权码已使用，不能再撤销；如需停止成员核心访问，请暂停该成员授权")
		}
		prev := r.RevokedAt
		if r.RevokedAt == 0 {
			r.RevokedAt = s.now().Unix()
			if err := s.saveLocked(); err != nil {
				r.RevokedAt = prev
				return MemberAuthorizationView{}, err
			}
		}
		return s.memberAuthorizationViewLocked(*r), nil
	}
	return MemberAuthorizationView{}, ErrCoreAuthorizationRequired
}
func (s *Service) verifyMemberAuthorizationTokenLocked(token string) (memberAuthorizationPayload, MemberAuthorization, []byte, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != memberAuthorizationPrefix {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payloadBytes) > 4096 {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) != ed25519.SignatureSize {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	pub, err := base64.RawStdEncoding.DecodeString(s.identity.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(pub), payloadBytes, sig) {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	var payload memberAuthorizationPayload
	if json.Unmarshal(payloadBytes, &payload) != nil || payload.Version != 1 || payload.InstanceID != s.identity.ID || payload.Fingerprint != s.identity.Fingerprint || payload.OrganizationID != s.organization.ID || payload.ExpiresAt <= s.now().Unix() {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	var record MemberAuthorization
	found := false
	for _, c := range s.memberAuthorizations {
		if c.ID == payload.AuthorizationID {
			record = c
			found = true
			break
		}
	}
	if !found || record.OrganizationID != s.organization.ID || record.UserID != payload.UserID || record.ExpiresAt != payload.ExpiresAt || record.UsedAt != 0 || record.RevokedAt != 0 {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	sum := sha256.Sum256([]byte(payload.Nonce))
	expected, err := base64.RawStdEncoding.DecodeString(record.SecretHash)
	if err != nil || len(expected) != sha256.Size || subtle.ConstantTimeCompare(sum[:], expected) != 1 {
		return memberAuthorizationPayload{}, MemberAuthorization{}, nil, ErrCoreAuthorizationRequired
	}
	return payload, record, sig, nil
}
func (s *Service) memberAuthorizationViewLocked(r MemberAuthorization) MemberAuthorizationView {
	status := "active"
	if r.RevokedAt != 0 {
		status = "revoked"
	} else if r.UsedAt != 0 {
		status = "used"
	} else if r.ExpiresAt <= s.now().Unix() {
		status = "expired"
	}
	username := ""
	if idx := s.accountIndexLocked(r.UserID); idx >= 0 {
		username = s.accounts[idx].Username
	}
	return MemberAuthorizationView{ID: r.ID, UserID: r.UserID, Username: username, VerificationCode: r.VerificationCode, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, UsedAt: r.UsedAt, RevokedAt: r.RevokedAt, Status: status}
}
func (s *Service) accountIndexLocked(userID string) int {
	for i := range s.accounts {
		if s.accounts[i].ID == strings.TrimSpace(userID) {
			return i
		}
	}
	return -1
}
func (s *Service) refreshUserSessionsLocked(user User, epoch uint64, resetAssurance bool) {
	for key, session := range s.sessions {
		if session.User.ID != user.ID {
			continue
		}
		session.User = user
		session.authEpoch = epoch
		if resetAssurance {
			session.stepUpUntil = time.Time{}
		}
		s.sessions[key] = session
	}
}
func (s *Service) invalidateUserSessionsLocked(userID string) {
	for key, session := range s.sessions {
		if session.User.ID == userID {
			delete(s.sessions, key)
		}
	}
}

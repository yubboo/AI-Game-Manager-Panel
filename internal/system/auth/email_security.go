package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const (
	smtpConfigSecretName      = "auth:smtp:config"
	emailChallengeLifetime    = 10 * time.Minute
	emailChallengeMaxAttempts = 5
	passwordResetLifetime     = 15 * time.Minute
)

var (
	ErrEmailDeliveryUnavailable = errors.New("邮件服务尚未配置，请联系管理员在系统设置中配置邮箱服务")
	ErrInvalidEmailCode         = errors.New("邮箱验证码无效或已过期")
	ErrPasswordResetUnavailable = errors.New("该账号未配置可用于找回密码的已验证邮箱")
)

type SMTPSettings struct {
	Configured  bool   `json:"configured"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	From        string `json:"from"`
	TLSMode     string `json:"tlsMode"`
	HasPassword bool   `json:"hasPassword"`
}
type SaveSMTPSettingsRequest struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password,omitempty"`
	From            string `json:"from"`
	TLSMode         string `json:"tlsMode"`
	AccountPassword string `json:"accountPassword"`
	SecurityKey     string `json:"securityKey,omitempty"`
}
type smtpSecretConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	TLSMode  string `json:"tlsMode"`
}
type BindEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type EmailSecurityStatus struct {
	Email                  string `json:"email"`
	Bound                  bool   `json:"bound"`
	Verified               bool   `json:"verified"`
	VerifiedAt             int64  `json:"verifiedAt,omitempty"`
	RiskState              string `json:"riskState"`
	RiskReason             string `json:"riskReason,omitempty"`
	RiskUpdatedAt          int64  `json:"riskUpdatedAt,omitempty"`
	StepUpVerified         bool   `json:"stepUpVerified"`
	StepUpValidUntil       int64  `json:"stepUpValidUntil,omitempty"`
	CoreAccess             string `json:"coreAccess"`
	SMTPConfigured         bool   `json:"smtpConfigured"`
	EmailFeaturesAvailable bool   `json:"emailFeaturesAvailable"`
}
type RequestEmailVerificationRequest struct {
	Purpose string `json:"purpose"`
}
type ConfirmEmailVerificationRequest struct {
	Code string `json:"code"`
}
type RequestPasswordResetRequest struct {
	Username string `json:"username"`
}
type ConfirmPasswordResetRequest struct {
	Username    string `json:"username"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}
type PasswordResetRequestStatus struct {
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}
type emailChallenge struct {
	Purpose   string
	Hash      [32]byte
	ExpiresAt time.Time
	Attempts  int
}
type passwordResetChallenge struct {
	UserID    string
	Hash      [32]byte
	ExpiresAt time.Time
	Attempts  int
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", errors.New("邮箱地址不能为空")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || !strings.EqualFold(strings.TrimSpace(parsed.Address), value) || strings.ContainsAny(value, "\r\n") || len(value) > 254 {
		return "", errors.New("邮箱地址格式无效")
	}
	return value, nil
}

func (s *Service) SMTPSettings(token string) (SMTPSettings, error) {
	caller, err := s.Validate(token)
	if err != nil {
		return SMTPSettings{}, err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return SMTPSettings{}, ErrForbidden
	}
	cfg, err := s.loadSMTPConfig()
	if err != nil {
		if errors.Is(err, ErrEmailDeliveryUnavailable) {
			return SMTPSettings{}, nil
		}
		return SMTPSettings{}, err
	}
	return smtpSettingsView(cfg), nil
}
func (s *Service) SaveSMTPSettings(token string, request SaveSMTPSettingsRequest) (SMTPSettings, error) {
	// SMTP is an instance security/recovery setting, not a paid/core AI feature.
	// An organization administrator may configure it before receiving any
	// member-core grant; current account credentials are still required below.
	caller, err := s.Validate(token)
	if err != nil {
		return SMTPSettings{}, err
	}
	if caller.Role != RoleOwner && caller.Role != RoleAdministrator {
		return SMTPSettings{}, ErrForbidden
	}
	// SMTP controls password recovery and email second-factor delivery. Initial
	// setup must be possible before any mailbox is verified, so current account
	// credentials are always required. Subsequent changes prefer a recent
	// step-up when available, but email itself never becomes mandatory.
	s.mu.RLock()
	idx := s.accountIndexLocked(caller.ID)
	if idx < 0 {
		s.mu.RUnlock()
		return SMTPSettings{}, ErrUnauthorized
	}
	account := s.accounts[idx]
	s.mu.RUnlock()
	if !verifyPassword(request.AccountPassword, account) {
		return SMTPSettings{}, ErrInvalidLogin
	}
	if account.SecurityKeyHash != "" && !verifySecurityKey(request.SecurityKey, account.SecurityKeyHash) {
		return SMTPSettings{}, ErrSecurityKeyRequired
	}
	host := strings.TrimSpace(request.Host)
	if host == "" || strings.ContainsAny(host, "\r\n") {
		return SMTPSettings{}, errors.New("SMTP Host 不能为空")
	}
	port := request.Port
	if port <= 0 || port > 65535 {
		return SMTPSettings{}, errors.New("SMTP 端口无效")
	}
	from, err := normalizeEmail(request.From)
	if err != nil {
		return SMTPSettings{}, fmt.Errorf("发件邮箱无效: %w", err)
	}
	mode := strings.ToLower(strings.TrimSpace(request.TLSMode))
	if mode == "" {
		mode = "starttls"
	}
	if mode != "starttls" && mode != "tls" && mode != "none" {
		return SMTPSettings{}, errors.New("SMTP TLS 模式只允许 starttls、tls 或 none")
	}
	if mode == "none" && !isLoopbackHost(host) {
		return SMTPSettings{}, errors.New("非本机 SMTP 禁止明文 none 模式，请使用 starttls 或 tls")
	}
	password := request.Password
	if strings.TrimSpace(password) == "" {
		if current, loadErr := s.loadSMTPConfig(); loadErr == nil {
			password = current.Password
		}
	}
	cfg := smtpSecretConfig{Host: host, Port: port, Username: strings.TrimSpace(request.Username), Password: password, From: from, TLSMode: mode}
	data, err := json.Marshal(cfg)
	if err != nil {
		return SMTPSettings{}, err
	}
	if err := s.vault.Put(smtpConfigSecretName, string(data)); err != nil {
		return SMTPSettings{}, fmt.Errorf("保存 SMTP 安全配置失败: %w", err)
	}
	return smtpSettingsView(cfg), nil
}

func (s *Service) BindMyEmail(token string, request BindEmailRequest) (EmailSecurityStatus, error) {
	user, err := s.Validate(token)
	if err != nil {
		return EmailSecurityStatus{}, err
	}
	email, err := normalizeEmail(request.Email)
	if err != nil {
		return EmailSecurityStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return EmailSecurityStatus{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	if !verifyPassword(request.Password, *a) {
		return EmailSecurityStatus{}, ErrInvalidLogin
	}
	if a.Email == email && a.EmailVerifiedAt != 0 {
		return s.emailSecurityStatusLocked(token, *a), nil
	}
	previous := a.User
	a.Email = email
	a.EmailVerifiedAt = 0
	a.RiskUpdatedAt = s.now().Unix()
	if err := s.saveLocked(); err != nil {
		a.User = previous
		return EmailSecurityStatus{}, err
	}
	s.refreshUserSessionsLocked(a.User, a.AuthEpoch, true)
	return s.emailSecurityStatusLocked(token, *a), nil
}
func (s *Service) UnbindMyEmail(token, password string) (EmailSecurityStatus, error) {
	user, err := s.Validate(token)
	if err != nil {
		return EmailSecurityStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return EmailSecurityStatus{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	if !verifyPassword(password, *a) {
		return EmailSecurityStatus{}, ErrInvalidLogin
	}
	a.Email = ""
	a.EmailVerifiedAt = 0
	if err := s.saveLocked(); err != nil {
		return EmailSecurityStatus{}, err
	}
	s.refreshUserSessionsLocked(a.User, a.AuthEpoch, true)
	return s.emailSecurityStatusLocked(token, *a), nil
}
func (s *Service) MyEmailSecurityStatus(token string) (EmailSecurityStatus, error) {
	user, err := s.Validate(token)
	if err != nil {
		return EmailSecurityStatus{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return EmailSecurityStatus{}, ErrUnauthorized
	}
	return s.emailSecurityStatusLocked(token, s.accounts[idx]), nil
}
func (s *Service) RequestEmailVerification(token string, request RequestEmailVerificationRequest) error {
	user, err := s.Validate(token)
	if err != nil {
		return err
	}
	purpose := strings.ToLower(strings.TrimSpace(request.Purpose))
	if purpose == "" {
		purpose = "bind"
	}
	if purpose != "bind" && purpose != "risk" {
		return errors.New("邮箱验证 purpose 只允许 bind 或 risk")
	}
	s.mu.RLock()
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		s.mu.RUnlock()
		return ErrUnauthorized
	}
	a := s.accounts[idx]
	s.mu.RUnlock()
	if a.Email == "" {
		return errors.New("当前账号未绑定邮箱")
	}
	if purpose == "risk" && a.EmailVerifiedAt == 0 {
		return errors.New("当前邮箱尚未验证，不能用于风险二次验证")
	}
	cfg, err := s.loadSMTPConfig()
	if err != nil {
		return err
	}
	code, err := randomNumericCode(8)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(user.ID + "|" + purpose + "|" + code))
	s.mu.Lock()
	s.emailChallenges[user.ID] = emailChallenge{Purpose: purpose, Hash: hash, ExpiresAt: s.now().Add(emailChallengeLifetime)}
	s.mu.Unlock()
	subject := "AI Game Manager Panel 邮箱验证"
	body := "你的 AGMP 邮箱验证码为：" + code + "\n\n验证码 10 分钟内有效。若不是你本人操作，请立即联系超级管理员并检查账号安全。"
	if purpose == "risk" {
		subject = "AI Game Manager Panel 高风险操作二次验证"
		body = "检测到高风险操作，需要确认是你本人。验证码：" + code + "\n\n验证码 10 分钟内有效。未完成验证前，AGMP 将继续阻止该高风险操作。"
	}
	if err := sendSMTP(cfg, a.Email, subject, body); err != nil {
		s.mu.Lock()
		delete(s.emailChallenges, user.ID)
		s.mu.Unlock()
		return fmt.Errorf("发送验证邮件失败: %w", err)
	}
	return nil
}
func (s *Service) ConfirmEmailVerification(token string, request ConfirmEmailVerificationRequest) (EmailSecurityStatus, error) {
	user, err := s.Validate(token)
	if err != nil {
		return EmailSecurityStatus{}, err
	}
	code := strings.TrimSpace(request.Code)
	if len(code) != 8 {
		return EmailSecurityStatus{}, ErrInvalidEmailCode
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.emailChallenges[user.ID]
	if !ok || !challenge.ExpiresAt.After(s.now()) || challenge.Attempts >= emailChallengeMaxAttempts {
		delete(s.emailChallenges, user.ID)
		return EmailSecurityStatus{}, ErrInvalidEmailCode
	}
	challenge.Attempts++
	s.emailChallenges[user.ID] = challenge
	actual := sha256.Sum256([]byte(user.ID + "|" + challenge.Purpose + "|" + code))
	if subtle.ConstantTimeCompare(actual[:], challenge.Hash[:]) != 1 {
		if challenge.Attempts >= emailChallengeMaxAttempts {
			delete(s.emailChallenges, user.ID)
		}
		return EmailSecurityStatus{}, ErrInvalidEmailCode
	}
	idx := s.accountIndexLocked(user.ID)
	if idx < 0 {
		return EmailSecurityStatus{}, ErrUnauthorized
	}
	a := &s.accounts[idx]
	now := s.now()
	if challenge.Purpose == "bind" {
		a.EmailVerifiedAt = now.Unix()
	} else if challenge.Purpose == "risk" {
		a.RiskState = RiskStateNormal
		a.RiskReason = ""
		a.RiskUpdatedAt = now.Unix()
	}
	if err := s.saveLocked(); err != nil {
		return EmailSecurityStatus{}, err
	}
	delete(s.emailChallenges, user.ID)
	for key, session := range s.sessions {
		if session.User.ID != user.ID {
			continue
		}
		session.User = a.User
		if challenge.Purpose == "risk" {
			session.stepUpUntil = now.Add(stepUpLifetime)
		}
		s.sessions[key] = session
	}
	return s.emailSecurityStatusLocked(token, *a), nil
}

// RequestPasswordReset never reveals whether the username exists or has email.
// When email recovery is unavailable, the caller still receives the same generic
// response and may ask the Owner for an administrative reset workflow later.
func (s *Service) RequestPasswordReset(request RequestPasswordResetRequest) (PasswordResetRequestStatus, error) {
	generic := PasswordResetRequestStatus{Accepted: true, Message: "如果该账号已绑定并验证邮箱，密码重置验证码将发送到该邮箱。"}
	username := normalizeUsername(request.Username)
	if username == "" {
		return generic, nil
	}
	cfg, err := s.loadSMTPConfig()
	if err != nil {
		return generic, nil
	}
	s.mu.RLock()
	idx := -1
	for i := range s.accounts {
		if normalizeUsername(s.accounts[i].Username) == username {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.RUnlock()
		return generic, nil
	}
	a := s.accounts[idx]
	s.mu.RUnlock()
	if a.Email == "" || a.EmailVerifiedAt == 0 {
		return generic, nil
	}
	code, err := randomNumericCode(8)
	if err != nil {
		return generic, nil
	}
	hash := sha256.Sum256([]byte(a.ID + "|password-reset|" + code))
	s.mu.Lock()
	s.passwordResetChallenges[a.ID] = passwordResetChallenge{UserID: a.ID, Hash: hash, ExpiresAt: s.now().Add(passwordResetLifetime)}
	s.mu.Unlock()
	if err := sendSMTP(cfg, a.Email, "AI Game Manager Panel 密码找回", "你的密码重置验证码为："+code+"\n\n验证码 15 分钟内有效。若不是你本人操作，请忽略此邮件并联系管理员。"); err != nil {
		s.mu.Lock()
		delete(s.passwordResetChallenges, a.ID)
		s.mu.Unlock()
	}
	return generic, nil
}
func (s *Service) ConfirmPasswordReset(request ConfirmPasswordResetRequest) error {
	if err := validatePassword(request.NewPassword); err != nil {
		return err
	}
	username := normalizeUsername(request.Username)
	code := strings.TrimSpace(request.Code)
	if username == "" || len(code) != 8 {
		return ErrInvalidEmailCode
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.accounts {
		if normalizeUsername(s.accounts[i].Username) == username {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrInvalidEmailCode
	}
	a := &s.accounts[idx]
	challenge, ok := s.passwordResetChallenges[a.ID]
	if !ok || !challenge.ExpiresAt.After(s.now()) || challenge.Attempts >= emailChallengeMaxAttempts {
		delete(s.passwordResetChallenges, a.ID)
		return ErrInvalidEmailCode
	}
	challenge.Attempts++
	s.passwordResetChallenges[a.ID] = challenge
	actual := sha256.Sum256([]byte(a.ID + "|password-reset|" + code))
	if subtle.ConstantTimeCompare(actual[:], challenge.Hash[:]) != 1 {
		if challenge.Attempts >= emailChallengeMaxAttempts {
			delete(s.passwordResetChallenges, a.ID)
		}
		return ErrInvalidEmailCode
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	a.PasswordSalt = encodeRaw(salt)
	a.PasswordHash = encodeRaw(derivePassword([]byte(request.NewPassword), salt, defaultIterations, 32))
	a.Iterations = defaultIterations
	a.AuthEpoch++
	a.RiskState = RiskStateNormal
	a.RiskReason = ""
	a.RiskUpdatedAt = s.now().Unix()
	if err := s.saveLocked(); err != nil {
		return err
	}
	delete(s.passwordResetChallenges, a.ID)
	s.invalidateUserSessionsLocked(a.ID)
	return nil
}

func (s *Service) emailSecurityStatusLocked(token string, a accountRecord) EmailSecurityStatus {
	status := EmailSecurityStatus{Email: a.Email, Bound: a.Email != "", Verified: a.EmailVerifiedAt != 0, VerifiedAt: a.EmailVerifiedAt, RiskState: a.RiskState, RiskReason: a.RiskReason, RiskUpdatedAt: a.RiskUpdatedAt, CoreAccess: a.CoreAccess, SMTPConfigured: s.vault.Exists(smtpConfigSecretName), EmailFeaturesAvailable: a.Email != "" && a.EmailVerifiedAt != 0 && s.vault.Exists(smtpConfigSecretName)}
	if session, ok := s.sessions[strings.TrimSpace(token)]; ok && session.stepUpUntil.After(s.now()) {
		status.StepUpVerified = true
		status.StepUpValidUntil = session.stepUpUntil.Unix()
	}
	return status
}
func (s *Service) recordLoginFailureLocked(accountIndex int) {
	if accountIndex < 0 || accountIndex >= len(s.accounts) {
		return
	}
	a := &s.accounts[accountIndex]
	now := s.now()
	cutoff := now.Add(-10 * time.Minute)
	items := s.loginFailures[a.ID][:0]
	for _, item := range s.loginFailures[a.ID] {
		if item.After(cutoff) {
			items = append(items, item)
		}
	}
	items = append(items, now)
	s.loginFailures[a.ID] = items
	if len(items) >= 10 {
		a.RiskState = RiskStateLocked
		a.RiskReason = "短时间内大量登录失败，账号已进入强风控"
		a.RiskUpdatedAt = now.Unix()
		_ = s.saveLocked()
		s.refreshUserSessionsLocked(a.User, a.AuthEpoch, true)
	} else if len(items) >= 5 && a.RiskState != RiskStateLocked {
		a.RiskState = RiskStateChallenge
		a.RiskReason = "短时间内多次登录失败"
		a.RiskUpdatedAt = now.Unix()
		_ = s.saveLocked()
		s.refreshUserSessionsLocked(a.User, a.AuthEpoch, true)
	}
}
func (s *Service) clearLoginFailuresLocked(userID string) { delete(s.loginFailures, userID) }
func (s *Service) loadSMTPConfig() (smtpSecretConfig, error) {
	if !s.vault.Exists(smtpConfigSecretName) {
		return smtpSecretConfig{}, ErrEmailDeliveryUnavailable
	}
	value, err := s.vault.Get(smtpConfigSecretName)
	if err != nil {
		return smtpSecretConfig{}, fmt.Errorf("读取 SMTP 安全配置失败: %w", err)
	}
	var cfg smtpSecretConfig
	if json.Unmarshal([]byte(value), &cfg) != nil || strings.TrimSpace(cfg.Host) == "" || cfg.Port <= 0 || strings.TrimSpace(cfg.From) == "" {
		return smtpSecretConfig{}, errors.New("SMTP 安全配置损坏")
	}
	return cfg, nil
}
func smtpSettingsView(cfg smtpSecretConfig) SMTPSettings {
	return SMTPSettings{Configured: cfg.Host != "", Host: cfg.Host, Port: cfg.Port, Username: cfg.Username, From: cfg.From, TLSMode: cfg.TLSMode, HasPassword: cfg.Password != ""}
}
func randomNumericCode(digits int) (string, error) {
	if digits < 6 || digits > 10 {
		return "", errors.New("invalid code length")
	}
	buf := make([]byte, digits)
	for i := range buf {
		for {
			var b [1]byte
			if _, err := rand.Read(b[:]); err != nil {
				return "", err
			}
			if b[0] >= 250 {
				continue
			}
			buf[i] = '0' + (b[0] % 10)
			break
		}
	}
	return string(buf), nil
}
func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
func sendSMTP(cfg smtpSecretConfig, to, subject, body string) error {
	to, err := normalizeEmail(to)
	if err != nil {
		return err
	}
	if strings.ContainsAny(subject, "\r\n") {
		return errors.New("邮件主题无效")
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	serverName := strings.Trim(cfg.Host, "[]")
	var client *smtp.Client
	if cfg.TLSMode == "tls" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, serverName)
		if err != nil {
			_ = conn.Close()
			return err
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return err
		}
		if cfg.TLSMode == "starttls" {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				_ = client.Close()
				return errors.New("SMTP 服务不支持 STARTTLS")
			}
			if err := client.StartTLS(&tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12}); err != nil {
				_ = client.Close()
				return err
			}
		}
	}
	defer client.Close()
	if cfg.Username != "" {
		if cfg.TLSMode == "none" && !isLoopbackHost(cfg.Host) {
			return errors.New("禁止在非本机明文 SMTP 上发送认证凭据")
		}
		if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, serverName)); err != nil {
			return err
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := "From: " + cfg.From + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(body, "\n", "\r\n")
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
func encodeRaw(value []byte) string { return base64.RawStdEncoding.EncodeToString(value) }

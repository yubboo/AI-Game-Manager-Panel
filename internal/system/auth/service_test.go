package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func tamperSignedToken(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token: %q", token)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) == 0 {
		t.Fatalf("decode signature: %v", err)
	}
	sig[0] ^= 0x80
	parts[2] = base64.RawURLEncoding.EncodeToString(sig)
	return strings.Join(parts, ".")
}

func ownerWithStepUp(t *testing.T, s *Service) Session {
	t.Helper()
	owner, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	return owner
}

func registerMember(t *testing.T, s *Service, owner Session, username, email string) Session {
	t.Helper()
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	invite, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleOperator, TargetUsername: username})
	if err != nil {
		t.Fatal(err)
	}
	member, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: invite.Token, Username: username, Email: email, Password: "password456"})
	if err != nil {
		t.Fatal(err)
	}
	return member
}

func authorizeMember(t *testing.T, s *Service, owner Session, member Session) User {
	t.Helper()
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	grant, err := s.CreateMemberAuthorization(owner.Token, CreateMemberAuthorizationRequest{UserID: member.User.ID})
	if err != nil {
		t.Fatal(err)
	}
	user, err := s.RedeemMemberAuthorization(member.Token, RedeemMemberAuthorizationRequest{Token: grant.Token})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func TestBootstrapOwnerIsOneTimeAndCreatesOrganization(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	initial := s.BootstrapStatus()
	if initial.OwnerExists || !initial.RegistrationOpen || initial.InstanceFingerprint == "" {
		t.Fatalf("bad initial status: %+v", initial)
	}
	owner, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", DisplayName: "Boss", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if owner.User.Role != RoleOwner || owner.User.OrganizationID == "" || owner.User.GroupID != DefaultManagementGroup {
		t.Fatalf("bad owner: %+v", owner.User)
	}
	if owner.User.CoreAccess != CoreAccessAuthorized {
		t.Fatalf("owner core access=%s", owner.User.CoreAccess)
	}
	if _, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "other", Password: "password123"}); !errors.Is(err, ErrBootstrapClosed) {
		t.Fatalf("bootstrap reopened: %v", err)
	}
	status := s.BootstrapStatus()
	if !status.OwnerExists || status.RegistrationOpen || !status.InvitationRegistrationOnly || status.UserCount != 0 {
		t.Fatalf("bad closed status: %+v", status)
	}
}

func TestPublicDirectMemberCreationIsDisabled(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	if _, err := s.CreateUser(owner.Token, CreateUserRequest{Username: "member", Password: "password456", Role: RoleOperator}); !errors.Is(err, ErrInvitationRequired) {
		t.Fatalf("direct creation not blocked: %v", err)
	}
}

func TestSignedInvitationBoundToInstanceAndOneTime(t *testing.T) {
	root := t.TempDir()
	s := New(filepath.Join(root, "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	invite, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleOperator, TargetUsername: "member", ExpiresInHours: 24})
	if err != nil {
		t.Fatal(err)
	}
	if invite.Token == "" || invite.VerificationCode == "" || invite.InstanceFingerprint != s.InstanceIdentity().Fingerprint {
		t.Fatalf("bad invite: %+v", invite)
	}
	if _, err := s.InspectInvitation(tamperSignedToken(t, invite.Token)); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("forged invite accepted: %v", err)
	}
	other := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	if _, err := other.InspectInvitation(invite.Token); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("cross-instance invite accepted: %v", err)
	}
	member, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: invite.Token, Username: "member", Password: "password456"})
	if err != nil {
		t.Fatal(err)
	}
	if member.User.OrganizationID != owner.User.OrganizationID || member.User.CoreAccess != CoreAccessPending {
		t.Fatalf("bad member: %+v", member.User)
	}
	if _, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: invite.Token, Username: "member2", Password: "password456"}); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("reused invite accepted: %v", err)
	}
}

func TestInvitationOptionalEmailAndOptionalTargetEmail(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	// No email: registration still succeeds.
	invite, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleOperator, TargetUsername: "noemail"})
	if err != nil {
		t.Fatal(err)
	}
	member, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: invite.Token, Username: "noemail", Password: "password456"})
	if err != nil {
		t.Fatalf("email became mandatory: %v", err)
	}
	if member.User.Email != "" {
		t.Fatalf("unexpected email: %q", member.User.Email)
	}
	// If an administrator explicitly binds the invitation to an email, that one
	// invite must match, but the product still has no global email requirement.
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	bound, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleOperator, TargetUsername: "bound", TargetEmail: "employee@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: bound.Token, Username: "bound", Email: "wrong@example.test", Password: "password456"}); err == nil {
		t.Fatal("email-bound invite accepted wrong email")
	}
	if _, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: bound.Token, Username: "bound", Email: "employee@example.test", Password: "password456"}); err != nil {
		t.Fatal(err)
	}
}

func TestMemberCoreAuthorizationDoesNotRequireEmail(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	member := registerMember(t, s, owner, "member", "")
	if _, err := s.RequireCoreAccess(member.Token); !errors.Is(err, ErrCoreAuthorizationRequired) {
		t.Fatalf("pending member core access=%v", err)
	}
	user := authorizeMember(t, s, owner, member)
	if user.CoreAccess != CoreAccessAuthorized || s.CoreSeatCount() != 2 {
		t.Fatalf("core auth failed user=%+v seats=%d", user, s.CoreSeatCount())
	}
}

func TestMemberAuthorizationForgeryReplayAndWrongUserAreRejected(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	a := registerMember(t, s, owner, "usera", "")
	b := registerMember(t, s, owner, "userb", "")
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	grant, err := s.CreateMemberAuthorization(owner.Token, CreateMemberAuthorizationRequest{UserID: a.User.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemMemberAuthorization(a.Token, RedeemMemberAuthorizationRequest{Token: tamperSignedToken(t, grant.Token)}); err == nil {
		t.Fatal("forged grant accepted")
	}
	if _, err := s.RedeemMemberAuthorization(b.Token, RedeemMemberAuthorizationRequest{Token: grant.Token}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong user accepted: %v", err)
	}
	if _, err := s.RedeemMemberAuthorization(a.Token, RedeemMemberAuthorizationRequest{Token: grant.Token}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemMemberAuthorization(a.Token, RedeemMemberAuthorizationRequest{Token: grant.Token}); err == nil {
		t.Fatal("grant replay accepted")
	}
}

func TestInvitationConcurrentConsumptionOnlyOneWins(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	invite, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleOperator})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: invite.Token, Username: fmt.Sprintf("member%d", i), Password: "password456"})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	success, failed := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrInvalidInvitation) {
			failed++
		} else {
			t.Fatalf("unexpected race error: %v", err)
		}
	}
	if success != 1 || failed != 1 {
		t.Fatalf("race success=%d failed=%d", success, failed)
	}
}

func TestSessionInvalidatesImmediatelyWhenMemberRemoved(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	member := registerMember(t, s, owner, "member", "")
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveMember(owner.Token, member.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Validate(member.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("removed member session survived: %v", err)
	}
}

func TestRiskStepUpUsesCredentialsWhenEmailUnavailable(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequireSensitiveAction(owner.Token, "danger"); !errors.Is(err, ErrCredentialStepUpRequired) {
		t.Fatalf("no-email stepup=%v", err)
	}
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequireSensitiveAction(owner.Token, "danger"); err != nil {
		t.Fatalf("credential stepup did not authorize: %v", err)
	}
}

func TestRiskStepUpRequiresSecurityKeyWhenConfigured(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	key, err := GenerateSecurityKey()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123", SecurityKey: key.Key})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123"}); !errors.Is(err, ErrSecurityKeyRequired) {
		t.Fatalf("configured key not required for step-up: %v", err)
	}
	if _, err := s.ConfirmCredentialStepUp(owner.Token, CredentialStepUpRequest{Password: "password123", SecurityKey: key.Key}); err != nil {
		t.Fatal(err)
	}
}

func TestLoginFailuresEnterRiskLock(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	s.Logout(owner.Token)
	for i := 0; i < 10; i++ {
		_, _ = s.Login(LoginRequest{Username: "owner", Password: "wrong-password"})
	}
	login, err := s.Login(LoginRequest{Username: "owner", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.MyEmailSecurityStatus(login.Token)
	if err != nil {
		t.Fatal(err)
	}
	if status.RiskState != RiskStateLocked {
		t.Fatalf("risk=%s", status.RiskState)
	}
	if _, err := s.RequireSensitiveAction(login.Token, "danger"); !errors.Is(err, ErrRiskLocked) {
		t.Fatalf("locked risk action=%v", err)
	}
}

func TestOptionalEmailBindingDoesNotChangeCoreAccess(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	member := registerMember(t, s, owner, "member", "")
	authorizeMember(t, s, owner, member)
	status, err := s.BindMyEmail(member.Token, BindEmailRequest{Email: "member@example.test", Password: "password456"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Bound || status.Verified || status.CoreAccess != CoreAccessAuthorized {
		t.Fatalf("bad email bind status: %+v", status)
	}
	if _, err := s.RequireCoreAccess(member.Token); err != nil {
		t.Fatalf("binding unverified email blocked core: %v", err)
	}
	status, err = s.UnbindMyEmail(member.Token, "password456")
	if err != nil {
		t.Fatal(err)
	}
	if status.Bound {
		t.Fatal("email did not unbind")
	}
	if _, err := s.RequireCoreAccess(member.Token); err != nil {
		t.Fatalf("unbinding email blocked core: %v", err)
	}
}

func TestInstanceIdentityStableAndSignedStoreTamperFailsClosed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "auth", "accounts.json")
	s := New(path)
	id1 := s.InstanceIdentity()
	if _, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	reloaded := New(path)
	id2 := reloaded.InstanceIdentity()
	if id1.ID != id2.ID || id1.Fingerprint != id2.Fingerprint {
		t.Fatalf("identity changed: %+v %+v", id1, id2)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("\"owner\""), []byte("\"hacker\""), 1)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	broken := New(path)
	status := broken.BootstrapStatus()
	if status.StoreError == "" || status.RegistrationOpen {
		t.Fatalf("tamper not fail-closed: %+v", status)
	}
}

func TestBootstrapVaultMarkerSurvivesAccountAndLockDeletion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "auth", "accounts.json")
	s := New(path)
	if _, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
	_ = os.Remove(filepath.Join(filepath.Dir(path), "bootstrap.lock"))
	reloaded := New(path)
	status := reloaded.BootstrapStatus()
	if status.RegistrationOpen || !status.BootstrapLocked {
		t.Fatalf("bootstrap reopened: %+v", status)
	}
}

func TestLegacyUnsignedStoreMigratesOnceThenDowngradeFailsClosed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "auth", "accounts.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	legacyOwner, err := newAccount("owner", "Boss", "password123", RoleOwner, time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	legacy := accountFile{Version: 2, Accounts: []accountRecord{legacyOwner}}
	raw, _ := json.MarshalIndent(legacy, "", "  ")
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "bootstrap.lock"), []byte("closed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	migrated := New(path)
	if st := migrated.BootstrapStatus(); st.StoreError != "" || !st.OwnerExists {
		t.Fatalf("migration failed: %+v", st)
	}
	data, _ := os.ReadFile(path)
	var signed accountFile
	if err := json.Unmarshal(data, &signed); err != nil {
		t.Fatal(err)
	}
	if signed.Version != 3 || signed.Signature == "" {
		t.Fatalf("not signed v3: %+v", signed)
	}
	signed.Version = 2
	signed.Signature = ""
	signed.Accounts[0].Username = "attacker"
	down, _ := json.MarshalIndent(signed, "", "  ")
	_ = os.WriteFile(path, append(down, '\n'), 0o600)
	locked := New(path)
	if st := locked.BootstrapStatus(); st.StoreError == "" || st.RegistrationOpen {
		t.Fatalf("downgrade accepted: %+v", st)
	}
}

func TestSecurityKeyPlaintextNeverPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth", "accounts.json")
	s := New(path)
	key, err := GenerateSecurityKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateInitialOwner(CreateOwnerRequest{Username: "owner", Password: "password123", SecurityKey: key.Key}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte(key.Key)) {
		t.Fatal("security key plaintext persisted")
	}
	if !bytes.Contains(data, []byte("securityKeyHash")) {
		t.Fatal("security key hash missing")
	}
}

func TestEmailChallengeCanBeConfirmedWithoutBecomingCorePrerequisite(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	member := registerMember(t, s, owner, "member", "member@example.test")
	authorizeMember(t, s, owner, member)
	hash := sha256.Sum256([]byte(member.User.ID + "|bind|31415926"))
	s.mu.Lock()
	s.emailChallenges[member.User.ID] = emailChallenge{Purpose: "bind", Hash: hash, ExpiresAt: s.now().Add(time.Minute)}
	s.mu.Unlock()
	status, err := s.ConfirmEmailVerification(member.Token, ConfirmEmailVerificationRequest{Code: "31415926"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Verified {
		t.Fatal("email not verified")
	}
	if _, err := s.RequireCoreAccess(member.Token); err != nil {
		t.Fatalf("email flow changed core entitlement: %v", err)
	}
}

func TestEmailFeaturesAreOptionalAndDependOnAdminSMTPConfiguration(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	status, err := s.MyEmailSecurityStatus(owner.Token)
	if err != nil {
		t.Fatal(err)
	}
	if status.SMTPConfigured || status.EmailFeaturesAvailable {
		t.Fatalf("fresh instance unexpectedly reports email features available: %+v", status)
	}
	if _, err := s.RequireCoreAccess(owner.Token); err != nil {
		t.Fatalf("missing SMTP must not block AGMP core access: %v", err)
	}
	if err := s.RequestEmailVerification(owner.Token, RequestEmailVerificationRequest{Purpose: "bind"}); !errors.Is(err, ErrEmailDeliveryUnavailable) && !strings.Contains(err.Error(), "未绑定邮箱") {
		// No mailbox is bound yet; either condition is an acceptable earlier guard,
		// but neither may mutate core entitlement.
		t.Fatalf("unexpected email feature error without SMTP: %v", err)
	}
	if _, err := s.SaveSMTPSettings(owner.Token, SaveSMTPSettingsRequest{Host: "127.0.0.1", Port: 2525, From: "agmp@example.test", TLSMode: "none", AccountPassword: "password123"}); err != nil {
		t.Fatalf("Owner should be able to configure local SMTP without first binding email: %v", err)
	}
	settings, err := s.SMTPSettings(owner.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Configured || settings.Host != "127.0.0.1" || settings.Port != 2525 {
		t.Fatalf("SMTP settings not persisted: %+v", settings)
	}
	status, err = s.MyEmailSecurityStatus(owner.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !status.SMTPConfigured || status.EmailFeaturesAvailable {
		t.Fatalf("SMTP should be globally configured while account-specific email features remain unavailable until the user binds/verifies an address: %+v", status)
	}
}

func TestSMTPConfigurationRequiresAdministratorAndNeverReturnsPassword(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "auth", "accounts.json"))
	owner := ownerWithStepUp(t, s)
	invite, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleOperator, TargetUsername: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	operator, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: invite.Token, Username: "operator", Password: "password456"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveSMTPSettings(operator.Token, SaveSMTPSettingsRequest{Host: "127.0.0.1", Port: 2525, From: "agmp@example.test", TLSMode: "none", AccountPassword: "password456"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("operator unexpectedly configured SMTP: %v", err)
	}
	adminInvite, err := s.CreateInvitation(owner.Token, CreateInvitationRequest{Role: RoleAdministrator, TargetUsername: "mail-admin"})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := s.RegisterWithInvitation(RegisterInvitationRequest{InvitationToken: adminInvite.Token, Username: "mail-admin", Password: "admin-password-456"})
	if err != nil {
		t.Fatal(err)
	}
	// Email delivery is an instance-level security setting, not a core feature:
	// an invited administrator may configure it before any member-core grant.
	if _, err := s.SaveSMTPSettings(admin.Token, SaveSMTPSettingsRequest{Host: "127.0.0.1", Port: 2525, From: "agmp@example.test", TLSMode: "none", AccountPassword: "admin-password-456"}); err != nil {
		t.Fatalf("pending administrator should be allowed to configure SMTP: %v", err)
	}
	settings, err := s.SaveSMTPSettings(owner.Token, SaveSMTPSettingsRequest{Host: "127.0.0.1", Port: 2525, Username: "mailer", Password: "smtp-secret", From: "agmp@example.test", TLSMode: "none", AccountPassword: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if !settings.HasPassword {
		t.Fatal("SMTP view should report presence of password without exposing it")
	}
	serialized, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(serialized, []byte("smtp-secret")) {
		t.Fatal("SMTP secret leaked through settings view")
	}
}

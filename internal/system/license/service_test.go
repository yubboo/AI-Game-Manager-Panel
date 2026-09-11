//go:build !agmp_dev_license

package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testService(t *testing.T) (*Service, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	s := New(Config{Enabled: true, Product: "AI Game Manager Panel", PublicKey: base64.StdEncoding.EncodeToString(pub)}, filepath.Join(root, "license", "activation.json"))
	return s, priv
}

func TestCertificateActivationAndFeatureEntitlements(t *testing.T) {
	s, priv := testService(t)
	status := s.Status()
	if status.Valid || status.State != StateUnlicensed {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	payload := CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "pro",
		Features: []string{"server.basic", "ai.workbench"}, SeatLimit: 3,
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(48 * time.Hour).Unix(),
	}
	certificate, err := IssueCertificate(priv, payload)
	if err != nil {
		t.Fatal(err)
	}
	activated, err := s.Activate(ActivateRequest{Certificate: certificate})
	if err != nil {
		t.Fatal(err)
	}
	if !activated.Valid || activated.State != StateExpiring || activated.LicenseID != payload.LicenseID || activated.ActivationID != payload.ActivationID {
		t.Fatalf("unexpected activated status: %+v", activated)
	}
	if !s.HasFeature("ai.workbench").Allowed {
		t.Fatal("expected ai.workbench entitlement")
	}
	if s.HasFeature("plugin.extensions").Allowed {
		t.Fatal("unexpected plugin.extensions entitlement")
	}
}

func TestCertificateRejectsWrongMachineAndInstallID(t *testing.T) {
	s, priv := testService(t)
	status := s.Status()
	base := CertificatePayload{LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel", MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix()}

	wrongMachine := base
	wrongMachine.MachineCode = "BFM-00000000-00000000-00000000-00000000"
	cert, err := IssueCertificate(priv, wrongMachine)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Activate(ActivateRequest{Certificate: cert}); err == nil || !strings.Contains(err.Error(), "机器码") {
		t.Fatalf("expected machine mismatch, got %v", err)
	}

	wrongInstall := base
	wrongInstall.ActivationID = NewTimeOrderedID("ACT")
	wrongInstall.InstallID = "BFID-00000000-00000000-00000000"
	cert, err = IssueCertificate(priv, wrongInstall)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Activate(ActivateRequest{Certificate: cert}); err == nil || !strings.Contains(err.Error(), "安装 ID") {
		t.Fatalf("expected install mismatch, got %v", err)
	}
}

func TestCertificateRejectsTamperedSignature(t *testing.T) {
	s, priv := testService(t)
	status := s.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(cert, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected certificate format: %q", cert)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) == 0 {
		t.Fatalf("decode signature: %v", err)
	}
	sig[0] ^= 0x01
	parts[2] = base64.RawURLEncoding.EncodeToString(sig)
	_, err = s.Activate(ActivateRequest{Certificate: strings.Join(parts, ".")})
	if err == nil {
		t.Fatal("tampered certificate signature must be rejected")
	}
}

func TestExpiredCertificateAndUnbind(t *testing.T) {
	s, priv := testService(t)
	status := s.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel", MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Add(-2 * time.Hour).Unix(), ExpiresAt: time.Now().Add(-time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Activate(ActivateRequest{Certificate: cert}); err == nil || !strings.Contains(err.Error(), "过期") {
		t.Fatalf("expected expired certificate, got %v", err)
	}

	valid, err := IssueCertificate(priv, CertificatePayload{LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel", MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Activate(ActivateRequest{Certificate: valid}); err != nil {
		t.Fatal(err)
	}
	unbound, err := s.Unbind()
	if err != nil {
		t.Fatal(err)
	}
	if unbound.Valid || unbound.Activated || unbound.State != StateUnlicensed {
		t.Fatalf("unexpected unbound status: %+v", unbound)
	}
}

func TestShortActivationCodeRequiresOnlineServer(t *testing.T) {
	s, _ := testService(t)
	_, err := s.Activate(ActivateRequest{ActivationCode: "BF-7K2P-M8QX-4N6R-W9TY"})
	if err == nil || !strings.Contains(err.Error(), "在线") {
		t.Fatalf("expected online activation unavailable, got %v", err)
	}
}

func TestInstallIDPersists(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	root := t.TempDir()
	path := filepath.Join(root, "license", "activation.json")
	cfg := Config{Enabled: true, Product: "AI Game Manager Panel", PublicKey: base64.StdEncoding.EncodeToString(pub)}
	first := New(cfg, path).Status().InstallID
	second := New(cfg, path).Status().InstallID
	if first == "" || first != second {
		t.Fatalf("install id should persist: %q vs %q", first, second)
	}
	if _, err := os.Stat(filepath.Join(root, "license", "install.id")); err != nil {
		t.Fatal(err)
	}
}

func TestTimeOrderedIDsCarryPrefixAndDiffer(t *testing.T) {
	a := NewTimeOrderedID("LIC")
	b := NewTimeOrderedID("LIC")
	if !strings.HasPrefix(a, "LIC-") || a == b {
		t.Fatalf("unexpected ids: %q %q", a, b)
	}
}

func TestRepeatedStatusKeepsStableDeviceIdentifiers(t *testing.T) {
	s, _ := testService(t)
	first := s.Status()
	for i := 0; i < 5; i++ {
		next := s.Status()
		if next.MachineCode != first.MachineCode {
			t.Fatalf("machine code changed during same core process: %q -> %q", first.MachineCode, next.MachineCode)
		}
		if next.InstallID != first.InstallID {
			t.Fatalf("install id changed during same service lifecycle: %q -> %q", first.InstallID, next.InstallID)
		}
	}
}

func TestCertificateCarriesIssuerKeyID(t *testing.T) {
	s, priv := testService(t)
	status := s.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := s.Activate(ActivateRequest{Certificate: cert})
	if err != nil {
		t.Fatal(err)
	}
	expected := PublicKeyID(priv.Public().(ed25519.PublicKey))
	if activated.IssuerKeyID != expected {
		t.Fatalf("issuer key id mismatch: got %q want %q", activated.IssuerKeyID, expected)
	}
}

func TestRetiredTrustedKeyStillVerifiesHistoricalCertificate(t *testing.T) {
	oldPub, oldPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	newPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	s := New(Config{
		Enabled:           true,
		Product:           "AI Game Manager Panel",
		PublicKey:         base64.StdEncoding.EncodeToString(newPub),
		TrustedPublicKeys: []string{base64.StdEncoding.EncodeToString(oldPub), base64.StdEncoding.EncodeToString(newPub)},
	}, filepath.Join(root, "license", "activation.json"))
	status := s.Status()
	cert, err := IssueCertificate(oldPriv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := s.Activate(ActivateRequest{Certificate: cert})
	if err != nil {
		t.Fatalf("historical certificate signed by retired trusted key must remain valid: %v", err)
	}
	if !activated.Valid || activated.IssuerKeyID != PublicKeyID(oldPub) {
		t.Fatalf("unexpected historical activation status: %+v", activated)
	}
}

func TestPublicKeyIDAndFingerprintAreStable(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if got := PublicKeyID(pub); !strings.HasPrefix(got, "AGMP-KID-") || got != PublicKeyID(pub) {
		t.Fatalf("unexpected key id: %q", got)
	}
	if got := PublicKeyFingerprint(pub); !strings.HasPrefix(got, "SHA256:") || got != PublicKeyFingerprint(pub) {
		t.Fatalf("unexpected fingerprint: %q", got)
	}
}

func TestCertificateAcceptsLegacyIssuerKeyID(t *testing.T) {
	service, priv := testService(t)
	status := service.Status()
	pub := priv.Public().(ed25519.PublicKey)
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix(),
		IssuerKeyID: legacyPublicKeyID(pub),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.VerifyCertificateForDevice(cert, status.MachineCode, status.InstallID); err != nil {
		t.Fatalf("legacy issuer key id must remain valid after AGMP naming migration: %v", err)
	}
}

func TestCertificateRejectsSpoofedIssuerKeyID(t *testing.T) {
	s, priv := testService(t)
	status := s.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "standard", IssuedAt: time.Now().Unix(),
		IssuerKeyID: "AGMP-KID-00000000-00000000-00000000-00000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Activate(ActivateRequest{Certificate: cert}); err == nil {
		t.Fatal("certificate whose issuerKeyId does not match the signing public key must be rejected")
	}
}

func TestActivationPersistsAcrossServiceRestart(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	recordPath := filepath.Join(root, "license", "activation.json")
	cfg := Config{Enabled: true, Product: "AI Game Manager Panel", PublicKey: base64.StdEncoding.EncodeToString(pub)}
	first := New(cfg, recordPath)
	status := first.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "pro", IssuedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := first.Activate(ActivateRequest{Certificate: cert})
	if err != nil {
		t.Fatal(err)
	}
	if !activated.Valid {
		t.Fatalf("expected first service to be activated: %+v", activated)
	}

	second := New(cfg, recordPath)
	reloaded := second.Status()
	if !reloaded.Valid || reloaded.LicenseID != activated.LicenseID || reloaded.ActivationID != activated.ActivationID {
		t.Fatalf("license must survive service restart: first=%+v second=%+v", activated, reloaded)
	}
	if reloaded.MachineCode != activated.MachineCode || reloaded.InstallID != activated.InstallID {
		t.Fatalf("device identifiers changed after service restart: first=%+v second=%+v", activated, reloaded)
	}
}

func TestMachineCodeDerivationProtocolIsStable(t *testing.T) {
	const raw = "AI" + "GMP-MACHINE-STABILITY-TEST"
	const want = "BFM-32A1F5AC-EFCD03DF-C1BC2B7C-F13407DE"
	if got := deriveMachineCode(raw); got != want {
		t.Fatalf("machine-code derivation protocol changed: got %q want %q; changing this value would invalidate existing device-bound licenses", got, want)
	}
}

func TestRequireFeatureBlocksUnlicensedAndAllowsLicensed(t *testing.T) {
	s, priv := testService(t)
	if err := s.RequireFeature("ai.workbench"); err == nil || !errors.Is(err, ErrFeatureNotEntitled) {
		t.Fatalf("unlicensed feature must be blocked, got %v", err)
	}
	status := s.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: status.MachineCode, InstallID: status.InstallID, Edition: "pro", IssuedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Activate(ActivateRequest{Certificate: cert}); err != nil {
		t.Fatal(err)
	}
	for _, feature := range ProFeatures {
		if err := s.RequireFeature(feature); err != nil {
			t.Fatalf("pro feature %s should be allowed: %v", feature, err)
		}
	}
}

func TestStandardAndProFeatureMatrix(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edition string
		allowed []string
		blocked []string
	}{
		{name: "standard", edition: "standard", allowed: StandardFeatures, blocked: []string{"server.multi", "backup.advanced", "remote.agent", "web.remote", "automation", "ai.workbench", "plugin.extensions"}},
		{name: "pro", edition: "pro", allowed: ProFeatures},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, priv := testService(t)
			st := s.Status()
			cert, err := IssueCertificate(priv, CertificatePayload{
				LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
				MachineCode: st.MachineCode, InstallID: st.InstallID, Edition: tc.edition, IssuedAt: time.Now().Unix(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Activate(ActivateRequest{Certificate: cert}); err != nil {
				t.Fatal(err)
			}
			for _, feature := range tc.allowed {
				if err := s.RequireFeature(feature); err != nil {
					t.Fatalf("%s must allow %s: %v", tc.edition, feature, err)
				}
			}
			for _, feature := range tc.blocked {
				if err := s.RequireFeature(feature); err == nil {
					t.Fatalf("%s must block %s", tc.edition, feature)
				}
			}
		})
	}
}

func TestExpiredAndUnboundLicenseBlockFeatures(t *testing.T) {
	s, priv := testService(t)
	st := s.Status()
	cert, err := IssueCertificate(priv, CertificatePayload{
		LicenseID: NewTimeOrderedID("LIC"), ActivationID: NewTimeOrderedID("ACT"), Product: "AI Game Manager Panel",
		MachineCode: st.MachineCode, InstallID: st.InstallID, Edition: "pro", IssuedAt: time.Now().Add(-2 * time.Hour).Unix(), ExpiresAt: time.Now().Add(-time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Persist an expired certificate to exercise runtime status/feature gating rather than Activate's preflight rejection.
	record := Record{Certificate: cert, ActivatedAt: time.Now().Add(-2 * time.Hour).Unix()}
	b, _ := json.Marshal(record)
	if err := os.MkdirAll(filepath.Dir(s.recordPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.recordPath, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.RequireFeature("ai.workbench"); err == nil {
		t.Fatal("expired license must block feature")
	}
	if _, err := s.Unbind(); err != nil {
		t.Fatal(err)
	}
	if err := s.RequireFeature("server.basic"); err == nil {
		t.Fatal("unbound license must block feature")
	}
}

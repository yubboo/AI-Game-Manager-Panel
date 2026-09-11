package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	license "github.com/yubboo/AI-Game-Manager-Panel/internal/system/license"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type keyMetadata struct {
	SchemaVersion int    `json:"schemaVersion"`
	KeyID         string `json:"keyId"`
	Fingerprint   string `json:"fingerprint"`
	PublicKey     string `json:"publicKey"`
	CreatedAt     int64  `json:"createdAt"`
	PrivateFile   string `json:"privateFile"`
	PublicFile    string `json:"publicFile"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "keygen":
		keygen(os.Args[2:])
	case "key-info":
		keyInfo(os.Args[2:])
	case "issue":
		issue(os.Args[2:])
	case "sync-public-key":
		syncPublicKey(os.Args[2:])
	case "verify":
		verifyCertificate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("AGMP License Admin")
	fmt.Println("  keygen --output-dir DIR")
	fmt.Println("  key-info --private-key FILE")
	fmt.Println("  issue --private-key FILE --expected-key-id AGMP-KID-... --machine BFM-... --install-id BFID-... [--license LIC-...] [--activation ACT-...] [--edition pro] [--features a,b] [--seats 1] [--expires 0] [--out FILE] [--out-dir DIR]")
	fmt.Println("  sync-public-key --ring FILE --key-root DIR")
	fmt.Println("  verify --ring FILE --certificate-file FILE --machine BFM-... --install-id BFID-...")
}

func keygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	outputDir := fs.String("output-dir", "", "directory outside the source repository")
	_ = fs.Parse(args)
	if strings.TrimSpace(*outputDir) == "" {
		fmt.Fprintln(os.Stderr, "keygen 必须显式指定 --output-dir；发行私钥禁止默认写入当前项目目录。")
		os.Exit(2)
	}

	absOutput, err := filepath.Abs(*outputDir)
	if err != nil {
		fatal(err)
	}
	if root, ok := findRepositoryRoot(); ok && pathInside(root, absOutput) {
		fmt.Fprintf(os.Stderr, "拒绝生成发行私钥：目标目录位于源码仓库内部：%s\n", absOutput)
		os.Exit(3)
	}
	if err := os.MkdirAll(absOutput, 0o700); err != nil {
		fatal(err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal(err)
	}
	keyID := license.PublicKeyID(pub)
	fingerprint := license.PublicKeyFingerprint(pub)
	privateName := "agmp-release-private.key"
	publicName := "agmp-release-public.pub"
	metadataName := "key-metadata.json"
	privatePath := filepath.Join(absOutput, privateName)
	publicPath := filepath.Join(absOutput, publicName)
	metadataPath := filepath.Join(absOutput, metadataName)

	if err := writeExclusive(privatePath, []byte(base64.StdEncoding.EncodeToString(priv)+"\n"), 0o600); err != nil {
		fatal(err)
	}
	if err := writeExclusive(publicPath, []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644); err != nil {
		_ = os.Remove(privatePath)
		fatal(err)
	}
	meta := keyMetadata{
		SchemaVersion: 1,
		KeyID:         keyID,
		Fingerprint:   fingerprint,
		PublicKey:     base64.StdEncoding.EncodeToString(pub),
		CreatedAt:     time.Now().Unix(),
		PrivateFile:   privateName,
		PublicFile:    publicName,
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := writeExclusive(metadataPath, append(b, '\n'), 0o644); err != nil {
		fatal(err)
	}

	fmt.Printf("KeyID: %s\n", keyID)
	fmt.Printf("Fingerprint: %s\n", fingerprint)
	fmt.Printf("PrivateKey: %s\n", privatePath)
	fmt.Printf("PublicKey: %s\n", publicPath)
	fmt.Printf("Metadata: %s\n", metadataPath)
	fmt.Println("发行私钥已生成在源码仓库外；不要上传 GitHub、云盘公开链接或放入安装包。")
}

func keyInfo(args []string) {
	fs := flag.NewFlagSet("key-info", flag.ExitOnError)
	keyFile := fs.String("private-key", "", "base64 Ed25519 private key file")
	_ = fs.Parse(args)
	if strings.TrimSpace(*keyFile) == "" {
		fs.Usage()
		os.Exit(2)
	}
	priv := readPrivateKey(*keyFile)
	pub := priv.Public().(ed25519.PublicKey)
	value := struct {
		KeyID       string `json:"keyId"`
		Fingerprint string `json:"fingerprint"`
		PublicKey   string `json:"publicKey"`
	}{
		KeyID:       license.PublicKeyID(pub),
		Fingerprint: license.PublicKeyFingerprint(pub),
		PublicKey:   base64.StdEncoding.EncodeToString(pub),
	}
	b, _ := json.Marshal(value)
	fmt.Println(string(b))
}

func issue(args []string) {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	keyFile := fs.String("private-key", "", "base64 Ed25519 private key file")
	expectedKeyID := fs.String("expected-key-id", "", "active repository key id; rejects wrong private key")
	machine := fs.String("machine", "", "machine code")
	installID := fs.String("install-id", "", "AI Game Manager Panel install id (BFID-...)")
	legacyIdentification := fs.String("identification", "", "deprecated alias of --install-id")
	id := fs.String("license", "", "license id; omitted = generated")
	activation := fs.String("activation", "", "activation id; omitted = generated")
	edition := fs.String("edition", "standard", "edition")
	featuresText := fs.String("features", "", "comma separated entitlements; omitted = edition defaults")
	seats := fs.Int("seats", 1, "seat limit")
	expires := fs.Int64("expires", 0, "unix seconds, 0=never")
	output := fs.String("out", "", "optional output file for BFLC2 certificate")
	outputDir := fs.String("out-dir", "", "optional output directory; filename is derived from LicenseID/ActivationID")
	_ = fs.Parse(args)
	if strings.TrimSpace(*installID) == "" {
		*installID = strings.TrimSpace(*legacyIdentification)
	}
	if strings.TrimSpace(*keyFile) == "" || strings.TrimSpace(*machine) == "" || strings.TrimSpace(*installID) == "" {
		fs.Usage()
		os.Exit(2)
	}
	if strings.TrimSpace(*id) == "" {
		*id = license.NewTimeOrderedID("LIC")
	}
	if strings.TrimSpace(*activation) == "" {
		*activation = license.NewTimeOrderedID("ACT")
	}

	priv := readPrivateKey(*keyFile)
	pub := priv.Public().(ed25519.PublicKey)
	issuerKeyID := license.PublicKeyID(pub)
	if expected := strings.TrimSpace(*expectedKeyID); expected != "" && !strings.EqualFold(expected, issuerKeyID) {
		fmt.Fprintf(os.Stderr, "发行私钥与当前 active 公钥不匹配：expected=%s actual=%s\n", expected, issuerKeyID)
		os.Exit(4)
	}

	features := []string{}
	for _, value := range strings.Split(*featuresText, ",") {
		if value = strings.TrimSpace(value); value != "" {
			features = append(features, value)
		}
	}
	certificate, err := license.IssueCertificate(priv, license.CertificatePayload{
		LicenseID: *id, ActivationID: *activation, Product: "AI Game Manager Panel", MachineCode: *machine, InstallID: *installID,
		Edition: *edition, Features: features, SeatLimit: *seats, IssuedAt: time.Now().Unix(), ExpiresAt: *expires, IssuerKeyID: issuerKeyID,
	})
	if err != nil {
		fatal(err)
	}
	outValue := strings.TrimSpace(*output)
	if outValue == "" && strings.TrimSpace(*outputDir) != "" {
		outValue = filepath.Join(*outputDir, *id+"_"+*activation+".bflc")
	}
	if outValue != "" {
		outPath, err := filepath.Abs(outValue)
		if err != nil {
			fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
			fatal(err)
		}
		if err := os.WriteFile(outPath, []byte(certificate+"\n"), 0o600); err != nil {
			fatal(err)
		}
		fmt.Printf("CertificateFile: %s\n", outPath)
	}
	fmt.Printf("LicenseID: %s\nActivationID: %s\nIssuerKeyID: %s\nCertificate:\n%s\n", *id, *activation, issuerKeyID, certificate)
}

func readPrivateKey(path string) ed25519.PrivateKey {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		fatal(fmt.Errorf("Ed25519 私钥格式无效：%s", path))
	}
	return ed25519.PrivateKey(priv)
}

func writeExclusive(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("拒绝覆盖已有文件 %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}

func findRepositoryRoot() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	current := cwd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func pathInside(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func syncPublicKey(args []string) {
	fs := flag.NewFlagSet("sync-public-key", flag.ExitOnError)
	ringFile := fs.String("ring", "", "vendor_public_keys.json path")
	keyRoot := fs.String("key-root", "", "external ReleaseKeys root")
	_ = fs.Parse(args)
	if strings.TrimSpace(*ringFile) == "" || strings.TrimSpace(*keyRoot) == "" {
		fs.Usage()
		os.Exit(2)
	}
	ringPath, err := filepath.Abs(*ringFile)
	if err != nil {
		fatal(err)
	}
	rootPath, err := filepath.Abs(*keyRoot)
	if err != nil {
		fatal(err)
	}
	meta, metaPath, err := newestValidKeyMetadata(rootPath)
	if err != nil {
		fatal(err)
	}

	var ring license.VendorKeyRing
	if b, err := os.ReadFile(ringPath); err == nil {
		if err := json.Unmarshal(b, &ring); err != nil {
			fatal(fmt.Errorf("公开密钥环格式无效: %w", err))
		}
	} else if !os.IsNotExist(err) {
		fatal(err)
	}
	if ring.SchemaVersion == 0 {
		ring.SchemaVersion = 1
	}

	found := false
	for i := range ring.Keys {
		entry := &ring.Keys[i]
		if strings.EqualFold(strings.TrimSpace(entry.KeyID), meta.KeyID) {
			entry.KeyID = meta.KeyID
			entry.Fingerprint = meta.Fingerprint
			entry.PublicKey = meta.PublicKey
			entry.Status = "active"
			entry.CreatedAt = meta.CreatedAt
			entry.Note = "由项目所有者本机发行密钥目录同步；私钥仍保存在源码仓库外。"
			found = true
			continue
		}
		if strings.EqualFold(strings.TrimSpace(entry.Status), "active") {
			entry.Status = "retired"
			if strings.TrimSpace(entry.Note) == "" {
				entry.Note = "已由后续发行密钥轮换替代；继续用于验证历史许可证。"
			}
		}
	}
	if !found {
		ring.Keys = append(ring.Keys, license.VendorKeyEntry{
			KeyID: meta.KeyID, Fingerprint: meta.Fingerprint, PublicKey: meta.PublicKey,
			Status: "active", CreatedAt: meta.CreatedAt,
			Note: "由项目所有者本机发行密钥目录同步；私钥仍保存在源码仓库外。",
		})
	}
	ring.ActiveKeyID = meta.KeyID
	b, err := json.MarshalIndent(ring, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := writeReplace(ringPath, append(b, '\n'), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("SyncedKeyID: %s\nFingerprint: %s\nMetadata: %s\nRing: %s\n", meta.KeyID, meta.Fingerprint, metaPath, ringPath)
}

func newestValidKeyMetadata(root string) (keyMetadata, string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return keyMetadata{}, "", fmt.Errorf("无法读取发行密钥目录 %s: %w", root, err)
	}
	var best keyMetadata
	bestPath := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "key-metadata.json")
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var meta keyMetadata
		if json.Unmarshal(b, &meta) != nil {
			continue
		}
		pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(meta.PublicKey))
		if err != nil || len(pub) != ed25519.PublicKeySize {
			continue
		}
		if !strings.EqualFold(meta.KeyID, license.PublicKeyID(pub)) || !strings.EqualFold(meta.Fingerprint, license.PublicKeyFingerprint(pub)) {
			continue
		}
		if bestPath == "" || meta.CreatedAt > best.CreatedAt {
			best = meta
			bestPath = path
		}
	}
	if bestPath == "" {
		return keyMetadata{}, "", fmt.Errorf("%s 中没有找到有效的 key-metadata.json", root)
	}
	return best, bestPath, nil
}

func verifyCertificate(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	ringFile := fs.String("ring", "", "vendor_public_keys.json path")
	certificateFile := fs.String("certificate-file", "", "BFLC2 certificate file")
	certificateValue := fs.String("certificate", "", "BFLC2 certificate text")
	machine := fs.String("machine", "", "machine code")
	installID := fs.String("install-id", "", "install id")
	_ = fs.Parse(args)
	if strings.TrimSpace(*ringFile) == "" || strings.TrimSpace(*machine) == "" || strings.TrimSpace(*installID) == "" {
		fs.Usage()
		os.Exit(2)
	}
	certificate := strings.TrimSpace(*certificateValue)
	if certificate == "" && strings.TrimSpace(*certificateFile) != "" {
		b, err := os.ReadFile(*certificateFile)
		if err != nil {
			fatal(err)
		}
		certificate = strings.TrimSpace(string(b))
	}
	if certificate == "" {
		fatal(fmt.Errorf("必须提供 --certificate-file 或 --certificate"))
	}
	b, err := os.ReadFile(*ringFile)
	if err != nil {
		fatal(err)
	}
	var ring license.VendorKeyRing
	if err := json.Unmarshal(b, &ring); err != nil {
		fatal(err)
	}
	trusted := make([]string, 0, len(ring.Keys))
	active := ""
	for _, entry := range ring.Keys {
		if strings.TrimSpace(entry.PublicKey) == "" {
			continue
		}
		trusted = append(trusted, entry.PublicKey)
		if strings.EqualFold(entry.KeyID, ring.ActiveKeyID) && strings.EqualFold(entry.Status, "active") {
			active = entry.PublicKey
		}
	}
	service := license.New(license.Config{Enabled: true, Product: "AI Game Manager Panel", PublicKey: active, TrustedPublicKeys: trusted}, filepath.Join(os.TempDir(), "agmp-license-verify", "activation.json"))
	payload, legacy, err := service.VerifyCertificateForDevice(certificate, *machine, *installID)
	if err != nil {
		fatal(err)
	}
	result := struct {
		Valid        bool                       `json:"valid"`
		Legacy       bool                       `json:"legacy"`
		License      license.CertificatePayload `json:"license"`
		VerifiedWith string                     `json:"verifiedWith"`
	}{Valid: true, Legacy: legacy, License: payload, VerifiedWith: payload.IssuerKeyID}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

func writeReplace(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agmp-public-ring-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
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
	return platformfiles.AtomicReplace(tmpPath, path)
}

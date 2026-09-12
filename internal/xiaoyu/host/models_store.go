package host

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	platformsecurity "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/security"
)

type modelStoreData struct {
	Version        int            `json:"version"`
	DefaultBrainID string         `json:"defaultBrainId,omitempty"`
	Profiles       []ModelProfile `json:"profiles"`
}

type ModelManager struct {
	mu    sync.RWMutex
	path  string
	vault *platformsecurity.Vault
}

func NewModelManager(path, vaultRoot string) *ModelManager {
	return &ModelManager{path: filepath.Clean(path), vault: platformsecurity.NewVault(vaultRoot)}
}

func (m *ModelManager) Catalog() (ModelCatalog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := m.loadLocked()
	if err != nil {
		return ModelCatalog{}, err
	}
	profiles := make([]ModelProfileView, 0, len(data.Profiles))
	for _, profile := range data.Profiles {
		profiles = append(profiles, m.viewLocked(profile, data.DefaultBrainID))
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt) })
	ready := false
	message := "请先在系统设置 → 模型管理中配置小鱼的大模型。"
	if data.DefaultBrainID != "" {
		if profile, ok := findProfile(data.Profiles, data.DefaultBrainID); ok {
			ready = ModelProfileReady(profile, m.vault.Exists(profile.SecretRef))
			if ready {
				message = "小鱼默认大脑已配置。"
			} else if !ModelBrainEligible(profile) {
				message = "该 Provider 已接入模型中心，但尚未完成 XiaoYu Brain 安全适配。"
			} else {
				message = "默认模型配置不完整，请检查授权方式、接口和模型名称。"
			}
		}
	}
	return ModelCatalog{Presets: BuiltinModelPresets(), Profiles: profiles, DefaultBrainID: data.DefaultBrainID, BrainReady: ready, Message: message}, nil
}

func (m *ModelManager) Save(request SaveModelRequest) (ModelProfileView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := m.loadLocked()
	if err != nil {
		return ModelProfileView{}, err
	}
	profile, index, exists := profileFromRequest(request, data.Profiles)
	if err := validateModelProfile(profile); err != nil {
		return ModelProfileView{}, err
	}
	now := time.Now().UTC()
	if exists {
		profile.CreatedAt = data.Profiles[index].CreatedAt
		profile.SecretRef = data.Profiles[index].SecretRef
		profile.LastTestAt = data.Profiles[index].LastTestAt
		profile.LastTestOK = data.Profiles[index].LastTestOK
		profile.LastTestMessage = data.Profiles[index].LastTestMessage
		if profile.SecretRef == "" {
			profile.SecretRef = modelSecretRef(profile.ID)
		}
	} else {
		profile.ID, err = newModelID()
		if err != nil {
			return ModelProfileView{}, err
		}
		profile.CreatedAt = now
		profile.SecretRef = modelSecretRef(profile.ID)
	}
	profile.UpdatedAt = now
	if strings.TrimSpace(request.APIKey) != "" {
		if err := m.vault.Put(profile.SecretRef, strings.TrimSpace(request.APIKey)); err != nil {
			return ModelProfileView{}, fmt.Errorf("保存模型密钥失败: %w", err)
		}
	}
	if ModelAuthNeedsSecret(profile) && !m.vault.Exists(profile.SecretRef) {
		return ModelProfileView{}, errors.New("该模型服务的 API Key 尚未配置")
	}
	if exists {
		data.Profiles[index] = profile
	} else {
		data.Profiles = append(data.Profiles, profile)
	}
	if data.DefaultBrainID == "" && ModelProfileReady(profile, m.vault.Exists(profile.SecretRef)) {
		data.DefaultBrainID = profile.ID
	}
	if err := m.saveLocked(data); err != nil {
		return ModelProfileView{}, err
	}
	return m.viewLocked(profile, data.DefaultBrainID), nil
}

func (m *ModelManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := m.loadLocked()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	next := data.Profiles[:0]
	var removed *ModelProfile
	for _, profile := range data.Profiles {
		if profile.ID == id {
			copy := profile
			removed = &copy
			continue
		}
		next = append(next, profile)
	}
	if removed == nil {
		return fmt.Errorf("model profile not found: %s", id)
	}
	data.Profiles = append([]ModelProfile(nil), next...)
	if data.DefaultBrainID == id {
		data.DefaultBrainID = ""
	}
	if err := m.saveLocked(data); err != nil {
		return err
	}
	return m.vault.Delete(removed.SecretRef)
}

func (m *ModelManager) SetDefault(id string) (ModelProfileView, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := m.loadLocked()
	if err != nil {
		return ModelProfileView{}, err
	}
	profile, ok := findProfile(data.Profiles, strings.TrimSpace(id))
	if !ok {
		return ModelProfileView{}, fmt.Errorf("model profile not found: %s", id)
	}
	if !profile.Enabled {
		return ModelProfileView{}, errors.New("不能把已禁用模型设为小鱼默认大脑")
	}
	if !ModelBrainEligible(profile) {
		return ModelProfileView{}, errors.New("该 Provider 尚未完成 XiaoYu Brain 安全适配，不能设为默认大脑")
	}
	if !ModelProfileReady(profile, m.vault.Exists(profile.SecretRef)) {
		return ModelProfileView{}, errors.New("模型配置不完整，请检查授权方式、接口和模型名称")
	}
	data.DefaultBrainID = profile.ID
	if err := m.saveLocked(data); err != nil {
		return ModelProfileView{}, err
	}
	return m.viewLocked(profile, data.DefaultBrainID), nil
}

func (m *ModelManager) Default() (ModelProfile, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, err := m.loadLocked()
	if err != nil {
		return ModelProfile{}, "", err
	}
	if data.DefaultBrainID == "" {
		return ModelProfile{}, "", ErrBrainUnavailable
	}
	profile, ok := findProfile(data.Profiles, data.DefaultBrainID)
	if !ok || !profile.Enabled || !ModelBrainEligible(profile) {
		return ModelProfile{}, "", ErrBrainUnavailable
	}
	secret := ""
	if m.vault.Exists(profile.SecretRef) {
		secret, err = m.vault.Get(profile.SecretRef)
		if err != nil {
			return ModelProfile{}, "", err
		}
	}
	if secret == "" && ModelAuthNeedsSecret(profile) {
		return ModelProfile{}, "", errors.New("xiaoyu model api key is missing")
	}
	return cloneModelProfile(profile), secret, nil
}

func (m *ModelManager) ResolveConnection(request ModelConnectionRequest) (ModelProfile, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	profile := ModelProfile{ID: strings.TrimSpace(request.ID), Provider: strings.TrimSpace(request.Provider), Protocol: request.Protocol, AuthMode: request.AuthMode, BaseURL: strings.TrimSpace(request.BaseURL), Model: strings.TrimSpace(request.Model), ThinkingMode: strings.TrimSpace(request.ThinkingMode), ReasoningEffort: strings.TrimSpace(request.ReasoningEffort), Extra: cloneMap(request.Extra), Enabled: true}
	if profile.ID != "" {
		data, err := m.loadLocked()
		if err != nil {
			return ModelProfile{}, "", err
		}
		if existing, ok := findProfile(data.Profiles, profile.ID); ok {
			if profile.Provider == "" {
				profile.Provider = existing.Provider
			}
			if profile.Protocol == "" {
				profile.Protocol = existing.Protocol
			}
			if profile.AuthMode == "" {
				profile.AuthMode = existing.AuthMode
			}
			if profile.BaseURL == "" {
				profile.BaseURL = existing.BaseURL
			}
			if profile.Model == "" {
				profile.Model = existing.Model
			}
			if profile.ThinkingMode == "" {
				profile.ThinkingMode = existing.ThinkingMode
			}
			if profile.ReasoningEffort == "" {
				profile.ReasoningEffort = existing.ReasoningEffort
			}
			if len(profile.Extra) == 0 {
				profile.Extra = cloneMap(existing.Extra)
			}
			profile.SecretRef = existing.SecretRef
		}
	}
	applyPresetDefaults(&profile)
	if err := validateConnectionProfile(profile); err != nil {
		return ModelProfile{}, "", err
	}
	secret := strings.TrimSpace(request.APIKey)
	if secret == "" && profile.SecretRef != "" && m.vault.Exists(profile.SecretRef) {
		var err error
		secret, err = m.vault.Get(profile.SecretRef)
		if err != nil {
			return ModelProfile{}, "", err
		}
	}
	if secret == "" && ModelAuthNeedsSecret(profile) {
		return ModelProfile{}, "", errors.New("API Key 未配置")
	}
	return profile, secret, nil
}

func (m *ModelManager) RecordTest(id string, result ModelConnectionResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := m.loadLocked()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for i := range data.Profiles {
		if data.Profiles[i].ID != id {
			continue
		}
		data.Profiles[i].LastTestAt = time.Now().UTC()
		data.Profiles[i].LastTestOK = result.OK
		data.Profiles[i].LastTestMessage = strings.TrimSpace(result.Message)
		data.Profiles[i].UpdatedAt = time.Now().UTC()
		return m.saveLocked(data)
	}
	return nil
}

func (m *ModelManager) loadLocked() (modelStoreData, error) {
	data := modelStoreData{Version: 1, Profiles: []ModelProfile{}}
	raw, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return modelStoreData{}, fmt.Errorf("解析模型配置失败: %w", err)
	}
	if data.Version <= 0 {
		data.Version = 1
	}
	if data.Profiles == nil {
		data.Profiles = []ModelProfile{}
	}
	for i := range data.Profiles {
		data.Profiles[i].SecretRef = modelSecretRef(data.Profiles[i].ID)
		migrateBuiltinProtocol(&data.Profiles[i])
		if data.Profiles[i].AuthMode == "" {
			data.Profiles[i].AuthMode = ResolveModelAuthMode(data.Profiles[i])
		}
	}
	return data, nil
}

func (m *ModelManager) saveLocked(data modelStoreData) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	temp := m.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(temp, m.path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

func (m *ModelManager) viewLocked(profile ModelProfile, defaultID string) ModelProfileView {
	hasAPIKey := m.vault.Exists(profile.SecretRef)
	authMode := ResolveModelAuthMode(profile)
	hasCredential := hasAPIKey || authMode == ModelAuthSubscription || authMode == ModelAuthLocal
	return ModelProfileView{ID: profile.ID, Name: profile.Name, Provider: profile.Provider, Protocol: profile.Protocol, AuthMode: authMode, ProviderKind: ModelProviderKindFor(profile), BrainEligible: ModelBrainEligible(profile), BaseURL: profile.BaseURL, Model: profile.Model, Enabled: profile.Enabled, ContextWindow: profile.ContextWindow, MaxOutputTokens: profile.MaxOutputTokens, ThinkingMode: profile.ThinkingMode, ReasoningEffort: profile.ReasoningEffort, Extra: cloneMap(profile.Extra), Capabilities: ResolveModelCapabilities(profile), HasAPIKey: hasAPIKey, HasCredential: hasCredential, LastTestAt: profile.LastTestAt, LastTestOK: profile.LastTestOK, LastTestMessage: profile.LastTestMessage, IsDefault: profile.ID == defaultID, CreatedAt: profile.CreatedAt, UpdatedAt: profile.UpdatedAt}
}

func profileFromRequest(request SaveModelRequest, profiles []ModelProfile) (ModelProfile, int, bool) {
	id := strings.TrimSpace(request.ID)
	index := -1
	exists := false
	if id != "" {
		for i := range profiles {
			if profiles[i].ID == id {
				index, exists = i, true
				break
			}
		}
	}
	profile := ModelProfile{ID: id, Name: strings.TrimSpace(request.Name), Provider: strings.TrimSpace(request.Provider), Protocol: request.Protocol, AuthMode: request.AuthMode, BaseURL: strings.TrimSpace(request.BaseURL), Model: strings.TrimSpace(request.Model), Enabled: request.Enabled, ContextWindow: request.ContextWindow, MaxOutputTokens: request.MaxOutputTokens, ThinkingMode: strings.TrimSpace(request.ThinkingMode), ReasoningEffort: strings.TrimSpace(request.ReasoningEffort), Extra: cloneMap(request.Extra)}
	applyPresetDefaults(&profile)
	return profile, index, exists
}

func applyPresetDefaults(profile *ModelProfile) {
	if profile == nil {
		return
	}
	for _, preset := range BuiltinModelPresets() {
		if preset.ID != profile.Provider {
			continue
		}
		if profile.Protocol == "" {
			profile.Protocol = preset.Protocol
		}
		if profile.AuthMode == "" {
			profile.AuthMode = preset.DefaultAuthMode
		}
		if strings.TrimSpace(profile.BaseURL) == "" {
			profile.BaseURL = preset.DefaultBaseURL
		}
		break
	}
	if profile.AuthMode == "" {
		profile.AuthMode = ModelAuthAPIKey
	}
	profile.BaseURL = strings.TrimRight(strings.TrimSpace(profile.BaseURL), "/")
	if profile.ContextWindow <= 0 {
		profile.ContextWindow = 128000
	}
	if profile.MaxOutputTokens <= 0 {
		profile.MaxOutputTokens = 8192
	}
	if profile.ThinkingMode == "" {
		profile.ThinkingMode = "default"
	}
	if profile.ReasoningEffort == "" {
		profile.ReasoningEffort = "default"
	}
	if profile.Extra == nil {
		profile.Extra = map[string]any{}
	}
}

func migrateBuiltinProtocol(profile *ModelProfile) {
	if profile == nil || profile.Protocol != ProtocolOpenAICompatible {
		return
	}
	base := strings.TrimRight(strings.ToLower(strings.TrimSpace(profile.BaseURL)), "/")
	switch strings.TrimSpace(profile.Provider) {
	case "openai":
		if base == "https://api.openai.com/v1" {
			profile.Protocol = ProtocolOpenAIResponses
		}
	case "deepseek":
		if base == "https://api.deepseek.com" || base == "https://api.deepseek.com/v1" {
			profile.Protocol = ProtocolDeepSeek
		}
	}
}

func validateModelProfile(profile ModelProfile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return errors.New("模型名称不能为空")
	}
	if strings.TrimSpace(profile.Provider) == "" {
		return errors.New("服务商不能为空")
	}
	if strings.TrimSpace(profile.Model) == "" && ModelBrainEligible(profile) {
		return errors.New("模型代码不能为空")
	}
	return validateConnectionProfile(profile)
}

func validateConnectionProfile(profile ModelProfile) error {
	if profile.Protocol != ProtocolOpenAIResponses && profile.Protocol != ProtocolDeepSeek && profile.Protocol != ProtocolOpenAICompatible && profile.Protocol != ProtocolAnthropic && profile.Protocol != ProtocolGemini && profile.Protocol != ProtocolCodexAppServer {
		return fmt.Errorf("不支持的模型协议: %s", profile.Protocol)
	}
	mode := ResolveModelAuthMode(profile)
	if !ModelAuthModeAllowed(profile.Provider, mode) {
		return fmt.Errorf("服务商 %s 不支持授权方式 %s", profile.Provider, mode)
	}
	if profile.Protocol == ProtocolCodexAppServer {
		if mode != ModelAuthSubscription {
			return errors.New("Codex Provider 必须使用套餐/订阅授权")
		}
		if strings.TrimSpace(ModelProviderExecutable(profile)) == "" {
			return errors.New("Codex Provider 缺少官方 CLI 执行器")
		}
	} else {
		parsed, err := url.Parse(strings.TrimSpace(profile.BaseURL))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return errors.New("Base URL 必须是有效的 HTTP/HTTPS 地址")
		}
	}
	if err := validateModelExtra(profile.Extra); err != nil {
		return err
	}
	return nil
}

func validateModelExtra(extra map[string]any) error {
	var walk func(path string, value any) error
	walk = func(path string, value any) error {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				normalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(key))
				for _, marker := range []string{"apikey", "authorization", "accesstoken", "refreshtoken", "password", "secret", "bearertoken"} {
					if strings.Contains(normalized, marker) {
						location := key
						if path != "" {
							location = path + "." + key
						}
						return fmt.Errorf("额外请求参数 %s 看起来包含密钥；请使用专用 API Key 字段保存敏感信息", location)
					}
				}
				next := key
				if path != "" {
					next = path + "." + key
				}
				if err := walk(next, child); err != nil {
					return err
				}
			}
		case []any:
			for i, child := range typed {
				if err := walk(fmt.Sprintf("%s[%d]", path, i), child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk("", extra)
}

func presetAPIKeyOptional(provider string) bool {
	for _, preset := range BuiltinModelPresets() {
		if preset.ID == strings.TrimSpace(provider) {
			return preset.APIKeyOptional
		}
	}
	return false
}

func findProfile(profiles []ModelProfile, id string) (ModelProfile, bool) {
	for _, profile := range profiles {
		if profile.ID == id {
			return cloneModelProfile(profile), true
		}
	}
	return ModelProfile{}, false
}

func modelSecretRef(id string) string { return "xiaoyu:model:" + strings.TrimSpace(id) + ":api-key" }

func newModelID() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "model_" + hex.EncodeToString(raw), nil
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for k, v := range value {
		out[k] = v
	}
	return out
}
func cloneModelProfile(value ModelProfile) ModelProfile {
	value.Extra = cloneMap(value.Extra)
	return value
}

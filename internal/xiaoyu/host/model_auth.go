package host

import "strings"

func ModelPresetByID(id string) (ModelPreset, bool) {
	id = strings.TrimSpace(id)
	for _, preset := range BuiltinModelPresets() {
		if preset.ID == id {
			return preset, true
		}
	}
	return ModelPreset{}, false
}

func ResolveModelAuthMode(profile ModelProfile) ModelAuthMode {
	if profile.AuthMode != "" {
		return profile.AuthMode
	}
	if preset, ok := ModelPresetByID(profile.Provider); ok && preset.DefaultAuthMode != "" {
		return preset.DefaultAuthMode
	}
	return ModelAuthAPIKey
}

func ModelAuthModeAllowed(provider string, mode ModelAuthMode) bool {
	preset, ok := ModelPresetByID(provider)
	if !ok {
		return mode == ModelAuthAPIKey
	}
	for _, allowed := range preset.AuthModes {
		if allowed == mode {
			return true
		}
	}
	return false
}

func ModelBrainEligible(profile ModelProfile) bool {
	preset, ok := ModelPresetByID(profile.Provider)
	return !ok || preset.BrainEligible
}

func ModelProviderKindFor(profile ModelProfile) ModelProviderKind {
	if preset, ok := ModelPresetByID(profile.Provider); ok {
		return preset.Kind
	}
	return ModelProviderAPI
}

func ModelProviderExecutable(profile ModelProfile) string {
	if preset, ok := ModelPresetByID(profile.Provider); ok {
		return strings.TrimSpace(preset.Executable)
	}
	return ""
}

func ModelAuthNeedsSecret(profile ModelProfile) bool {
	return ResolveModelAuthMode(profile) == ModelAuthAPIKey && !presetAPIKeyOptional(profile.Provider)
}

func ModelProfileReady(profile ModelProfile, hasSecret bool) bool {
	if !profile.Enabled || !ModelBrainEligible(profile) || strings.TrimSpace(profile.Model) == "" {
		return false
	}
	switch ResolveModelAuthMode(profile) {
	case ModelAuthSubscription:
		return strings.TrimSpace(ModelProviderExecutable(profile)) != ""
	case ModelAuthLocal:
		return strings.TrimSpace(profile.BaseURL) != ""
	default:
		return strings.TrimSpace(profile.BaseURL) != "" && (hasSecret || presetAPIKeyOptional(profile.Provider))
	}
}

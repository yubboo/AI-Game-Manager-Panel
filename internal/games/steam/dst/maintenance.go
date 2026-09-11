package dst

// RequiresStoppedRuntimeForSteamMaintenance contains DST-specific safety policy.
// The shared Steam maintenance service intentionally knows nothing about DST.
// Only the Dedicated Server AppID may have files in active use by AGMP-managed
// Master/Caves processes, so install/validate must be blocked while that runtime is active.
func RequiresStoppedRuntimeForSteamMaintenance(appID uint32) bool {
	return appID == ServerAppID
}

package game

import "strings"

// CatalogEntry describes a game that AI Game Manager Panel knows how to present to users.
// A catalog entry is static provider metadata; it is not a user collection/favourite record.
type CatalogEntry struct {
	ID          ID             `json:"id"`
	Family      Family         `json:"family"`
	NameZH      string         `json:"nameZh"`
	NameEN      string         `json:"nameEn"`
	Description string         `json:"description"`
	Aliases     []string       `json:"aliases"`
	Steam       *SteamMetadata `json:"steam,omitempty"`
}

// SteamMetadata keeps Steam-specific IDs next to the game catalog entry while
// the generic game core remains independent from the Steam platform package.
type SteamMetadata struct {
	GameAppID   uint32 `json:"gameAppId"`
	ServerAppID uint32 `json:"serverAppId"`
}

func (e CatalogEntry) Matches(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	values := []string{
		string(e.ID),
		string(e.Family),
		e.NameZH,
		e.NameEN,
		e.Description,
	}
	values = append(values, e.Aliases...)
	if e.Steam != nil {
		values = append(values,
			uint32String(e.Steam.GameAppID),
			uint32String(e.Steam.ServerAppID),
		)
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func uint32String(value uint32) string {
	if value == 0 {
		return ""
	}
	const digits = "0123456789"
	var buf [10]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}

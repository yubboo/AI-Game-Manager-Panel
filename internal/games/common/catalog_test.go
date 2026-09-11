package game

import "testing"

func TestCatalogEntryMatchesLocalizedNamesAndAppIDs(t *testing.T) {
	entry := CatalogEntry{
		ID:      "steam.dst",
		Family:  FamilySteam,
		NameZH:  "饥荒联机版",
		NameEN:  "Don't Starve Together",
		Aliases: []string{"DST"},
		Steam: &SteamMetadata{
			GameAppID:   322330,
			ServerAppID: 343050,
		},
	}

	for _, query := range []string{"饥荒", "don't starve", "dst", "322330", "343050", "steam.dst"} {
		if !entry.Matches(query) {
			t.Fatalf("expected query %q to match", query)
		}
	}
	if entry.Matches("palworld") {
		t.Fatal("unexpected match")
	}
}

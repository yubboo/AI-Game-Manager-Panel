package dst

// SaveSource mirrors DSTCamp's SERVER/LOCAL distinction. SERVER clusters live
// directly under the Klei root; LOCAL clusters live inside the numeric user-ID
// directory.
type SaveSource string

const (
	SaveSourceServer SaveSource = "server"
	SaveSourceLocal  SaveSource = "local"
)

// Distribution is intentionally separate from AGMP's generic Steam
// platform abstraction because the Python baseline also supports WeGame/Rail.
type Distribution string

const (
	DistributionSteam  Distribution = "steam"
	DistributionWeGame Distribution = "wegame"
)

type Shard struct {
	Name             string `json:"name"`
	Path             string `json:"path"`
	ModOverridesPath string `json:"modOverridesPath"`
	LevelDataPath    string `json:"levelDataPath"`
}

type Cluster struct {
	Name             string       `json:"name"`
	Path             string       `json:"path"`
	Source           SaveSource   `json:"source"`
	Distribution     Distribution `json:"distribution"`
	Shards           []Shard      `json:"shards"`
	ModOverridesPath string       `json:"modOverridesPath"`
	AdminListPath    string       `json:"adminListPath"`
	TokenPath        string       `json:"tokenPath"`
	BlockListPath    string       `json:"blockListPath"`
}

type Environment struct {
	DocumentsDir   string    `json:"documentsDir"`
	KleiRoot       string    `json:"kleiRoot"`
	WeGameKleiRoot string    `json:"wegameKleiRoot"`
	UserID         string    `json:"userId"`
	WeGameUserID   string    `json:"wegameUserId"`
	ClientConfig   string    `json:"clientConfig"`
	Clusters       []Cluster `json:"clusters"`
}

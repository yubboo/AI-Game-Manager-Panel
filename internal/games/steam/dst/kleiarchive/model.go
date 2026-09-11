package kleiarchive

const (
	ModeTokenOnly  = "token_only"
	ModeFullConfig = "full_config"
)

type InspectRequest struct {
	ArchiveName   string `json:"archiveName"`
	ArchiveBase64 string `json:"archiveBase64"`
}

type ImportRequest struct {
	ClusterPath   string `json:"clusterPath"`
	ArchiveName   string `json:"archiveName"`
	ArchiveBase64 string `json:"archiveBase64"`
	Mode          string `json:"mode"`
}

type Preview struct {
	ArchiveName        string `json:"archiveName"`
	RootPrefix         string `json:"rootPrefix"`
	HasToken           bool   `json:"hasToken"`
	HasClusterINI      bool   `json:"hasClusterIni"`
	HasMasterServerINI bool   `json:"hasMasterServerIni"`
	HasCavesServerINI  bool   `json:"hasCavesServerIni"`
	ServerName         string `json:"serverName"`
	MaxPlayers         string `json:"maxPlayers"`
	GameMode           string `json:"gameMode"`
	Description        string `json:"description"`
	Passworded         bool   `json:"passworded"`
	EntryCount         int    `json:"entryCount"`
	UncompressedBytes  int64  `json:"uncompressedBytes"`
}

type ImportResult struct {
	Mode              string   `json:"mode"`
	ClusterPath       string   `json:"clusterPath"`
	TokenConfigured   bool     `json:"tokenConfigured"`
	ConfigFilesCopied int      `json:"configFilesCopied"`
	BackupPath        string   `json:"backupPath"`
	Warnings          []string `json:"warnings"`
	Preview           Preview  `json:"preview"`
}

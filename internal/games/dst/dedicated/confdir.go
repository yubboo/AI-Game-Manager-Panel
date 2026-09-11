package dedicated

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrConfDirCrossDrive = errors.New("klei root and Documents/Klei are on different drives")

// ResolveConfDirArg mirrors DSTCamp resolve_conf_dir_arg(). The DST engine
// interprets -conf_dir relative to the real Windows Documents/Klei directory.
// For the default DoNotStarveTogether root, the argument is omitted entirely.
func ResolveConfDirArg(documentsDir, kleiRoot string) (string, error) {
	base := filepath.Join(documentsDir, "Klei")
	defaultRoot := filepath.Join(base, "DoNotStarveTogether")
	if sameFilesystemPath(kleiRoot, defaultRoot) {
		return "", nil
	}

	baseVolume := windowsVolume(base)
	rootVolume := windowsVolume(kleiRoot)
	if baseVolume == "" {
		baseVolume = filepath.VolumeName(base)
	}
	if rootVolume == "" {
		rootVolume = filepath.VolumeName(kleiRoot)
	}
	if baseVolume != "" && rootVolume != "" && !strings.EqualFold(baseVolume, rootVolume) {
		return "", ErrConfDirCrossDrive
	}

	rel, err := filepath.Rel(base, kleiRoot)
	if err != nil {
		return "", ErrConfDirCrossDrive
	}
	return rel, nil
}

func DescribeConfDir(documentsDir, kleiRoot string) ConfDirInfo {
	result := ConfDirInfo{
		DocumentsDir: documentsDir,
		BaseDir:      filepath.Join(documentsDir, "Klei"),
		KleiRoot:     kleiRoot,
	}
	if strings.TrimSpace(kleiRoot) == "" {
		result.Error = "尚未检测到 Steam 版 Klei 根目录"
		return result
	}
	arg, err := ResolveConfDirArg(documentsDir, kleiRoot)
	if err != nil {
		result.Error = "Klei 根目录与 Windows 文档目录不在同一盘符，DST 的 -conf_dir 无法表达这个相对路径"
		return result
	}
	result.Argument = arg
	result.Default = arg == ""
	result.Valid = true
	return result
}

func sameFilesystemPath(left, right string) bool {
	cleanLeft := filepath.Clean(left)
	cleanRight := filepath.Clean(right)
	if runtime.GOOS == "windows" || windowsVolume(left) != "" || windowsVolume(right) != "" {
		return strings.EqualFold(strings.ReplaceAll(cleanLeft, "/", `\`), strings.ReplaceAll(cleanRight, "/", `\`))
	}
	return cleanLeft == cleanRight
}

func windowsVolume(path string) string {
	value := strings.TrimSpace(path)
	if len(value) >= 2 && value[1] == ':' {
		c := value[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return strings.ToUpper(value[:2])
		}
	}
	return ""
}

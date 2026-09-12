package minecraft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	MojangMetaBase = "https://piston-meta.mojang.com"
	PaperAPIBase   = "https://fill.papermc.io/v3"
	FabricMetaBase = "https://meta.fabricmc.net"
)

type Resolver struct {
	Client                            *http.Client
	MojangBase, PaperBase, FabricBase string
}

type manifestJSON struct {
	Latest struct {
		Release string `json:"release"`
	} `json:"latest"`
	Versions []struct{ ID, Type, URL string } `json:"versions"`
}
type versionJSON struct {
	JavaVersion *struct {
		Major int `json:"majorVersion"`
	} `json:"javaVersion"`
	Downloads *struct {
		Server *struct {
			URL, SHA1 string
			Size      int64
		} `json:"server"`
	} `json:"downloads"`
}
type paperJSON struct {
	ID        int `json:"id"`
	Downloads struct {
		Server struct {
			Name, URL string
			Size      int64
			Checksums struct {
				SHA256 string `json:"sha256"`
			} `json:"checksums"`
		} `json:"server:default"`
	} `json:"downloads"`
}
type fabricLoader struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}
type fabricInstaller struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func (r Resolver) defaults() Resolver {
	if r.Client == nil {
		r.Client = http.DefaultClient
	}
	if strings.TrimSpace(r.MojangBase) == "" {
		r.MojangBase = MojangMetaBase
	}
	if strings.TrimSpace(r.PaperBase) == "" {
		r.PaperBase = PaperAPIBase
	}
	if strings.TrimSpace(r.FabricBase) == "" {
		r.FabricBase = FabricMetaBase
	}
	r.MojangBase = strings.TrimRight(r.MojangBase, "/")
	r.PaperBase = strings.TrimRight(r.PaperBase, "/")
	r.FabricBase = strings.TrimRight(r.FabricBase, "/")
	return r
}

func (r Resolver) Resolve(ctx context.Context, requested string, software Software) (VersionFacts, error) {
	r = r.defaults()
	var man manifestJSON
	if err := r.getJSON(ctx, r.MojangBase+"/mc/game/version_manifest_v2.json", &man, 8<<20); err != nil {
		return VersionFacts{}, fmt.Errorf("查询 Mojang 版本清单失败: %w", err)
	}
	version := strings.TrimSpace(requested)
	if version == "" {
		version = man.Latest.Release
	}
	var entryURL string
	for _, e := range man.Versions {
		if e.ID == version {
			entryURL = e.URL
			break
		}
	}
	if entryURL == "" {
		return VersionFacts{}, fmt.Errorf("Minecraft 版本 %q 不存在于 Mojang 官方清单；当前最新稳定版 %s", version, man.Latest.Release)
	}
	var detail versionJSON
	if err := r.getJSON(ctx, entryURL, &detail, 8<<20); err != nil {
		return VersionFacts{}, fmt.Errorf("查询 Minecraft %s 官方详情失败: %w", version, err)
	}
	javaMajor := 0
	if detail.JavaVersion != nil {
		javaMajor = detail.JavaVersion.Major
	}
	if javaMajor == 0 {
		return VersionFacts{}, errors.New("Mojang 官方版本详情没有 javaVersion，拒绝猜测 Java 版本")
	}
	facts := VersionFacts{RequestedVersion: requested, Version: version, LatestRelease: man.Latest.Release, JavaMajor: javaMajor, Sources: []string{"mojang"}}
	switch software {
	case SoftwareVanilla:
		if detail.Downloads == nil || detail.Downloads.Server == nil {
			return VersionFacts{}, fmt.Errorf("Minecraft %s 没有 Mojang 官方服务端下载", version)
		}
		d := detail.Downloads.Server
		facts.Artifact = Artifact{Software: software, URL: d.URL, FileName: "server.jar", HashAlgorithm: "sha1", Hash: strings.ToLower(d.SHA1), Size: d.Size, Trust: "Mojang 官方 URL + SHA1"}
	case SoftwarePaper:
		var p paperJSON
		endpoint := fmt.Sprintf("%s/projects/paper/versions/%s/builds/latest", r.PaperBase, url.PathEscape(version))
		if err := r.getJSON(ctx, endpoint, &p, 8<<20); err != nil {
			return VersionFacts{}, fmt.Errorf("查询 Paper %s 最新构建失败: %w", version, err)
		}
		if p.Downloads.Server.URL == "" || p.Downloads.Server.Checksums.SHA256 == "" {
			return VersionFacts{}, errors.New("Paper API 返回的构建缺少下载地址或 SHA256")
		}
		facts.Sources = append(facts.Sources, "papermc")
		facts.Artifact = Artifact{Software: software, URL: p.Downloads.Server.URL, FileName: "server.jar", HashAlgorithm: "sha256", Hash: strings.ToLower(p.Downloads.Server.Checksums.SHA256), Size: p.Downloads.Server.Size, Build: fmt.Sprint(p.ID), Trust: "PaperMC Fill API v3 + SHA256"}
	case SoftwareFabric:
		var loaders []fabricLoader
		if err := r.getJSON(ctx, fmt.Sprintf("%s/v2/versions/loader/%s", r.FabricBase, url.PathEscape(version)), &loaders, 8<<20); err != nil {
			return VersionFacts{}, fmt.Errorf("查询 Fabric loader 失败: %w", err)
		}
		loader := ""
		for _, item := range loaders {
			if item.Loader.Stable {
				loader = item.Loader.Version
				break
			}
		}
		if loader == "" && len(loaders) > 0 {
			loader = loaders[0].Loader.Version
		}
		if loader == "" {
			return VersionFacts{}, fmt.Errorf("Fabric 对 Minecraft %s 没有可用 loader", version)
		}
		var installers []fabricInstaller
		if err := r.getJSON(ctx, r.FabricBase+"/v2/versions/installer", &installers, 8<<20); err != nil {
			return VersionFacts{}, fmt.Errorf("查询 Fabric installer 失败: %w", err)
		}
		installer := ""
		for _, item := range installers {
			if item.Stable {
				installer = item.Version
				break
			}
		}
		if installer == "" && len(installers) > 0 {
			installer = installers[0].Version
		}
		if installer == "" {
			return VersionFacts{}, errors.New("Fabric installer 列表为空")
		}
		artifactURL := fmt.Sprintf("%s/v2/versions/loader/%s/%s/%s/server/jar", r.FabricBase, url.PathEscape(version), url.PathEscape(loader), url.PathEscape(installer))
		facts.Sources = append(facts.Sources, "fabric")
		facts.Artifact = Artifact{Software: software, URL: artifactURL, FileName: "server.jar", Loader: loader, Installer: installer, Trust: "Fabric 官方 Meta；整包无官方哈希，下载后记录本地 SHA256"}
	default:
		return VersionFacts{}, fmt.Errorf("不支持的 Minecraft 服务端类型 %q；当前支持 vanilla、paper、fabric", software)
	}
	return facts, nil
}

func (r Resolver) getJSON(ctx context.Context, endpoint string, out any, max int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "AGMP/0.3.0")
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, max)).Decode(out)
}

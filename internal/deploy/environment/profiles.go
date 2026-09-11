package environment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

type GameRuntimeRequirement struct {
	ID        string      `json:"id"`
	Kind      RuntimeKind `json:"kind"`
	Major     int         `json:"major,omitempty"`
	Required  bool        `json:"required"`
	Satisfied bool        `json:"satisfied"`
	RuntimeID string      `json:"runtimeId,omitempty"`
	Message   string      `json:"message"`
}

type SystemPrerequisite struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Required       bool   `json:"required"`
	Detected       bool   `json:"detected"`
	Installable    bool   `json:"installable"`
	PackageManager string `json:"packageManager,omitempty"`
	Message        string `json:"message"`
}

type GameRuntimeProfile struct {
	GameID              string                   `json:"gameId"`
	Name                string                   `json:"name"`
	Platform            string                   `json:"platform"`
	Architecture        string                   `json:"architecture"`
	Ready               bool                     `json:"ready"`
	Requirements        []GameRuntimeRequirement `json:"requirements"`
	SystemPrerequisites []SystemPrerequisite     `json:"systemPrerequisites"`
}

type GameRuntimeProfileRequest struct {
	GameID    string `json:"gameId"`
	JavaMajor int    `json:"javaMajor,omitempty"`
}

type InstallSystemPrerequisiteRequest struct {
	ID string `json:"id"`
}

func (s *Service) GameRuntimeProfiles(ctx context.Context) []GameRuntimeProfile {
	return []GameRuntimeProfile{
		s.GameRuntimeProfile(ctx, GameRuntimeProfileRequest{GameID: "minecraft", JavaMajor: 21}),
		s.GameRuntimeProfile(ctx, GameRuntimeProfileRequest{GameID: "dst"}),
	}
}

func (s *Service) GameRuntimeProfile(ctx context.Context, request GameRuntimeProfileRequest) GameRuntimeProfile {
	id := strings.ToLower(strings.TrimSpace(request.GameID))
	profile := GameRuntimeProfile{GameID: id, Platform: runtime.GOOS, Architecture: runtime.GOARCH, Ready: true}
	switch id {
	case "minecraft":
		profile.Name = "Minecraft"
		major := request.JavaMajor
		if major == 0 {
			major = 21
		}
		req := GameRuntimeRequirement{ID: fmt.Sprintf("java-%d", major), Kind: RuntimeJava, Major: major, Required: true}
		if value, err := s.ResolveRuntime(ResolveRuntimeRequest{Kind: RuntimeJava, Major: major}); err == nil {
			req.Satisfied = true
			req.RuntimeID = value.ID
			req.Message = "Java Runtime 已就绪"
		} else {
			req.Message = fmt.Sprintf("需要 Java %d；请在运行环境中安装或登记已有 Java", major)
			profile.Ready = false
		}
		profile.Requirements = []GameRuntimeRequirement{req}
	case "dst":
		profile.Name = "Don't Starve Together"
		req := GameRuntimeRequirement{ID: "steamcmd", Kind: RuntimeSteamCMD, Required: true}
		if value, err := s.ResolveRuntime(ResolveRuntimeRequest{Kind: RuntimeSteamCMD}); err == nil {
			req.Satisfied = true
			req.RuntimeID = value.ID
			req.Message = "SteamCMD Runtime 已就绪"
		} else if detected := s.Status(ctx).SteamCMD; detected.Detected {
			req.Satisfied = true
			req.Message = "已检测外部 SteamCMD"
		} else {
			req.Message = "DST 安装/更新需要 SteamCMD"
			profile.Ready = false
		}
		profile.Requirements = []GameRuntimeRequirement{req}
		if runtime.GOOS == "linux" {
			profile.SystemPrerequisites = s.linuxSteamCMDPrerequisites()
			for _, p := range profile.SystemPrerequisites {
				if p.Required && !p.Detected {
					profile.Ready = false
				}
			}
		}
	default:
		profile.Name = request.GameID
		profile.Ready = false
		profile.Requirements = []GameRuntimeRequirement{{ID: "unknown-game", Required: true, Satisfied: false, Message: "未知 Game Runtime Profile"}}
	}
	return profile
}

func (s *Service) linuxSteamCMDPrerequisites() []SystemPrerequisite {
	if runtime.GOOS != "linux" {
		return nil
	}
	// steamcmd_linux is a 32-bit bootstrap on common amd64 distributions.
	// Detect representative loader/libstdc++ files instead of invoking a shell.
	if runtime.GOARCH != "amd64" {
		return []SystemPrerequisite{{ID: "steamcmd-linux-arch", Name: "SteamCMD Linux architecture", Required: true, Detected: false, Installable: false, Message: "官方 SteamCMD Linux 主要面向 amd64；当前架构需要管理员手动提供兼容 Runtime"}}
	}
	loader := firstExisting([]string{"/lib/ld-linux.so.2", "/lib32/ld-linux.so.2", "/lib/i386-linux-gnu/ld-linux.so.2", "/usr/lib32/ld-linux.so.2"})
	cpp := firstExisting([]string{"/usr/lib32/libstdc++.so.6", "/usr/lib/i386-linux-gnu/libstdc++.so.6", "/lib/i386-linux-gnu/libstdc++.so.6"})
	manager := detectPackageManager()
	installable := manager != ""
	p1 := SystemPrerequisite{ID: "steamcmd-linux-32bit-loader", Name: "32-bit glibc loader", Required: true, Detected: loader != "", Installable: installable, PackageManager: manager, Message: "SteamCMD Linux 需要 32 位运行时"}
	p2 := SystemPrerequisite{ID: "steamcmd-linux-32bit-cpp", Name: "32-bit libstdc++", Required: true, Detected: cpp != "", Installable: installable, PackageManager: manager, Message: "SteamCMD Linux 需要 32 位 C++ 运行库"}
	return []SystemPrerequisite{p1, p2}
}

func (s *Service) InstallSystemPrerequisite(ctx context.Context, request InstallSystemPrerequisiteRequest) (SystemPrerequisite, error) {
	if runtime.GOOS != "linux" {
		return SystemPrerequisite{}, errors.New("系统前置依赖自动安装当前只用于 Linux")
	}
	id := strings.TrimSpace(request.ID)
	allowed := map[string]bool{"steamcmd-linux-32bit-loader": true, "steamcmd-linux-32bit-cpp": true}
	if !allowed[id] {
		return SystemPrerequisite{}, errors.New("不允许安装未声明的系统依赖")
	}
	manager := detectPackageManager()
	if manager == "" {
		return SystemPrerequisite{}, errors.New("未检测到受支持的 Linux 包管理器")
	}
	exe := findExecutableInPATH(manager)
	if exe == "" {
		return SystemPrerequisite{}, errors.New("包管理器路径失效")
	}
	var args []string
	switch manager {
	case "apt-get":
		// dpkg multiarch enabling is intentionally NOT automated because it is
		// global system configuration. If i386 isn't enabled, apt will report it.
		args = []string{"install", "-y", "libc6:i386", "libstdc++6:i386"}
	case "dnf", "yum":
		args = []string{"install", "-y", "glibc.i686", "libstdc++.i686"}
	case "pacman":
		args = []string{"-S", "--noconfirm", "lib32-glibc", "lib32-gcc-libs"}
	default:
		return SystemPrerequisite{}, errors.New("不支持的包管理器")
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	result, err := platformruntime.Run(runCtx, platformruntime.RunSpec{Spec: platformruntime.Spec{Executable: exe, Arguments: args}, MaxOutputBytes: 2 << 20})
	if err != nil || result.ExitCode != 0 {
		return SystemPrerequisite{}, fmt.Errorf("安装 Linux 系统依赖失败（需要 root/sudo 权限）：%s", strings.TrimSpace(result.Stderr+"\n"+result.Stdout))
	}
	for _, item := range s.linuxSteamCMDPrerequisites() {
		if item.ID == id {
			return item, nil
		}
	}
	return SystemPrerequisite{}, errors.New("依赖安装完成但重新检测失败")
}

func firstExisting(paths []string) string {
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}
func detectPackageManager() string {
	for _, name := range []string{"apt-get", "dnf", "yum", "pacman"} {
		if findExecutableInPATH(name) != "" {
			return name
		}
	}
	return ""
}

// Keep filepath imported as part of the platform-safe package location checks.
var _ = filepath.Separator

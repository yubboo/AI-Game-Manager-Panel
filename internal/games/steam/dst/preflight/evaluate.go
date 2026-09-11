package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
)

type Input struct {
	Dedicated dedicated.Snapshot
	Cluster   *dstdomain.Cluster
	Wanted    string
	Token     token.Status
}

func Evaluate(input Input) Result {
	checks := make([]Check, 0, 10)
	if input.Dedicated.Installation.Valid {
		checks = append(checks, ok("dedicated", "Dedicated Server", fmt.Sprintf("%d-bit 专用服务器可用", input.Dedicated.Installation.Bitness)))
	} else {
		checks = append(checks, blocker("dedicated", "Dedicated Server", "未检测到有效的 DST Dedicated Server", "前往“安装与更新”完成安装或校验"))
	}
	if input.Dedicated.ConfDir.Valid {
		checks = append(checks, ok("conf_dir", "Klei 启动目录", input.Dedicated.ConfDir.KleiRoot))
	} else {
		checks = append(checks, blocker("conf_dir", "Klei 启动目录", input.Dedicated.ConfDir.Error, "检查 Windows 文档/Klei 路径"))
	}
	if input.Cluster == nil {
		checks = append(checks, blocker("cluster", "Cluster", "未找到所选 Cluster", "重新检测或导入世界"))
		return NewResult(input.Wanted, checks)
	}
	cluster := *input.Cluster
	if cluster.Distribution == dstdomain.DistributionSteam && cluster.Source == dstdomain.SaveSourceServer {
		checks = append(checks, ok("cluster", "Cluster", cluster.Name))
	} else {
		checks = append(checks, blocker("cluster", "Cluster", "当前只允许启动 Steam SERVER 类型 Cluster", "先把本地世界导入为 SERVER Cluster"))
	}
	clusterINI := filepath.Join(cluster.Path, "cluster.ini")
	if isRegular(clusterINI) {
		checks = append(checks, ok("cluster_ini", "cluster.ini", "服务器配置存在"))
		if dedicated.ConsoleEnabled(clusterINI) {
			checks = append(checks, ok("console", "控制台", "cluster.ini [MISC] console_enabled = true"))
		} else {
			checks = append(checks, Check{Code: "console", Label: "控制台", Severity: SeverityWarning, Message: "cluster.ini 尚未启用 console_enabled；AGMP 将在启动前自动补全", Action: "无需手工处理"})
		}
	} else {
		checks = append(checks, blocker("cluster_ini", "cluster.ini", "缺少 cluster.ini", "重新导入或修复 Cluster"))
	}
	masterINI := filepath.Join(cluster.Path, "Master", "server.ini")
	if hasShard(cluster, "Master") && isRegular(masterINI) {
		checks = append(checks, ok("master", "Master", "Master/server.ini 完整"))
	} else {
		checks = append(checks, blocker("master", "Master", "缺少 Master/server.ini", "修复地面 Shard 配置"))
	}
	if input.Token.Configured {
		checks = append(checks, ok("token", "服务器令牌", "cluster_token.txt 已配置（实际有效性由 Klei 启动认证确认）"))
	} else {
		message := input.Token.Message
		if strings.TrimSpace(message) == "" {
			message = "尚未配置 cluster_token.txt"
		}
		checks = append(checks, blocker("token", "服务器令牌", message, "前往“令牌管理”上传或粘贴 Klei 服务器令牌"))
	}
	cavesDir := filepath.Join(cluster.Path, "Caves")
	if info, err := os.Stat(cavesDir); err == nil && info.IsDir() {
		if isRegular(filepath.Join(cavesDir, "server.ini")) {
			checks = append(checks, ok("caves", "Caves", "Caves/server.ini 已检测到"))
		} else {
			checks = append(checks, Check{Code: "caves", Label: "Caves", Severity: SeverityWarning, Message: "存在 Caves 文件夹但缺少 server.ini；当前 Master 仍可启动", Action: "如需洞穴，请补齐 Caves 配置"})
		}
	}

	if topology, err := dedicated.InspectShardTopology(cluster.Path); err == nil && topology.HasCaves {
		if topology.Ready {
			checks = append(checks, ok("shard_topology", "地面 / 洞穴联动", topology.Summary()))
		} else {
			checks = append(checks, blocker("shard_topology", "地面 / 洞穴联动", topology.Summary(), "修复 cluster.ini 与 Master/Caves server.ini 的 Shard 配置"))
		}
	}

	if plan, err := dedicated.ReadPortPlan(cluster.Path); err == nil {
		hasCavesConfig := hasShard(cluster, "Caves") && isRegular(filepath.Join(cluster.Path, "Caves", "server.ini"))
		if hasCavesConfig && len(plan.Missing) > 0 {
			checks = append(checks, blocker("ports_explicit", "双 Shard 端口", "Master/Caves 联动开服要求显式配置全部端口："+strings.Join(plan.Missing, "；"), "前往“网络与端口”，一键应用 AGMP 推荐配置；高级用户也可以可视化自定义"))
		}
		if len(plan.Collisions) == 0 {
			message := "未发现 Master/Caves 配置端口重复"
			if len(plan.Ports()) > 0 {
				message = fmt.Sprintf("已规划 %d 个监听端口，配置之间无冲突", len(plan.Ports()))
			}
			checks = append(checks, ok("ports_config", "端口配置", message))
		} else {
			parts := make([]string, 0, len(plan.Collisions))
			for _, collision := range plan.Collisions {
				names := make([]string, 0, len(collision.Uses))
				for _, use := range collision.Uses {
					names = append(names, use.ShardName+"/"+string(use.Purpose))
				}
				parts = append(parts, fmt.Sprintf("%d (%s)", collision.Port, strings.Join(names, ", ")))
			}
			checks = append(checks, blocker("ports_config", "端口配置", "存在重复端口："+strings.Join(parts, "；"), "前往“网络与端口”修复重复端口，普通用户可直接一键恢复推荐配置"))
		}
		for _, warning := range plan.Warnings {
			checks = append(checks, Check{Code: "ports_warning", Label: "端口配置", Severity: SeverityWarning, Message: warning, Action: "前往“网络与端口”一键补全，或在高级设置中自定义"})
		}
	}
	return NewResult(cluster.Path, checks)
}

func ok(code, label, message string) Check {
	return Check{Code: code, Label: label, Severity: SeverityOK, Message: message}
}
func blocker(code, label, message, action string) Check {
	return Check{Code: code, Label: label, Severity: SeverityBlocker, Message: message, Action: action}
}

func isRegular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
func hasShard(cluster dstdomain.Cluster, wanted string) bool {
	for _, shard := range cluster.Shards {
		if strings.EqualFold(shard.Name, wanted) {
			return true
		}
	}
	return false
}

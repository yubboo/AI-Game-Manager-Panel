package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

func (a *Application) requireModelAdministrator(token string) error {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return err
	}
	if user.Role != authservice.RoleOwner && user.Role != authservice.RoleAdministrator {
		return authservice.ErrForbidden
	}
	return nil
}

func (a *Application) XiaoYuModelCatalog(token string) (xiaoyuhost.ModelCatalog, error) {
	if _, err := a.requireXiaoYuMember(token); err != nil {
		return xiaoyuhost.ModelCatalog{}, err
	}
	if a.xiaoyuModels == nil {
		return xiaoyuhost.ModelCatalog{}, errors.New("XiaoYu 模型管理器不可用")
	}
	value, err := a.xiaoyuModels.Catalog()
	if err != nil {
		return value, err
	}
	if a.xiaoyuHost != nil {
		if brain, ok := a.xiaoyuHost.Brain(); ok {
			info := brain.Info(a.Context())
			value.BrainReady = info.Ready
			if strings.TrimSpace(info.Message) != "" {
				value.Message = info.Message
			}
		}
	}
	return value, nil
}

func (a *Application) SaveXiaoYuModel(token string, request xiaoyuhost.SaveModelRequest) (xiaoyuhost.ModelProfileView, error) {
	if err := a.requireModelAdministrator(token); err != nil {
		return xiaoyuhost.ModelProfileView{}, err
	}
	if a.xiaoyuModels == nil {
		return xiaoyuhost.ModelProfileView{}, errors.New("XiaoYu 模型管理器不可用")
	}
	value, err := a.xiaoyuModels.Save(request)
	if err != nil {
		return value, err
	}
	message := "保存 XiaoYu 模型 Provider 配置"
	if value.AuthMode == xiaoyuhost.ModelAuthAPIKey {
		message += "（API Key 已进入独立 Secret Vault）"
	} else if value.AuthMode == xiaoyuhost.ModelAuthSubscription {
		message += "（使用官方订阅/登录状态，不复制 Provider 凭证）"
	}
	a.recordOperation("info", "xiaoyu", "save_model", value.Name, "success", message, "", "")
	a.observeXiaoYu("model/saved", "", map[string]any{"id": value.ID, "provider": value.Provider, "model": value.Model})
	return value, nil
}

func (a *Application) DeleteXiaoYuModel(token, id string) error {
	if err := a.requireModelAdministrator(token); err != nil {
		return err
	}
	if a.xiaoyuModels == nil {
		return errors.New("XiaoYu 模型管理器不可用")
	}
	id = strings.TrimSpace(id)
	if err := a.xiaoyuModels.Delete(id); err != nil {
		return err
	}
	a.recordOperation("warning", "xiaoyu", "delete_model", id, "success", "删除 XiaoYu 模型配置及对应 Secret", "", "")
	a.observeXiaoYu("model/deleted", "", map[string]any{"id": id})
	return nil
}

func (a *Application) SetXiaoYuDefaultModel(token, id string) (xiaoyuhost.ModelProfileView, error) {
	if err := a.requireModelAdministrator(token); err != nil {
		return xiaoyuhost.ModelProfileView{}, err
	}
	if a.xiaoyuModels == nil {
		return xiaoyuhost.ModelProfileView{}, errors.New("XiaoYu 模型管理器不可用")
	}
	value, err := a.xiaoyuModels.SetDefault(id)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "xiaoyu", "set_default_model", value.Name, "success", "设置 XiaoYu 默认大脑模型", "", "")
	a.observeXiaoYu("model/default", "", map[string]any{"id": value.ID, "provider": value.Provider, "model": value.Model})
	return value, nil
}

func (a *Application) TestXiaoYuModel(token string, request xiaoyuhost.ModelConnectionRequest) (xiaoyuhost.ModelConnectionResult, error) {
	if err := a.requireModelAdministrator(token); err != nil {
		return xiaoyuhost.ModelConnectionResult{}, err
	}
	profile, apiKey, err := a.xiaoyuModels.ResolveConnection(request)
	if err != nil {
		return xiaoyuhost.ModelConnectionResult{}, err
	}
	var result xiaoyuhost.ModelConnectionResult
	if profile.Protocol == xiaoyuhost.ProtocolCodexAppServer {
		result, err = testCodexSubscription(a.Context(), profile)
	} else {
		result, err = xiaoyuhost.TestModelConnection(a.Context(), profile, apiKey)
	}
	if strings.TrimSpace(request.ID) != "" {
		_ = a.xiaoyuModels.RecordTest(request.ID, result)
	}
	status := "success"
	if err != nil {
		status = "failed"
	}
	a.recordOperation("info", "xiaoyu", "test_model", profile.Provider+":"+profile.Model, status, "测试模型 Provider 连接（不记录 API Key / OAuth Token / CLI 凭证）", "", "")
	return result, err
}

func (a *Application) DiscoverXiaoYuModels(token string, request xiaoyuhost.ModelConnectionRequest) (xiaoyuhost.ModelConnectionResult, error) {
	if err := a.requireModelAdministrator(token); err != nil {
		return xiaoyuhost.ModelConnectionResult{}, err
	}
	profile, apiKey, err := a.xiaoyuModels.ResolveConnection(request)
	if err != nil {
		return xiaoyuhost.ModelConnectionResult{}, err
	}
	if profile.Protocol == xiaoyuhost.ProtocolCodexAppServer {
		return xiaoyuhost.ModelConnectionResult{OK: false, Message: "Codex 套餐 Provider 当前只接入官方 CLI 登录状态探测；模型枚举与 XiaoYu Brain Adapter 将在 app-server 安全中介完成后启用。", Endpoint: "codex://local", Capabilities: xiaoyuhost.ResolveModelCapabilities(profile)}, errors.New("Codex app-server 模型枚举尚未启用")
	}
	models, endpoint, err := xiaoyuhost.DiscoverModelNames(a.Context(), profile, apiKey)
	if err != nil {
		return xiaoyuhost.ModelConnectionResult{OK: false, Message: err.Error(), Endpoint: endpoint}, err
	}
	return xiaoyuhost.ModelConnectionResult{OK: true, Message: "已拉取模型列表", Endpoint: endpoint, Models: models}, nil
}

func testCodexSubscription(parent context.Context, profile xiaoyuhost.ModelProfile) (xiaoyuhost.ModelConnectionResult, error) {
	executable := xiaoyuhost.ModelProviderExecutable(profile)
	if strings.TrimSpace(executable) == "" {
		return xiaoyuhost.ModelConnectionResult{OK: false, Message: "Codex Provider 缺少官方 CLI 执行器", Endpoint: "codex://local", Capabilities: xiaoyuhost.ResolveModelCapabilities(profile)}, errors.New("Codex CLI 未配置")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	started := time.Now()
	result, runErr := platformruntime.Run(ctx, platformruntime.RunSpec{Spec: platformruntime.Spec{Executable: executable, Arguments: []string{"login", "status"}}, MaxOutputBytes: 64 << 10})
	latency := time.Since(started).Milliseconds()
	view := xiaoyuhost.ModelConnectionResult{Endpoint: "codex://local", LatencyMS: latency, Capabilities: xiaoyuhost.ResolveModelCapabilities(profile)}
	if runErr != nil || result.ExitCode != 0 {
		if ctx.Err() != nil {
			view.Message = "Codex CLI 登录状态检查超时"
			return view, ctx.Err()
		}
		view.Message = "未检测到可用的 Codex 套餐登录；请先通过官方 Codex CLI 完成登录"
		if runErr != nil {
			return view, fmt.Errorf("Codex CLI 登录状态检查失败: %w", runErr)
		}
		return view, fmt.Errorf("Codex CLI 登录状态检查失败，退出码 %d", result.ExitCode)
	}
	view.OK = true
	view.Message = "已通过官方 Codex CLI 确认订阅登录状态；AGMP 未读取或复制 Codex 凭证"
	return view, nil
}

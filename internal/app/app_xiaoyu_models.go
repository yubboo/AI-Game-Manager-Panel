package app

import (
	"errors"
	"strings"

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
	a.recordOperation("info", "xiaoyu", "save_model", value.Name, "success", "保存 XiaoYu 模型配置（API Key 已进入独立 Secret Vault）", "", "")
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
	result, err := xiaoyuhost.TestModelConnection(a.Context(), profile, apiKey)
	if strings.TrimSpace(request.ID) != "" {
		_ = a.xiaoyuModels.RecordTest(request.ID, result)
	}
	status := "success"
	if err != nil {
		status = "failed"
	}
	a.recordOperation("info", "xiaoyu", "test_model", profile.Provider+":"+profile.Model, status, "测试模型接口连接（不记录 API Key）", "", "")
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
	models, endpoint, err := xiaoyuhost.DiscoverModelNames(a.Context(), profile, apiKey)
	if err != nil {
		return xiaoyuhost.ModelConnectionResult{OK: false, Message: err.Error(), Endpoint: endpoint}, err
	}
	return xiaoyuhost.ModelConnectionResult{OK: true, Message: "已拉取模型列表", Endpoint: endpoint, Models: models}, nil
}

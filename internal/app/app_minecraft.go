package app

import (
	"context"
	"errors"

	minecraft "github.com/yubboo/AI-Game-Manager-Panel/internal/games/minecraft"
	serverinstance "github.com/yubboo/AI-Game-Manager-Panel/internal/server/instance"
)

func (a *Application) MinecraftPlan(request minecraft.PlanRequest) (minecraft.Plan, error) {
	if a == nil || a.minecraft == nil {
		return minecraft.Plan{}, errors.New("Minecraft Game Pack 未初始化")
	}
	ctx := a.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return a.minecraft.Plan(ctx, request)
}
func (a *Application) DeployMinecraft(request minecraft.PlanRequest) (minecraft.DeploymentResult, error) {
	if a == nil || a.minecraft == nil {
		return minecraft.DeploymentResult{}, errors.New("Minecraft Game Pack 未初始化")
	}
	ctx := a.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.minecraft.Deploy(ctx, request)
	if err == nil {
		a.recordOperation("info", "minecraft", "deploy", value.Instance.ID, "success", "Minecraft 实例已通过共享部署内核创建", "", "")
	}
	return value, err
}
func (a *Application) StartMinecraft(id string) (minecraft.RuntimeSnapshot, error) {
	if a == nil || a.minecraft == nil {
		return minecraft.RuntimeSnapshot{}, errors.New("Minecraft Game Pack 未初始化")
	}
	ctx := a.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return a.minecraft.Start(ctx, id)
}
func (a *Application) StopMinecraft(id string) (minecraft.RuntimeSnapshot, error) {
	if a == nil || a.minecraft == nil {
		return minecraft.RuntimeSnapshot{}, errors.New("Minecraft Game Pack 未初始化")
	}
	return a.minecraft.Stop(id)
}
func (a *Application) MinecraftStatus(id string) minecraft.RuntimeSnapshot {
	if a == nil || a.minecraft == nil {
		return minecraft.RuntimeSnapshot{InstanceID: id, State: "unavailable"}
	}
	return a.minecraft.Status(id)
}
func (a *Application) MinecraftLogs(id string, after uint64, limit int) minecraft.LogBatch {
	if a == nil || a.minecraft == nil {
		return minecraft.LogBatch{Lines: []minecraft.LogLine{}, NextCursor: after}
	}
	return a.minecraft.Logs(id, after, limit)
}
func (a *Application) ProbeMinecraft(id string) (minecraft.ProbeResult, error) {
	if a == nil || a.minecraft == nil {
		return minecraft.ProbeResult{}, errors.New("Minecraft Game Pack 未初始化")
	}
	ctx := a.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return a.minecraft.ProbeInstance(ctx, id)
}

type gameDeployRequest struct {
	GameID           string             `json:"gameId"`
	Name             string             `json:"name"`
	Version          string             `json:"version,omitempty"`
	Software         minecraft.Software `json:"software"`
	MemoryMB         int                `json:"memoryMb"`
	Port             int                `json:"port"`
	OnlineMode       *bool              `json:"onlineMode"`
	Whitelist        bool               `json:"whitelist"`
	EULAAccepted     bool               `json:"eulaAccepted"`
	AutoInstallJava  bool               `json:"autoInstallJava"`
	StartAfterDeploy bool               `json:"startAfterDeploy"`
}

func (r gameDeployRequest) minecraft(origin serverinstance.Origin) minecraft.PlanRequest {
	return minecraft.PlanRequest{Name: r.Name, Version: r.Version, Software: r.Software, MemoryMB: r.MemoryMB, Port: r.Port, OnlineMode: r.OnlineMode, Whitelist: r.Whitelist, EULAAccepted: r.EULAAccepted, AutoInstallJava: r.AutoInstallJava, StartAfterDeploy: r.StartAfterDeploy, Origin: origin}
}

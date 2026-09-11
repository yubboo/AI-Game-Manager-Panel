package maintenance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam/protocol"
)

func New(steamPlatform SteamPlatform, launcher Launcher) *Service {
	if launcher == nil {
		launcher = protocol.NewSystemLauncher()
	}
	return &Service{steam: steamPlatform, launcher: launcher, tasks: map[string]TaskSnapshot{}}
}

func (s *Service) StartValidate(ctx context.Context, appID uint32) (TaskSnapshot, error) {
	if appID == 0 {
		return TaskSnapshot{}, errors.New("Steam AppID 不能为空")
	}
	if existing, ok := s.activeTask(appID, OperationValidate); ok {
		return existing, nil
	}
	env, err := s.steam.Detect(ctx)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if !env.Detected || strings.TrimSpace(env.InstallPath) == "" {
		return TaskSnapshot{}, errors.New("未检测到 Steam 官方客户端")
	}
	before, found, err := s.steam.FindApp(ctx, steam.AppID(appID))
	if err != nil {
		return TaskSnapshot{}, err
	}
	if !found || !before.InstallPathExists {
		return TaskSnapshot{}, fmt.Errorf("Steam AppID %d 尚未安装，无法执行文件校验", appID)
	}

	uri := protocol.BuildValidateURI(appID)
	cursor := steam.NewContentLogCursor(env.InstallPath)
	ioTracker := newValidationIOTracker()
	if err := s.launcher.Open(ctx, uri); err != nil {
		return TaskSnapshot{}, err
	}

	task := newTask(appID, OperationValidate, uri, before.BuildID)
	applyTaskPaths(&task, env, before, cursor)
	task.Message = "已向 Steam 发出校验请求，等待 Steam 真正进入验证流程"
	s.put(task)
	go s.monitor(task.ID, before, cursor, ioTracker)
	return task, nil
}

func (s *Service) StartInstall(ctx context.Context, appID uint32) (TaskSnapshot, error) {
	if appID == 0 {
		return TaskSnapshot{}, errors.New("Steam AppID 不能为空")
	}
	if existing, ok := s.activeTask(appID, OperationInstall); ok {
		return existing, nil
	}
	env, err := s.steam.Detect(ctx)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if !env.Detected || strings.TrimSpace(env.InstallPath) == "" {
		return TaskSnapshot{}, errors.New("未检测到 Steam 官方客户端，请先安装并登录 Steam")
	}
	before, found, err := s.steam.FindApp(ctx, steam.AppID(appID))
	if err != nil {
		return TaskSnapshot{}, err
	}
	if found && before.InstallPathExists {
		return TaskSnapshot{}, fmt.Errorf("Steam AppID %d 已安装，无需重复安装", appID)
	}

	uri := protocol.BuildInstallURI(appID)
	cursor := steam.NewContentLogCursor(env.InstallPath)
	if err := s.launcher.Open(ctx, uri); err != nil {
		return TaskSnapshot{}, err
	}

	task := newTask(appID, OperationInstall, uri, before.BuildID)
	applyTaskPaths(&task, env, before, cursor)
	task.Message = "已调用用户自己的 Steam 客户端，请在 Steam 中确认安装位置；AI Game Manager Panel 将持续监控真实安装状态"
	s.put(task)
	go s.monitor(task.ID, before, cursor, validationIOTracker{})
	return task, nil
}

func newTask(appID uint32, operation, uri, buildID string) TaskSnapshot {
	now := time.Now().Unix()
	return TaskSnapshot{
		ID: fmt.Sprintf("steam-%s-%d-%d", operation, appID, time.Now().UnixNano()), AppID: appID,
		Operation: operation, URI: uri, State: StateRequested, Phase: PhaseWaiting,
		StartedAt: now, UpdatedAt: now, BuildID: buildID, Progress: 0,
		ProgressMode: ProgressIndeterminate,
	}
}

func applyTaskPaths(task *TaskSnapshot, env steam.Environment, app steam.AppInstallation, cursor steam.ContentLogCursor) {
	task.SteamRoot = env.InstallPath
	task.LibraryPath = app.LibraryPath
	task.InstallPath = app.InstallPath
	task.ManifestPath = app.ManifestPath
	task.ContentLogPath = cursor.Path
	task.StateFlags = app.StateFlags
}

func (s *Service) Get(id string) (TaskSnapshot, bool) {
	s.mu.RLock()
	task, ok := s.tasks[id]
	s.mu.RUnlock()
	return task, ok
}

func (s *Service) activeTask(appID uint32, operation string) (TaskSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var newest TaskSnapshot
	found := false
	for _, task := range s.tasks {
		if task.AppID != appID || task.Operation != operation {
			continue
		}
		if task.State != StateRequested && task.State != StateMonitoring {
			continue
		}
		if !found || task.StartedAt > newest.StartedAt || (task.StartedAt == newest.StartedAt && task.ID > newest.ID) {
			newest = task
			found = true
		}
	}
	return newest, found
}

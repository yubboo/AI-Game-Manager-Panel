package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	environmentservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
	steammaintenance "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/steam/maintenance"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/logcenter"
	dstprefs "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	dstruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"

	clusterops "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/cluster"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/kleiarchive"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
	globallogs "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/logs"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	licenseservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/license"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/system/settings"
	xiaoyucontrol "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/control"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

// Options configures the browser/Web adapter. The HTTP bridge intentionally
// contains no DST, Steam, filesystem or process business logic.
type Options struct {
	ListenAddr  string
	Assets      fs.FS
	AllowRemote bool
}

// Server exposes the same Application used by the Wails desktop adapter.
// Desktop and Web therefore share one business contract instead of maintaining
// two independent implementations.
type Server struct {
	application *application.Application
	httpServer  *http.Server
	assets      fs.FS
	allowRemote bool
	authLimiter *authRateLimiter
}

func New(application *application.Application, options Options) *Server {
	addr := strings.TrimSpace(options.ListenAddr)
	if addr == "" {
		addr = "127.0.0.1:17890"
	}
	return &Server{
		application: application,
		assets:      options.Assets,
		allowRemote: options.AllowRemote,
		authLimiter: newAuthRateLimiter(),
		httpServer: &http.Server{
			Addr:              addr,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
	}
}

func (s *Server) Addr() string { return s.httpServer.Addr }

// ListenAndServe serves the same AGMP control plane used by desktop clients.
// Loopback is the secure default and is ideal behind a local Caddy/Nginx reverse
// proxy. Non-loopback binding is allowed only after the administrator explicitly
// enables server.allowRemoteWeb; authentication/RBAC/approval still apply.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if err := validateListenPolicy(s.httpServer.Addr, s.allowRemote); err != nil {
		return err
	}

	s.httpServer.Handler = s.routes()
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()

	err = s.httpServer.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func validateListenPolicy(addr string, allowRemote bool) error {
	if allowRemote {
		return nil
	}
	if err := requireLoopback(addr); err != nil {
		return fmt.Errorf("远程 Web 监听未显式启用: %w", err)
	}
	return nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/updater/check", func(w http.ResponseWriter, r *http.Request) {
		force := r.URL.Query().Get("force") == "1" || strings.EqualFold(r.URL.Query().Get("force"), "true")
		status, err := s.application.CheckForUpdates(force)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("GET /api/v1/license/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.LicenseStatus())
	})
	mux.HandleFunc("POST /api/v1/license/activate", func(w http.ResponseWriter, r *http.Request) {
		var request licenseservice.ActivateRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "许可证激活信息格式无效")
			return
		}
		status, err := s.application.ActivateLicenseForUser(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("POST /api/v1/license/unbind", func(w http.ResponseWriter, r *http.Request) {
		status, err := s.application.UnbindLicenseForUser(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("GET /api/v1/license/features/{feature}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.application.LicenseFeatureStatus(strings.TrimSpace(r.PathValue("feature"))))
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/runtime", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.XiaoYuRuntimeStatus())
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/tools", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuTools(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/capabilities", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuCapabilities(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/harness", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuHarnessStatus(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/models", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuModelCatalog(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/models", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.SaveModelRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "模型配置格式无效")
			return
		}
		value, err := s.application.SaveXiaoYuModel(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/xiaoyu/models/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.application.DeleteXiaoYuModel(bearerToken(r), strings.TrimSpace(r.PathValue("id"))); err != nil {
			writeAuthError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PUT /api/v1/xiaoyu/models/{id}/default", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.SetXiaoYuDefaultModel(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/models/test", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.ModelConnectionRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "模型连接测试请求无效")
			return
		}
		value, err := s.application.TestXiaoYuModel(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/models/discover", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.ModelConnectionRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "模型发现请求无效")
			return
		}
		value, err := s.application.DiscoverXiaoYuModels(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/intelligence", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuIntelligenceCatalog(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/intelligence/memories", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.MemorySaveRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "XiaoYu Memory 请求格式无效")
			return
		}
		value, err := s.application.SaveXiaoYuMemory(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/intelligence/skills", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.SkillSaveRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 192<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "XiaoYu Skill 请求格式无效")
			return
		}
		value, err := s.application.SaveXiaoYuSkill(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/intelligence/experts", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.ExpertSaveRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 384<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "XiaoYu Expert 请求格式无效")
			return
		}
		value, err := s.application.SaveXiaoYuExpert(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/xiaoyu/intelligence/{kind}/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.application.DeleteXiaoYuIntelligence(bearerToken(r), strings.TrimSpace(r.PathValue("kind")), strings.TrimSpace(r.PathValue("id"))); err != nil {
			writeAuthError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/runs", func(w http.ResponseWriter, r *http.Request) {
		var request application.XiaoYuRunRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "XiaoYu 自主任务请求格式无效")
			return
		}
		value, err := s.application.XiaoYuStartRun(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuRunState(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/runs/{id}/continue", func(w http.ResponseWriter, r *http.Request) {
		var request application.XiaoYuContinueRunRequest
		if r.ContentLength != 0 {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "XiaoYu 继续任务请求格式无效")
				return
			}
		}
		value, err := s.application.XiaoYuContinueRun(bearerToken(r), strings.TrimSpace(r.PathValue("id")), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/runs/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuCancelRun(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/runs/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
		var request application.XiaoYuControlRequest
		if r.ContentLength != 0 {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "XiaoYu 暂停请求格式无效")
				return
			}
		}
		value, err := s.application.XiaoYuPauseRun(bearerToken(r), strings.TrimSpace(r.PathValue("id")), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/runs/{id}/takeover", func(w http.ResponseWriter, r *http.Request) {
		var request application.XiaoYuControlRequest
		if r.ContentLength != 0 {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "人工接管请求格式无效")
				return
			}
		}
		value, err := s.application.XiaoYuTakeoverRun(bearerToken(r), strings.TrimSpace(r.PathValue("id")), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/runs/{id}/resume", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuResumeRun(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/trace", func(w http.ResponseWriter, r *http.Request) {
		after, _ := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("after")), 10, 64)
		limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
		value, err := s.application.XiaoYuTrace(bearerToken(r), after, limit)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/events", func(w http.ResponseWriter, r *http.Request) {
		after, _ := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("after")), 10, 64)
		stream, cancel, err := s.application.XiaoYuEventStream(bearerToken(r), after)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		defer cancel()
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "当前 HTTP Writer 不支持事件流")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-r.Context().Done():
				return // 只断开浏览器订阅；服务器端 XiaoYu Run 不受影响。
			case <-heartbeat.C:
				_, _ = fmt.Fprint(w, ": xiaoyu-heartbeat\n\n")
				flusher.Flush()
			case event, ok := <-stream:
				if !ok {
					return
				}
				payload, err := json.Marshal(event)
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(w, "id: %d\nevent: xiaoyu\ndata: %s\n\n", event.Sequence, payload)
				flusher.Flush()
			}
		}
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/plugins/dsh", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuDSHPlugins(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/plugins/dsh/mount", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuhost.DSHMountRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DSH 插件挂载请求格式无效")
			return
		}
		value, err := s.application.MountXiaoYuDSHPlugin(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/xiaoyu/plugins/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.application.UnmountXiaoYuPlugin(bearerToken(r), strings.TrimSpace(r.PathValue("id"))); err != nil {
			writeAuthError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/approval", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuApprovalState(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("PUT /api/v1/xiaoyu/approval/mode", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyucontrol.SetModeRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "小鱼审批模式请求格式无效")
			return
		}
		value, err := s.application.SetXiaoYuApprovalMode(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/xiaoyu/approvals", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.XiaoYuPendingApprovals(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/approvals/{id}", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyucontrol.ResolveRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "小鱼审批决定格式无效")
			return
		}
		value, err := s.application.ResolveXiaoYuApproval(bearerToken(r), strings.TrimSpace(r.PathValue("id")), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/tool", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuruntime.ToolCallRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "小鱼 Tool 请求格式无效")
			return
		}
		value, err := s.application.XiaoYuCallTool(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/xiaoyu/command", func(w http.ResponseWriter, r *http.Request) {
		var request xiaoyuruntime.CommandRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "小鱼命令请求格式无效")
			return
		}
		value, err := s.application.XiaoYuRunCommand(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/auth/bootstrap", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.AuthBootstrapStatus())
	})
	mux.HandleFunc("POST /api/v1/auth/security-key/generate", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "security-key-generate", 20, time.Minute) {
			return
		}
		material, err := s.application.GenerateSecurityKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, material)
	})
	mux.HandleFunc("POST /api/v1/auth/bootstrap/owner", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "bootstrap-owner", 5, 5*time.Minute) {
			return
		}
		var request authservice.CreateOwnerRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "最高管理员注册信息格式无效")
			return
		}
		session, err := s.application.CreateInitialAdministrator(request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		s.writeAuthenticatedSession(w, r, http.StatusCreated, session)
	})
	mux.HandleFunc("POST /api/v1/auth/invitation/inspect", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "invitation-inspect", 30, time.Minute) {
			return
		}
		var request struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邀请校验信息格式无效")
			return
		}
		value, err := s.application.InspectUserInvitation(strings.TrimSpace(request.Token))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/auth/invitation/register", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "invitation-register", 10, 5*time.Minute) {
			return
		}
		var request authservice.RegisterInvitationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "受邀注册信息格式无效")
			return
		}
		session, err := s.application.RegisterInvitedUser(request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		s.writeAuthenticatedSession(w, r, http.StatusCreated, session)
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "login", 12, 5*time.Minute) {
			return
		}
		var request authservice.LoginRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "登录信息格式无效")
			return
		}
		session, err := s.application.Login(request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		s.writeAuthenticatedSession(w, r, http.StatusOK, session)
	})
	mux.HandleFunc("POST /api/v1/auth/password-reset/request", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "password-reset-request", 6, 15*time.Minute) {
			return
		}
		var request authservice.RequestPasswordResetRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "密码找回请求格式无效")
			return
		}
		value, _ := s.application.RequestPasswordReset(request)
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/auth/password-reset/confirm", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "password-reset-confirm", 10, 15*time.Minute) {
			return
		}
		var request authservice.ConfirmPasswordResetRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "密码重置请求格式无效")
			return
		}
		if err := s.application.ConfirmPasswordReset(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/auth/organization", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.AuthOrganization(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		user, err := s.application.ValidateSession(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, user)
	})
	mux.HandleFunc("PUT /api/v1/auth/me/display-name", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.UpdateDisplayNameRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "显示名称信息格式无效")
			return
		}
		user, err := s.application.UpdateMyDisplayName(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, user)
	})
	mux.HandleFunc("PUT /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.UpdateDisplayNameRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "显示名称信息格式无效")
			return
		}
		user, err := s.application.UpdateMyDisplayName(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, user)
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		s.application.Logout(bearerToken(r))
		s.clearAuthenticatedSession(w, r)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		users, err := s.application.ListUsers(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, users)
	})
	mux.HandleFunc("GET /api/v1/users/invitations", func(w http.ResponseWriter, r *http.Request) {
		values, err := s.application.ListUserInvitations(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, values)
	})
	mux.HandleFunc("POST /api/v1/users/invitations", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.CreateInvitationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邀请信息格式无效")
			return
		}
		value, err := s.application.CreateUserInvitation(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	mux.HandleFunc("DELETE /api/v1/users/invitations/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.RevokeUserInvitation(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.application.RemoveOrganizationMember(bearerToken(r), strings.TrimSpace(r.PathValue("id"))); err != nil {
			writeAuthError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/users/{id}/risk/clear", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.ClearMemberRisk(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/users/{id}/core/revoke", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.RevokeMemberCoreAccess(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/users/core-authorizations", func(w http.ResponseWriter, r *http.Request) {
		values, err := s.application.ListMemberCoreAuthorizations(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, values)
	})
	mux.HandleFunc("POST /api/v1/users/core-authorizations", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.CreateMemberAuthorizationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "成员授权请求格式无效")
			return
		}
		value, err := s.application.CreateMemberCoreAuthorization(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	mux.HandleFunc("DELETE /api/v1/users/core-authorizations/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.RevokeMemberCoreAuthorization(bearerToken(r), strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/auth/core/redeem", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.RedeemMemberAuthorizationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "成员核心授权码格式无效")
			return
		}
		value, err := s.application.RedeemMyCoreAuthorization(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/auth/email/status", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.MyEmailSecurityStatus(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("PUT /api/v1/auth/email", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.BindEmailRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邮箱绑定请求格式无效")
			return
		}
		value, err := s.application.BindMyEmail(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/auth/email", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邮箱解绑请求格式无效")
			return
		}
		value, err := s.application.UnbindMyEmail(bearerToken(r), request.Password)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/auth/email/verification/request", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "email-verification-request", 8, 15*time.Minute) {
			return
		}
		var request authservice.RequestEmailVerificationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邮箱验证请求格式无效")
			return
		}
		if err := s.application.RequestMyEmailVerification(bearerToken(r), request); err != nil {
			writeAuthError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/auth/email/verification/confirm", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "email-verification-confirm", 12, 15*time.Minute) {
			return
		}
		var request authservice.ConfirmEmailVerificationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邮箱验证码格式无效")
			return
		}
		value, err := s.application.ConfirmMyEmailVerification(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/auth/step-up/credentials", func(w http.ResponseWriter, r *http.Request) {
		if !s.allowPublicAuth(w, r, "credential-stepup", 12, 15*time.Minute) {
			return
		}
		var request authservice.CredentialStepUpRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "二次验证凭据格式无效")
			return
		}
		value, err := s.application.ConfirmMyCredentialStepUp(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/settings/email", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.SMTPSettings(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("PUT /api/v1/settings/email", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.SaveSMTPSettingsRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "邮箱服务配置格式无效")
			return
		}
		value, err := s.application.SaveSMTPSettings(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/auth/security-key/status", func(w http.ResponseWriter, r *http.Request) {
		status, err := s.application.MySecurityStatus(bearerToken(r))
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("POST /api/v1/auth/security-key/rotate", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.RotateSecurityKeyRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "安全密钥重置信息格式无效")
			return
		}
		material, err := s.application.RotateMySecurityKey(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, material)
	})
	mux.HandleFunc("PUT /api/v1/auth/security-key/verification", func(w http.ResponseWriter, r *http.Request) {
		var request authservice.SetSecurityKeyVerificationRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "登录安全密钥设置格式无效")
			return
		}
		status, err := s.application.SetMySecurityKeyVerification(bearerToken(r), request)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	})

	mux.HandleFunc("GET /api/v1/environment/setup", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.EnvironmentSetupStatus())
	})
	mux.HandleFunc("POST /api/v1/environment/setup", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.InitializeRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "环境初始化信息格式无效")
			return
		}
		status, err := s.application.InitializeEnvironment(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("POST /api/v1/environment/setup/skip", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		status, err := s.application.SkipEnvironmentSetup()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("POST /api/v1/environment/steamcmd/install", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.SteamCMDInstallRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "SteamCMD 安装目录格式无效")
				return
			}
		}
		status, err := s.application.InstallSteamCMDAt(request)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("PUT /api/v1/environment/paths", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.StoragePathsRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "运行环境路径格式无效")
			return
		}
		status, err := s.application.UpdateEnvironmentPaths(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("POST /api/v1/environment/steamcmd/migrate", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.MigratePathRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "SteamCMD 迁移参数无效")
			return
		}
		result, err := s.application.MigrateSteamCMD(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	// Environment Manager v2: runtime registry, Java versions and game profiles.
	mux.HandleFunc("GET /api/v1/environment/runtime/catalog", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.EnvironmentRuntimeCatalog())
	})
	mux.HandleFunc("POST /api/v1/environment/runtime/java/install", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.JavaInstallRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Java Runtime 安装参数无效")
			return
		}
		value, err := s.application.InstallJavaRuntime(request)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/environment/runtime/register", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.RegisterRuntimeRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Runtime 登记参数无效")
			return
		}
		value, err := s.application.RegisterEnvironmentRuntime(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/environment/runtime/resolve", func(w http.ResponseWriter, r *http.Request) {
		var request environmentservice.ResolveRuntimeRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Runtime 解析参数无效")
			return
		}
		value, err := s.application.ResolveEnvironmentRuntime(request)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("PUT /api/v1/environment/runtime/default", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.SetDefaultRuntimeRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "默认 Runtime 参数无效")
			return
		}
		value, err := s.application.SetDefaultEnvironmentRuntime(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/environment/runtime/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		if err := s.application.RemoveEnvironmentRuntime(environmentservice.RemoveRuntimeRequest{ID: strings.TrimSpace(r.PathValue("id"))}); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/environment/games/profiles", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.EnvironmentGameRuntimeProfiles())
	})
	mux.HandleFunc("POST /api/v1/environment/games/profile", func(w http.ResponseWriter, r *http.Request) {
		var request environmentservice.GameRuntimeProfileRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Game Runtime Profile 参数无效")
			return
		}
		writeJSON(w, http.StatusOK, s.application.EnvironmentGameRuntimeProfile(request))
	})
	mux.HandleFunc("POST /api/v1/environment/system-prerequisites/install", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.InstallSystemPrerequisiteRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "系统依赖安装参数无效")
			return
		}
		value, err := s.application.InstallEnvironmentSystemPrerequisite(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})

	mux.HandleFunc("POST /api/v1/environment/games/migrate", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request environmentservice.MigratePathRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "游戏服务器库迁移参数无效")
			return
		}
		result, err := s.application.MigrateGameLibrary(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": s.application.Ping()})
	})
	mux.HandleFunc("GET /api/v1/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.Info())
	})
	mux.HandleFunc("GET /api/v1/platform/config", func(w http.ResponseWriter, _ *http.Request) {
		value, err := s.application.PlatformConfig()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/platform/runtime", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.PlatformRuntimeContract())
	})
	mux.HandleFunc("GET /api/v1/game-packs", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.ValidateSession(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, s.application.GamePacks())
	})
	mux.HandleFunc("GET /api/v1/instances", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.ValidateSession(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, s.application.GameInstances())
	})
	mux.HandleFunc("GET /api/v1/settings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.Settings())
	})
	mux.HandleFunc("PUT /api/v1/settings", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var value settings.Settings
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&value); err != nil {
			writeError(w, http.StatusBadRequest, "设置数据格式无效")
			return
		}
		if err := s.application.SaveSettings(value); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		writeJSON(w, http.StatusOK, s.application.RecentLogs(limit))
	})
	mux.HandleFunc("POST /api/v1/loghub/catalog", func(w http.ResponseWriter, r *http.Request) {
		var request globallogs.CatalogRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "日志目录筛选格式无效")
			return
		}
		value, err := s.application.GlobalLogCatalog(request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/loghub/read", func(w http.ResponseWriter, r *http.Request) {
		var request globallogs.ReadRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "日志读取请求格式无效")
			return
		}
		value, err := s.application.GlobalLogRead(request)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/loghub/logs/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.ExportGlobalLog(strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/loghub/export", func(w http.ResponseWriter, r *http.Request) {
		var request globallogs.CatalogRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "日志导出筛选格式无效")
			return
		}
		value, err := s.application.ExportGlobalLogs(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/loghub/logs/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		value, err := s.application.DeleteGlobalLog(strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/loghub/delete-filtered", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		var request globallogs.DeleteFilteredRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "日志批量删除筛选格式无效")
			return
		}
		value, err := s.application.DeleteGlobalLogs(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("DELETE /api/v1/loghub/history", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		value, err := s.application.ClearGlobalLogHistory()
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/loghub/open-folder", func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.application.RequireOrganizationAdministrator(bearerToken(r)); err != nil {
			writeAuthError(w, err)
			return
		}
		if err := s.application.OpenGlobalLogFolder(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/time", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"value": s.application.CurrentTime()})
	})
	mux.HandleFunc("GET /api/v1/steam/environment", func(w http.ResponseWriter, _ *http.Request) {
		value, err := s.application.SteamEnvironment()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/steam/apps", func(w http.ResponseWriter, _ *http.Request) {
		value, err := s.application.SteamApps()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/steam/snapshot", func(w http.ResponseWriter, _ *http.Request) {
		value, err := s.application.SteamSnapshot()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/steam/maintenance/install", func(w http.ResponseWriter, r *http.Request) {
		var request steammaintenance.InstallRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Steam 安装请求格式无效")
			return
		}
		value, err := s.application.StartSteamInstall(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, value)
	})
	mux.HandleFunc("POST /api/v1/steam/maintenance/validate", func(w http.ResponseWriter, r *http.Request) {
		var request steammaintenance.ValidateRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Steam 校验请求格式无效")
			return
		}
		value, err := s.application.StartSteamValidation(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, value)
	})
	mux.HandleFunc("GET /api/v1/steam/maintenance/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.SteamMaintenanceTask(strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/games/{id}/snapshot", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			writeError(w, http.StatusBadRequest, "缺少游戏 ID")
			return
		}
		value, err := s.application.GameWorkspaceSnapshot(id)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/dst/environment", func(w http.ResponseWriter, _ *http.Request) {
		value := s.application.DSTEnvironment()
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/workspace", func(w http.ResponseWriter, _ *http.Request) {
		value, err := s.application.DSTWorkspaceSnapshot()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/dst/dedicated", func(w http.ResponseWriter, _ *http.Request) {
		value, err := s.application.DSTDedicatedServer()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/dst/dedicated/preferences", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.application.DSTDedicatedPreferences())
	})
	mux.HandleFunc("PUT /api/v1/dst/dedicated/preferences", func(w http.ResponseWriter, r *http.Request) {
		var value dstprefs.Value
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&value); err != nil {
			writeError(w, http.StatusBadRequest, "DST 专用服务器偏好格式无效")
			return
		}
		if err := s.application.SaveDSTDedicatedPreferences(value); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/dst/token/page", func(w http.ResponseWriter, _ *http.Request) {
		if err := s.application.OpenDSTTokenPage(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/dst/token/status", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.DSTTokenStatus(r.URL.Query().Get("clusterPath"))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("PUT /api/v1/dst/token", func(w http.ResponseWriter, r *http.Request) {
		var request token.SaveRequest
		// Token bodies are deliberately capped and never logged by this adapter.
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 令牌请求格式无效")
			return
		}
		value, err := s.application.SaveDSTToken(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/token/import", func(w http.ResponseWriter, r *http.Request) {
		var request token.ImportRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 令牌导入请求格式无效")
			return
		}
		value, err := s.application.ImportDSTToken(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/klei-package/inspect", func(w http.ResponseWriter, r *http.Request) {
		var request kleiarchive.InspectRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Klei 配置包检查请求格式无效")
			return
		}
		value, err := s.application.InspectDSTKleiPackage(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/klei-package/import", func(w http.ResponseWriter, r *http.Request) {
		var request kleiarchive.ImportRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "Klei 配置包导入请求格式无效")
			return
		}
		value, err := s.application.ImportDSTKleiPackage(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/dst/preflight", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.DSTPreflight(r.URL.Query().Get("clusterPath"))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/clusters/import", func(w http.ResponseWriter, r *http.Request) {
		var request clusterops.ImportRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST Cluster 导入请求格式无效")
			return
		}
		value, err := s.application.ImportDSTCluster(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})

	mux.HandleFunc("POST /api/v1/dst/launch-spec", func(w http.ResponseWriter, r *http.Request) {
		var request dedicated.LaunchRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 启动参数请求格式无效")
			return
		}
		value, err := s.application.BuildDSTLaunchSpec(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})

	mux.HandleFunc("POST /api/v1/dst/runtime/master/start", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.StartMasterRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST Master 启动请求格式无效")
			return
		}
		value, err := s.application.StartDSTMaster(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/cluster/start", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.StartClusterRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST Cluster 启动请求格式无效")
			return
		}
		value, err := s.application.StartDSTCluster(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/shard/start", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.StartShardRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST Shard 启动请求格式无效")
			return
		}
		value, err := s.application.StartDSTShard(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/cluster/status", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.ClusterRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST Cluster 状态请求格式无效")
			return
		}
		value, err := s.application.DSTClusterStatus(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/cluster/stop", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.ClusterRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST Cluster 停止请求格式无效")
			return
		}
		value, err := s.application.StopDSTCluster(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/ports/status", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.ClusterRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 端口状态请求格式无效")
			return
		}
		value, err := s.application.DSTPortStatus(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/ports/configuration", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.ClusterRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 端口配置读取请求格式无效")
			return
		}
		value, err := s.application.DSTPortConfiguration(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("PUT /api/v1/dst/runtime/ports/configuration", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.PortConfigureRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 端口配置保存请求格式无效")
			return
		}
		value, err := s.application.ConfigureDSTPorts(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/ports/cleanup", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.PortCleanupRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 端口清理请求格式无效")
			return
		}
		value, err := s.application.CleanupDSTPorts(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})

	mux.HandleFunc("POST /api/v1/dst/runtime/status", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.ProcessRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 进程状态请求格式无效")
			return
		}
		writeJSON(w, http.StatusOK, s.application.DSTProcessStatus(request))
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/logs", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.LogRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 进程日志请求格式无效")
			return
		}
		value, err := s.application.DSTProcessLogs(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/command", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.CommandRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 控制台命令格式无效")
			return
		}
		value, err := s.application.SendDSTCommand(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/runtime/stop", func(w http.ResponseWriter, r *http.Request) {
		var request dstruntime.ProcessRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 停止请求格式无效")
			return
		}
		value, err := s.application.StopDSTProcess(request)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})

	mux.HandleFunc("POST /api/v1/dst/logs/sessions", func(w http.ResponseWriter, r *http.Request) {
		var request logcenter.ListRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 日志列表请求格式无效")
			return
		}
		value, err := s.application.DSTLogSessions(request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/logs/read", func(w http.ResponseWriter, r *http.Request) {
		var request logcenter.ReadRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 日志读取请求格式无效")
			return
		}
		value, err := s.application.DSTLogRead(request)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/logs/tail", func(w http.ResponseWriter, r *http.Request) {
		var request logcenter.TailRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 日志尾部读取请求格式无效")
			return
		}
		value, err := s.application.DSTLogTail(request)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/logs/search", func(w http.ResponseWriter, r *http.Request) {
		var request logcenter.SearchRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 日志搜索请求格式无效")
			return
		}
		value, err := s.application.DSTLogSearch(request)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("GET /api/v1/dst/logs/sessions/{id}/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		value, err := s.application.DSTLogDiagnostics(strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	mux.HandleFunc("POST /api/v1/dst/logs/sessions/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		result, err := s.application.ExportDSTLog(strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("GET /api/v1/dst/logs/sessions/{id}/download", func(w http.ResponseWriter, r *http.Request) {
		ref, err := s.application.DSTLogFile(strings.TrimSpace(r.PathValue("id")))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		file, err := os.Open(ref.Path)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		defer file.Close()
		setAttachmentHeader(w, ref.Name, "AI Game Manager Panel-DST.log")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeContent(w, r, ref.Name, ref.ModTime, file)
	})
	mux.HandleFunc("POST /api/v1/dst/logs/bundle/export", func(w http.ResponseWriter, r *http.Request) {
		var request logcenter.BundleRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 日志诊断包请求格式无效")
			return
		}
		result, err := s.application.ExportDSTLogBundle(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("GET /api/v1/dst/logs/bundle/download", func(w http.ResponseWriter, r *http.Request) {
		request := logcenter.BundleRequest{
			ClusterPath: strings.TrimSpace(r.URL.Query().Get("clusterPath")),
			ShardNames:  r.URL.Query()["shard"],
		}
		name, err := s.application.DSTLogBundleName(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		setAttachmentHeader(w, name, "AI Game Manager Panel-DST-Logs.zip")
		w.Header().Set("Content-Type", "application/zip")
		if _, err := s.application.StreamDSTLogBundle(request, w); err != nil {
			// Headers may already be committed. The browser will report a truncated
			// download instead of buffering a second error document in memory.
			return
		}
	})

	mux.HandleFunc("POST /api/v1/dst/logs/bundle/download", func(w http.ResponseWriter, r *http.Request) {
		var request logcenter.BundleRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "DST 日志诊断包请求格式无效")
			return
		}
		result, err := s.application.ExportDSTLogBundle(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		file, err := os.Open(result.Path)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		setAttachmentHeader(w, result.Name, "AI Game Manager Panel-DST-Logs.zip")
		w.Header().Set("Content-Type", "application/zip")
		http.ServeContent(w, r, result.Name, stat.ModTime(), file)
	})

	if s.assets != nil {
		mux.Handle("/", spaHandler(s.assets))
	}
	return withSecurityHeaders(withNoCacheAPI(withOriginProtection(s.requireSession(mux))))
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	public := map[string]struct{}{
		"GET /api/v1/health":                       {},
		"GET /api/v1/auth/bootstrap":               {},
		"POST /api/v1/auth/bootstrap/owner":        {},
		"POST /api/v1/auth/security-key/generate":  {},
		"POST /api/v1/auth/invitation/inspect":     {},
		"POST /api/v1/auth/invitation/register":    {},
		"POST /api/v1/auth/login":                  {},
		"POST /api/v1/auth/password-reset/request": {},
		"POST /api/v1/auth/password-reset/confirm": {},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Method + " " + r.URL.Path
		if _, ok := public[key]; ok {
			next.ServeHTTP(w, r)
			return
		}
		token, viaCookie := sessionTokenFromRequest(r)
		if viaCookie && methodRequiresCSRF(r.Method) && !validCSRFFromRequest(r) {
			writeError(w, http.StatusForbidden, "CSRF 安全校验失败，请刷新页面后重试")
			return
		}
		if _, err := s.application.ValidateSession(token); err != nil {
			if errors.Is(err, authservice.ErrOrganizationRequired) {
				writeError(w, http.StatusForbidden, authservice.ErrOrganizationRequired.Error())
			} else {
				writeError(w, http.StatusUnauthorized, "需要有效的组织成员登录会话")
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	token, _ := sessionTokenFromRequest(r)
	return token
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, authservice.ErrUnauthorized), errors.Is(err, authservice.ErrInvalidLogin), errors.Is(err, authservice.ErrSecurityKeyRequired):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, authservice.ErrBootstrapClosed):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, authservice.ErrForbidden), errors.Is(err, authservice.ErrOrganizationRequired), errors.Is(err, authservice.ErrInvitationRequired), errors.Is(err, authservice.ErrCoreAuthorizationRequired), errors.Is(err, authservice.ErrCoreAccessSuspended), errors.Is(err, authservice.ErrRiskLocked):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, authservice.ErrEmailStepUpRequired), errors.Is(err, authservice.ErrCredentialStepUpRequired):
		writeError(w, http.StatusPreconditionRequired, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func spaHandler(assets fs.FS) http.Handler {
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if info, err := fs.Stat(assets, path); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		index, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}

func setAttachmentHeader(w http.ResponseWriter, name, fallback string) {
	name = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(name, "\r", ""), "\n", ""), "\"", "")
	fallback = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fallback, "\r", ""), "\n", ""), "\"", "")
	if fallback == "" {
		fallback = "download.bin"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fallback, url.PathEscape(name)))
}

func requireLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("无效的 Web 监听地址 %q: %w", addr, err)
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("AI Game Manager Panel 0.1.45 仍只允许本机 Web 访问（127.0.0.1/localhost）；远程监听需后续显式启用传输安全与远程访问策略")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func withNoCacheAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

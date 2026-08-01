package workbench

import (
	"net/http"

	"slim-agent/internal/harness"
	"slim-agent/internal/skills"
)

// Server 装配 HTTP 服务：路由注册、鉴权中间件、CORS。
// 领域 handler 分文件注册；UsageHandler（task4）在此挂载。
type Server struct {
	store    *WorkbenchStore
	harness  *harness.HarnessStore
	auth     *AuthMiddleware
	hub      *EventHub
	drafts   *DraftService
	skills   *skills.Registry
	corsOrigins []string
}

// ServerOptions 可选项。
type ServerOptions struct {
	// CORSOrigins 浏览器直连时允许的源白名单；为空则不启用 CORS。
	CORSOrigins []string
	// AuthBypass 开发模式跳过鉴权。
	AuthBypass bool
	// Skills 已注册 Skill 注册表；nil 时 /api/skills 返回空列表。
	Skills *skills.Registry
	// Runtime 运行时执行引擎；nil 时批准草案仅创建 RunState 不启动执行。
	Runtime *harness.HarnessRuntime
	// EventHub 外部事件广播中心；非 nil 时使用外部实例而非内部新建。
	EventHub *EventHub
}

// NewServer 构造 Server 并装配全部路由。
func NewServer(store *WorkbenchStore, h *harness.HarnessStore, opts ServerOptions) *Server {
	hub := opts.EventHub
	if hub == nil {
		hub = NewEventHub()
	}
	s := &Server{
		store:       store,
		harness:     h,
		auth:        NewAuthMiddleware(store).WithAuthBypass(opts.AuthBypass),
		hub:         hub,
		drafts:      NewDraftService(store, h, opts.Runtime),
		skills:      opts.Skills,
		corsOrigins: opts.CORSOrigins,
	}
	return s
}

// Hub 暴露 EventHub 供运行时广播事件。
func (s *Server) Hub() *EventHub { return s.hub }

// Handler 返回完整路由树。
// 公开路由（health/login）免鉴权；其余路由受 AuthMiddleware 保护 + CORS。
func (s *Server) Handler() http.Handler {
	public := http.NewServeMux()
	public.HandleFunc("GET /health", s.handleHealth)
	public.HandleFunc("POST /api/auth/login", s.handleLogin)
	public.HandleFunc("POST /api/auth/register", s.handleRegister)

	protected := http.NewServeMux()
	// 认证
	protected.HandleFunc("POST /api/auth/logout", s.handleLogout)
	protected.HandleFunc("GET /api/auth/me", s.handleMe)
	protected.HandleFunc("PATCH /api/auth/me", s.handleUpdateMe)
	// 账户列表仅登录用户可见（防公开枚举）
	protected.HandleFunc("GET /api/accounts", s.handleListAccounts)

	// 项目
	protected.HandleFunc("GET /api/projects", s.handleListProjects)
	protected.HandleFunc("POST /api/projects", s.handleCreateProject)
	protected.HandleFunc("GET /api/projects/{id}", s.handleGetProject)
	protected.HandleFunc("PATCH /api/projects/{id}", s.handleUpdateProject)
	protected.HandleFunc("DELETE /api/projects/{id}", s.handleDeleteProject)

	// 会话与消息
	protected.HandleFunc("GET /api/conversations", s.handleListConversations)
	protected.HandleFunc("POST /api/conversations", s.handleCreateConversation)
	protected.HandleFunc("GET /api/conversations/{id}", s.handleGetConversation)
	protected.HandleFunc("PATCH /api/conversations/{id}", s.handleUpdateConversation)
	protected.HandleFunc("DELETE /api/conversations/{id}", s.handleDeleteConversation)
	protected.HandleFunc("GET /api/conversations/{id}/messages", s.handleListMessages)
	protected.HandleFunc("POST /api/conversations/{id}/messages", s.handleAppendMessage)

	// 任务草案与审批
	protected.HandleFunc("GET /api/task-drafts", s.handleListDrafts)
	protected.HandleFunc("POST /api/task-drafts", s.handleCreateDraft)
	protected.HandleFunc("GET /api/task-drafts/{id}", s.handleGetDraft)
	protected.HandleFunc("PATCH /api/task-drafts/{id}", s.handleUpdateDraft)
	protected.HandleFunc("DELETE /api/task-drafts/{id}", s.handleDeleteDraft)
	protected.HandleFunc("POST /api/task-drafts/{id}/approve", s.handleApproveDraft)
	protected.HandleFunc("POST /api/task-drafts/{id}/reject", s.handleRejectDraft)

	// 供应商档案
	protected.HandleFunc("GET /api/providers/presets", s.handleListPresets)
	protected.HandleFunc("GET /api/providers", s.handleListProfiles)
	protected.HandleFunc("POST /api/providers", s.handleCreateProfile)
	protected.HandleFunc("DELETE /api/providers/{id}", s.handleDeleteProfile)
	protected.HandleFunc("POST /api/providers/{id}/activate", s.handleActivateProfile)
	protected.HandleFunc("GET /api/providers/{id}/health", s.handleProfileHealth)

	// 用量（task4）
	NewUsageHandler(s.store).Routes(protected)

	// SSE
	protected.HandleFunc("GET /api/events", s.handleEvents)

	// Skill 占位（task7 挂接真实列表）
	protected.HandleFunc("GET /api/skills", s.handleListSkills)

	authProtected := s.auth.Wrap(protected)
	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register" {
			public.ServeHTTP(w, r)
			return
		}
		authProtected.ServeHTTP(w, r)
	})
	return s.corsMiddleware(root)
}

// corsMiddleware 按白名单严格匹配 Origin；不在白名单返回 403。
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	if len(s.corsOrigins) == 0 {
		return next
	}
	allowed := make(map[string]bool, len(s.corsOrigins))
	for _, o := range s.corsOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && !allowed[origin] {
			writeError(w, harness.ErrPermissionDenied("origin not allowed"))
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleHealth 健康检查。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleListSkills 返回已注册 Skill 列表（真实 Registry 挂接；nil 时返回空列表）。
func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	var manifests []*harness.SkillManifest
	if s.skills != nil {
		manifests = s.skills.List()
	}
	skillsOut := make([]map[string]any, 0, len(manifests))
	for _, m := range manifests {
		skillsOut = append(skillsOut, map[string]any{
			"skill_id":              m.SkillID,
			"version":               m.Version,
			"title":                 m.Title,
			"description":           m.Description,
			"output_artifact_kinds": m.OutputArtifactKinds,
			"required_validators":   m.RequiredValidators,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"skills": skillsOut})
}

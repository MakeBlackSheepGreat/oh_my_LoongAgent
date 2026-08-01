package workbench

import (
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"slim-agent/internal/harness"
)

// loginRequest 登录请求体：用户名 + 密码。
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleLogin 用户名密码登录：校验凭据 → CreateSession + Set-Cookie HttpOnly SameSite=Strict Path=/。
// POST /api/auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, harness.ErrValidation("username and password are required", nil))
		return
	}
	// 密码长度上限：防止超长输入在 PBKDF2 首轮造成过量哈希计算（未认证端点 CPU DoS 面）。
	// 按字符数（rune）计，与前端 maxlength=128 语义一致（UTF-16 字符 ≈ rune）。
	if utf8.RuneCountInString(req.Password) > 128 {
		writeError(w, harness.ErrValidation("password must be at most 128 characters", nil))
		return
	}
	acc, err := s.store.GetAccountByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	// 统一返回 UNAUTHORIZED，不区分"账户不存在/密码错误/未设密码"，避免枚举账户。
	// 账户不存在或已禁用时仍执行一次 PBKDF2（HashPassword），抹平与真实校验之间的时序差。
	if acc == nil || acc.Status != AccountActive {
		_, _ = HashPassword(req.Password)
		writeUnauthorized(w, "invalid username or password")
		return
	}
	if !verifyPassword(acc.PasswordHash, req.Password) {
		// 未设密码（旧库迁移账户）时 verifyPassword 立即返回，比真实校验快——补一次 dummy 抹平时序。
		if acc.PasswordHash == "" {
			_, _ = HashPassword(req.Password)
		}
		writeUnauthorized(w, "invalid username or password")
		return
	}
	sess, err := s.store.CreateSession(r.Context(), acc.AccountID)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // 本地服务默认 http
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, acc)
}

// handleLogout 注销：DeleteSession + 清除 cookie。
// POST /api/auth/logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = s.store.DeleteSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// handleMe 从 context 返回当前账户。
// GET /api/auth/me
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	acc, err := s.store.GetAccount(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	if acc == nil {
		writeUnauthorized(w, "account no longer exists")
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

// handleListAccounts 公开账户列表（兼容保留；登录页已改为用户名密码表单，不再依赖）。
// 仅返回 active 账户的公开字段（不含密码哈希）。
// GET /api/accounts
func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(accounts))
	for _, a := range accounts {
		if a.Status != AccountActive {
			continue
		}
		out = append(out, map[string]any{
			"account_id":   a.AccountID,
			"username":     a.Username,
			"display_name": a.DisplayName,
			"locale":       a.Locale,
			"status":       a.Status,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

// registerRequest 注册账户请求体。
type registerRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Locale      string `json:"locale"`
}

// handleRegister 注册新账户并自动登录：校验 → 哈希密码 → 创建账户 → 创建会话 → Set-Cookie。
// POST /api/auth/register
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	if req.DisplayName == "" {
		writeError(w, harness.ErrValidation("display_name is required", nil))
		return
	}
	if len(req.DisplayName) > 64 {
		writeError(w, harness.ErrValidation("display_name must be <= 64 characters", nil))
		return
	}
	if !reUsername.MatchString(req.Username) {
		writeError(w, harness.ErrValidation("username must be 3-32 characters of [A-Za-z0-9_-]", nil))
		return
	}
	if utf8.RuneCountInString(req.Password) < 6 {
		writeError(w, harness.ErrValidation("password must be at least 6 characters", nil))
		return
	}
	if utf8.RuneCountInString(req.Password) > 128 {
		writeError(w, harness.ErrValidation("password must be at most 128 characters", nil))
		return
	}
	if req.Locale == "" {
		req.Locale = defaultLocale
	}
	if req.Locale != "zh-CN" && req.Locale != "en" {
		writeError(w, harness.ErrValidation("locale must be zh-CN or en", nil))
		return
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		writeError(w, harness.NewHarnessError(harness.ErrCodeInternal, "failed to hash password", err))
		return
	}
	acc, err := s.store.CreateAccount(r.Context(), req.Username, req.DisplayName, req.Locale, hash)
	if err != nil {
		writeError(w, err)
		return
	}
	sess, err := s.store.CreateSession(r.Context(), acc.AccountID)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusCreated, acc)
}

// updateMeRequest 更新当前账户偏好请求体。
type updateMeRequest struct {
	Locale string `json:"locale"`
}

// handleUpdateMe 更新当前账户 locale 偏好并返回更新后的账户。
// PATCH /api/auth/me
func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req updateMeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	if req.Locale != "zh-CN" && req.Locale != "en" {
		writeError(w, harness.ErrValidation("locale must be zh-CN or en", nil))
		return
	}
	acc, err := s.store.UpdateAccountLocale(r.Context(), accountID, req.Locale)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

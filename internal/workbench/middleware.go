package workbench

import (
	"context"
	"encoding/json"
	"net/http"

	"slim-agent/internal/harness"
)

// contextKey 用于避免 context 键冲突。
type contextKey string

const accountContextKey contextKey = "workbench_account_id"

// sessionCookieName 会话 cookie 名称。
const sessionCookieName = "harness_session"

// AccountFromContext 从请求 context 读取当前 account_id；不存在返回错误。
func AccountFromContext(ctx context.Context) (string, error) {
	v, ok := ctx.Value(accountContextKey).(string)
	if !ok || v == "" {
		return "", harness.ErrPermissionDenied("no account_id in context; authentication required")
	}
	return v, nil
}

// AuthMiddleware 解析会话 cookie 并注入 account_id 到请求 context。
// 未登录访问返回 401 JSON；WithAuthBypass 用于开发模式跳过鉴权。
type AuthMiddleware struct {
	store    *WorkbenchStore
	bypass   bool
}

// NewAuthMiddleware 构造 AuthMiddleware；默认要求鉴权。
func NewAuthMiddleware(store *WorkbenchStore) *AuthMiddleware {
	return &AuthMiddleware{store: store}
}

// WithAuthBypass 设置开发模式跳过鉴权；返回 middleware 自身供链式调用。
// 开发模式下未登录请求注入第一个账户的 account_id。
func (m *AuthMiddleware) WithAuthBypass(bypass bool) *AuthMiddleware {
	m.bypass = bypass
	return m
}

// Wrap 包装 http.Handler，注入 account_id 到 context。
func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.bypass {
			// 开发模式：取第一个账户注入 context
			accounts, err := m.store.ListAccounts(r.Context())
			if err != nil || len(accounts) == 0 {
				writeUnauthorized(w, "no account available in bypass mode")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), accountContextKey, accounts[0].AccountID))
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeUnauthorized(w, "missing session cookie")
			return
		}
		sess, err := m.store.GetSession(r.Context(), cookie.Value)
		if err != nil {
			writeUnauthorized(w, "session lookup failed")
			return
		}
		if sess == nil {
			writeUnauthorized(w, "invalid or expired session")
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), accountContextKey, sess.AccountID))
		next.ServeHTTP(w, r)
	})
}

// writeUnauthorized 写入 401 JSON 错误响应。
func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":  "UNAUTHORIZED",
		"detail": message,
	})
}

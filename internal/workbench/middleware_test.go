package workbench

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAuthMiddleware_Unauthorized_NoCookie 验证无 cookie 返回 401。
func TestAuthMiddleware_Unauthorized_NoCookie(t *testing.T) {
	store := newTestStore(t)
	mw := NewAuthMiddleware(store)
	called := false
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if called {
		t.Fatal("handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestAuthMiddleware_Unauthorized_InvalidSession 验证无效 session 返回 401。
func TestAuthMiddleware_Unauthorized_InvalidSession(t *testing.T) {
	store := newTestStore(t)
	mw := NewAuthMiddleware(store)
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "invalid_session"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestAuthMiddleware_Authorized 验证有效 session 注入 account_id。
func TestAuthMiddleware_Authorized(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	sess, _ := store.CreateSession(context.Background(), acc.AccountID)
	mw := NewAuthMiddleware(store)

	var gotAccountID string
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccountID, _ = AccountFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.SessionID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if gotAccountID != acc.AccountID {
		t.Fatalf("account_id: got %q, want %q", gotAccountID, acc.AccountID)
	}
}

// TestAuthMiddleware_BypassMode 验证开发模式跳过鉴权。
func TestAuthMiddleware_BypassMode(t *testing.T) {
	store := newTestStore(t)
	// bypass 模式需要至少一个账户
	store.CreateAccount(context.Background(), "default", "default", "zh-CN", "test-hash")
	mw := NewAuthMiddleware(store).WithAuthBypass(true)

	var gotAccountID string
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccountID, _ = AccountFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if gotAccountID == "" {
		t.Fatal("account_id should be injected in bypass mode")
	}
}

// TestAccountFromContext_NoAccount 验证无 account_id 时返回错误。
func TestAccountFromContext_NoAccount(t *testing.T) {
	_, err := AccountFromContext(context.Background())
	if err == nil {
		t.Fatal("expected error for missing account_id")
	}
}

package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"slim-agent/internal/harness"
)

func TestListAccounts_ProtectedFields(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	if _, err := srv.store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash"); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out struct {
		Accounts []map[string]any `json:"accounts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Accounts) < 1 {
		t.Fatalf("expected at least 1 account, got %d", len(out.Accounts))
	}
	// 公开字段齐全，且不含敏感字段
	for _, a := range out.Accounts {
		if a["account_id"] == "" || a["display_name"] == "" || a["locale"] == "" {
			t.Fatalf("incomplete public account fields: %v", a)
		}
		if _, has := a["session_id"]; has {
			t.Fatalf("session_id must not be exposed: %v", a)
		}
		if _, has := a["password_hash"]; has {
			t.Fatalf("password_hash must not be exposed: %v", a)
		}
	}
}

func TestListAccounts_RequiresAuth(t *testing.T) {
	// 非 bypass 服务器：未登录访问 /api/accounts 必须 401（防公开枚举）。
	store := newTestStore(t)
	hstore := harness.NewHarnessStore(t.TempDir())
	if err := hstore.Initialize(); err != nil {
		t.Fatalf("harness Initialize: %v", err)
	}
	t.Cleanup(func() { _ = hstore.Close() })
	srv := NewServer(store, hstore, ServerOptions{})

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", rr.Code)
	}
}

func TestUpdateAccountLocale(t *testing.T) {
	srv := newTestHTTPServer(t, nil)

	// 合法 locale
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{"locale":"en"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var acc Account
	if err := json.Unmarshal(rr.Body.Bytes(), &acc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if acc.Locale != "en" {
		t.Fatalf("expected locale en, got %s", acc.Locale)
	}

	// 非法 locale → 400
	req = httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{"locale":"fr"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRegister_Success(t *testing.T) {
	srv := newTestHTTPServer(t, nil)

	// 注册新账户（用户名 + 密码）
	body := `{"display_name":"alice","username":"alice","password":"secret123","locale":"en"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var acc Account
	if err := json.Unmarshal(rr.Body.Bytes(), &acc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if acc.DisplayName != "alice" {
		t.Fatalf("display_name: got %q", acc.DisplayName)
	}
	if acc.Locale != "en" {
		t.Fatalf("locale: got %q", acc.Locale)
	}
	if acc.AccountID == "" {
		t.Fatal("account_id must not be empty")
	}
	// 验证 set-cookie
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "harness_session" && c.Value != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected session cookie to be set")
	}
}

func TestRegister_EmptyDisplayName(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	body := `{"display_name":"","username":"alice","password":"secret123","locale":"en"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegister_InvalidLocale(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	body := `{"display_name":"bob","username":"bob","password":"secret123","locale":"fr"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegister_DefaultLocale(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	body := `{"display_name":"charlie","username":"charlie","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var acc Account
	if err := json.Unmarshal(rr.Body.Bytes(), &acc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if acc.Locale != "zh-CN" {
		t.Fatalf("expected default locale zh-CN, got %s", acc.Locale)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	body := `{"display_name":"dave","username":"dave","password":"123","locale":"zh-CN"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRegister_OverlongPassword(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	// 129 个 ASCII 字符 = 129 字节 = 129 rune > 128 上限
	long := strings.Repeat("x", 129)
	body := fmt.Sprintf(`{"display_name":"eve","username":"eve","password":%q,"locale":"zh-CN"}`, long)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overlong password, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLogin_OverlongPassword(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if _, err := srv.store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", hash); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	// 超长密码在 PBKDF2 前被拒绝（400），不进入哈希计算
	long := strings.Repeat("y", 129)
	body := fmt.Sprintf(`{"username":"alice","password":%q}`, long)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for overlong login password, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLogin_EmptyHashAccount(t *testing.T) {
	// 旧库迁移账户（password_hash 为空）登录必须 401（统一错误消息），且不 panic。
	srv := newTestHTTPServer(t, nil)
	// 迁移回填后的旧账户 password_hash 为空——直接 SQL 插入模拟（CreateAccount 拒绝空 hash 是设计行为）。
	if _, err := srv.store.db.ExecContext(context.Background(),
		`INSERT INTO accounts(account_id, username, display_name, password_hash, status, locale, created_at)
		 VALUES ('06FVSA0R5PERW1PS21WDY13BVM', 'legacy', 'legacy', '', 'active', 'zh-CN', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert legacy account: %v", err)
	}
	body := `{"username":"legacy","password":"whatever123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty-hash account, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLogin_SuccessAndWrongPassword(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if _, err := srv.store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", hash); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	// 正确密码 → 200 + cookie
	body := `{"username":"alice","password":"correct-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var acc Account
	if err := json.Unmarshal(rr.Body.Bytes(), &acc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if acc.Username != "alice" {
		t.Fatalf("username: got %q", acc.Username)
	}
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session cookie")
	}

	// 错误密码 → 401（不区分账户是否存在）
	body = `{"username":"alice","password":"wrong-password"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rr.Code)
	}

	// 不存在的用户 → 401（同一条错误消息，防枚举）
	body = `{"username":"nobody","password":"whatever"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown user, got %d", rr.Code)
	}
}

func TestVerifyPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !verifyPassword(hash, "s3cret-pass") {
		t.Fatal("verifyPassword must accept correct password")
	}
	if verifyPassword(hash, "wrong") {
		t.Fatal("verifyPassword must reject wrong password")
	}
	if verifyPassword("", "s3cret-pass") {
		t.Fatal("empty hash must always fail")
	}
	if verifyPassword("not-a-valid-format", "s3cret-pass") {
		t.Fatal("malformed hash must fail")
	}
}

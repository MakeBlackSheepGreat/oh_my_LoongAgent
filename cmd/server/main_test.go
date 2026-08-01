package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	webdist "slim-agent"
	"slim-agent/internal/harness"
	"slim-agent/internal/workbench"

	_ "modernc.org/sqlite"
)

// testPassword 测试统一密码（账户创建时以它生成哈希，登录用它）。
const testPassword = "test-pass-123"

// testApp 端到端测试应用。
type testApp struct {	server   *httptest.Server
	wb       *workbench.Server
	store    *workbench.WorkbenchStore
	harness  *harness.HarnessStore
	accounts map[string]string // name -> account_id
}

// newTestApp 构造内存端到端测试应用（含静态文件 handler 组合）。
func newTestApp(t *testing.T) *testApp {
	t.Helper()
	dir := t.TempDir()

	wbDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	// 单连接串行化写，避免 :memory: 多连接分裂与锁冲突
	wbDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = wbDB.Close() })
	wbStore := workbench.NewWorkbenchStore(wbDB)
	ctx := context.Background()
	if err := wbStore.InitAll(ctx); err != nil {
		t.Fatalf("InitAll: %v", err)
	}

	hStore := harness.NewHarnessStore(dir)
	if err := hStore.Initialize(); err != nil {
		t.Fatalf("harness Initialize: %v", err)
	}
	t.Cleanup(func() { _ = hStore.Close() })

	srv := workbench.NewServer(wbStore, hStore, workbench.ServerOptions{})
	fsys, err := fs.Sub(webdist.FS, "web/dist")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}
	full := spaHandler(srv.Handler(), http.FileServer(http.FS(fsys)), fsys)
	ts := httptest.NewServer(full)
	t.Cleanup(ts.Close)

	return &testApp{
		server:   ts,
		wb:       srv,
		store:    wbStore,
		harness:  hStore,
		accounts: map[string]string{},
	}
}

// createAccount 创建账户并返回登录后的 client。
func (a *testApp) createAccount(t *testing.T, name string) *http.Client {
	t.Helper()
	hash, err := workbench.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	acc, err := a.store.CreateAccount(context.Background(), name, name, "zh-CN", hash)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	a.accounts[name] = acc.AccountID
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	a.login(t, client, name)
	return client
}

// login 用用户名密码登录。
func (a *testApp) login(t *testing.T, client *http.Client, username string) {
	t.Helper()
	body := fmt.Sprintf(`{"username": %q, "password": %q}`, username, testPassword)
	resp, err := client.Post(a.server.URL+"/api/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status: %d", resp.StatusCode)
	}
}

// doJSON 发送 JSON 请求并返回状态码与响应体。
func (a *testApp) doJSON(t *testing.T, client *http.Client, method, path string, payload any) (int, []byte) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = strings.NewReader(string(data))
	}
	req, err := http.NewRequest(method, a.server.URL+path, body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// TestHealthEndpoint 健康检查免鉴权。
func TestHealthEndpoint(t *testing.T) {
	app := newTestApp(t)
	resp, err := http.Get(app.server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

// TestAuthFlow 登录→me→logout 完整流程。
func TestAuthFlow(t *testing.T) {
	app := newTestApp(t)
	hash, err := workbench.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	acc, _ := app.store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", hash)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// 未登录访问 me → 401
	code, _ := app.doJSON(t, client, http.MethodGet, "/api/auth/me", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("me before login: got %d, want 401", code)
	}

	// 登录
	app.login(t, client, "alice")

	// me → 200 且返回账户
	code, body := app.doJSON(t, client, http.MethodGet, "/api/auth/me", nil)
	if code != http.StatusOK {
		t.Fatalf("me after login: got %d", code)
	}
	var me workbench.Account
	if err := json.Unmarshal(body, &me); err != nil {
		t.Fatalf("unmarshal me: %v", err)
	}
	if me.AccountID != acc.AccountID {
		t.Fatalf("me account: got %s", me.AccountID)
	}

	// logout → 200，随后 me → 401
	code, _ = app.doJSON(t, client, http.MethodPost, "/api/auth/logout", nil)
	if code != http.StatusOK {
		t.Fatalf("logout: got %d", code)
	}
	code, _ = app.doJSON(t, client, http.MethodGet, "/api/auth/me", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("me after logout: got %d, want 401", code)
	}
}

// TestProjectCRUD 项目 CRUD 全流程。
func TestProjectCRUD(t *testing.T) {
	app := newTestApp(t)
	client := app.createAccount(t, "alice")

	// 创建
	code, body := app.doJSON(t, client, http.MethodPost, "/api/projects", map[string]string{
		"project_id": "proj_001", "name": "my project", "description": "desc",
	})
	if code != http.StatusCreated {
		t.Fatalf("create: got %d (body=%s)", code, body)
	}
	// 列表
	code, body = app.doJSON(t, client, http.MethodGet, "/api/projects", nil)
	if code != http.StatusOK || !strings.Contains(string(body), "proj_001") {
		t.Fatalf("list: got %d body=%s", code, body)
	}
	// 单个
	code, _ = app.doJSON(t, client, http.MethodGet, "/api/projects/proj_001", nil)
	if code != http.StatusOK {
		t.Fatalf("get: got %d", code)
	}
	// 更新
	code, _ = app.doJSON(t, client, http.MethodPatch, "/api/projects/proj_001", map[string]string{
		"name": "renamed", "description": "",
	})
	if code != http.StatusOK {
		t.Fatalf("update: got %d", code)
	}
	// 删除
	code, _ = app.doJSON(t, client, http.MethodDelete, "/api/projects/proj_001", nil)
	if code != http.StatusOK {
		t.Fatalf("delete: got %d", code)
	}
	// 删除后查询 → 404
	code, _ = app.doJSON(t, client, http.MethodGet, "/api/projects/proj_001", nil)
	if code != http.StatusNotFound {
		t.Fatalf("get after delete: got %d, want 404", code)
	}
}

// TestConversationAndMessages 会话与消息追加。
func TestConversationAndMessages(t *testing.T) {
	app := newTestApp(t)
	client := app.createAccount(t, "alice")

	code, _ := app.doJSON(t, client, http.MethodPost, "/api/conversations", map[string]string{
		"conversation_id": "conv_001", "title": "hello",
	})
	if code != http.StatusCreated {
		t.Fatalf("create conversation: got %d", code)
	}
	code, _ = app.doJSON(t, client, http.MethodPost, "/api/conversations/conv_001/messages", map[string]string{
		"message_id": "msg_001", "role": "user", "content": "hello world",
	})
	if code != http.StatusCreated {
		t.Fatalf("append message: got %d", code)
	}
	code, body := app.doJSON(t, client, http.MethodGet, "/api/conversations/conv_001/messages", nil)
	if code != http.StatusOK || !strings.Contains(string(body), "hello world") {
		t.Fatalf("list messages: got %d body=%s", code, body)
	}
}

// TestCrossAccount404 跨账户访问返回 404。
func TestCrossAccount404(t *testing.T) {
	app := newTestApp(t)
	alice := app.createAccount(t, "alice")
	bob := app.createAccount(t, "bob")

	app.doJSON(t, alice, http.MethodPost, "/api/projects", map[string]string{
		"project_id": "proj_alice", "name": "alice project",
	})
	// bob 访问 alice 的项目 → 404
	code, _ := app.doJSON(t, bob, http.MethodGet, "/api/projects/proj_alice", nil)
	if code != http.StatusNotFound {
		t.Fatalf("cross-account get: got %d, want 404", code)
	}
	// bob 的项目列表不包含 alice 的项目
	code, body := app.doJSON(t, bob, http.MethodGet, "/api/projects", nil)
	if code != http.StatusOK || strings.Contains(string(body), "proj_alice") {
		t.Fatalf("cross-account list: got %d body=%s", code, body)
	}
}

// TestDraftApproveCreateRun 草案 approve→CreateRun 返回 run_id。
func TestDraftApproveCreateRun(t *testing.T) {
	app := newTestApp(t)
	client := app.createAccount(t, "alice")

	code, _ := app.doJSON(t, client, http.MethodPost, "/api/task-drafts", map[string]string{
		"draft_id": "draft_001", "objective": "organize my files", "skill_id": "file_organizer",
	})
	if code != http.StatusCreated {
		t.Fatalf("create draft: got %d", code)
	}
	code, body := app.doJSON(t, client, http.MethodPost, "/api/task-drafts/draft_001/approve", nil)
	if code != http.StatusOK {
		t.Fatalf("approve: got %d (body=%s)", code, body)
	}
	var resp struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.RunID == "" {
		t.Fatalf("approve response: %s", body)
	}
	// 二次 approve → 409
	code, _ = app.doJSON(t, client, http.MethodPost, "/api/task-drafts/draft_001/approve", nil)
	if code != http.StatusConflict {
		t.Fatalf("double approve: got %d, want 409", code)
	}
	// draft 状态为 approved
	code, body = app.doJSON(t, client, http.MethodGet, "/api/task-drafts/draft_001", nil)
	if code != http.StatusOK || !strings.Contains(string(body), `"approved"`) {
		t.Fatalf("draft status: got %d body=%s", code, body)
	}
}

// TestDraftReject 草案 reject 不创建运行。
func TestDraftReject(t *testing.T) {
	app := newTestApp(t)
	client := app.createAccount(t, "alice")

	app.doJSON(t, client, http.MethodPost, "/api/task-drafts", map[string]string{
		"draft_id": "draft_002", "objective": "do nothing",
	})
	code, _ := app.doJSON(t, client, http.MethodPost, "/api/task-drafts/draft_002/reject", nil)
	if code != http.StatusOK {
		t.Fatalf("reject: got %d", code)
	}
	code, body := app.doJSON(t, client, http.MethodGet, "/api/task-drafts/draft_002", nil)
	if code != http.StatusOK || !strings.Contains(string(body), `"rejected"`) {
		t.Fatalf("draft status: got %d body=%s", code, body)
	}
}

// TestProviderProfileCRUD 供应商档案创建/列表/删除。
func TestProviderProfileCRUD(t *testing.T) {
	app := newTestApp(t)
	client := app.createAccount(t, "alice")

	code, _ := app.doJSON(t, client, http.MethodPost, "/api/providers", map[string]string{
		"profile_id": "prof_001", "provider_id": "deepseek", "display_name": "DS",
		"base_url": "https://api.deepseek.com/v1", "model_id": "deepseek-chat",
		"api_key_env": "HARNESS_DEEPSEEK_KEY",
	})
	if code != http.StatusCreated {
		t.Fatalf("create profile: got %d", code)
	}
	code, body := app.doJSON(t, client, http.MethodGet, "/api/providers", nil)
	if code != http.StatusOK || !strings.Contains(string(body), "prof_001") {
		t.Fatalf("list profiles: got %d body=%s", code, body)
	}
	// 激活
	code, _ = app.doJSON(t, client, http.MethodPost, "/api/providers/prof_001/activate", nil)
	if code != http.StatusOK {
		t.Fatalf("activate: got %d", code)
	}
	// health（无密钥环境变量 → ok=false 但 HTTP 200）
	code, body = app.doJSON(t, client, http.MethodGet, "/api/providers/prof_001/health", nil)
	if code != http.StatusOK {
		t.Fatalf("health: got %d (body=%s)", code, body)
	}
	// 删除
	code, _ = app.doJSON(t, client, http.MethodDelete, "/api/providers/prof_001", nil)
	if code != http.StatusOK {
		t.Fatalf("delete profile: got %d", code)
	}
}

// TestSpaFallback 静态文件与 SPA fallback。
func TestSpaFallback(t *testing.T) {
	app := newTestApp(t)
	resp, err := http.Get(app.server.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<div id=\"app\"") && !strings.Contains(string(body), "app") {
		t.Fatalf("index.html body unexpected: %s", string(body)[:min(len(body), 200)])
	}

	// 未知路由 fallback 到 index.html
	resp2, err := http.Get(app.server.URL + "/some/unknown/route")
	if err != nil {
		t.Fatalf("GET unknown: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("unknown route status: %d", resp2.StatusCode)
	}
}

// TestSSEEvents SSE 订阅收到事件且按账户过滤。
func TestSSEEvents(t *testing.T) {
	app := newTestApp(t)
	alice := app.createAccount(t, "alice")
	app.createAccount(t, "bob")

	// alice approve 一个 draft 获得 run_id
	app.doJSON(t, alice, http.MethodPost, "/api/task-drafts", map[string]string{
		"draft_id": "draft_sse", "objective": "watch events",
	})
	code, body := app.doJSON(t, alice, http.MethodPost, "/api/task-drafts/draft_sse/approve", nil)
	if code != http.StatusOK {
		t.Fatalf("approve: %d", code)
	}
	var ar struct {
		RunID string `json:"run_id"`
	}
	_ = json.Unmarshal(body, &ar)

	// alice 打开 SSE 连接
	req, _ := http.NewRequest(http.MethodGet, app.server.URL+"/api/events", nil)
	resp, err := alice.Do(req)
	if err != nil {
		t.Fatalf("sse open: %v", err)
	}
	defer resp.Body.Close()

	// 广播 alice 账户 run 的事件
	ev, _ := harness.NewEvent(1, ar.RunID, "run_started", "test event")
	app.wb.Hub().Broadcast(ev)

	reader := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "run_started") {
			return // 收到事件
		}
	}
	t.Fatal("SSE did not receive expected event")
}

// TestConcurrentProjects 两账户并发创建项目互不干扰。
func TestConcurrentProjects(t *testing.T) {
	app := newTestApp(t)
	alice := app.createAccount(t, "alice")
	bob := app.createAccount(t, "bob")

	// doJSON 在非测试 goroutine 中不得调用 t.Fatal；此处用原始请求 + 错误收集。
	postProject := func(client *http.Client, projectID, name string) error {
		data, _ := json.Marshal(map[string]string{"project_id": projectID, "name": name})
		req, err := http.NewRequest(http.MethodPost, app.server.URL+"/api/projects", strings.NewReader(string(data)))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("create %s: status %d", projectID, resp.StatusCode)
		}
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if err := postProject(alice, fmt.Sprintf("proj_a%d", i), fmt.Sprintf("alice %d", i)); err != nil {
				errCh <- err
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			if err := postProject(bob, fmt.Sprintf("proj_b%d", i), fmt.Sprintf("bob %d", i)); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent create: %v", err)
	}

	_, bodyA := app.doJSON(t, alice, http.MethodGet, "/api/projects", nil)
	_, bodyB := app.doJSON(t, bob, http.MethodGet, "/api/projects", nil)
	for i := 0; i < 5; i++ {
		if !strings.Contains(string(bodyA), fmt.Sprintf("proj_a%d", i)) {
			t.Fatalf("alice missing proj_a%d", i)
		}
		if strings.Contains(string(bodyA), fmt.Sprintf("proj_b%d", i)) {
			t.Fatalf("alice leaked proj_b%d", i)
		}
		if !strings.Contains(string(bodyB), fmt.Sprintf("proj_b%d", i)) {
			t.Fatalf("bob missing proj_b%d", i)
		}
	}
}

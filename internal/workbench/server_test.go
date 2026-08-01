package workbench

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"slim-agent/internal/harness"
	"slim-agent/internal/skills"
	"slim-agent/internal/skills/literature_search"
)

func newTestHTTPServer(t *testing.T, reg *skills.Registry) *Server {
	t.Helper()
	store := newTestStore(t)
	// 创建测试账户（auth bypass 需要至少一个账户注入 account_id）
	if _, err := store.CreateAccount(context.Background(), "test", "test", "zh-CN", "test-hash"); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	hstore := harness.NewHarnessStore(t.TempDir())
	if err := hstore.Initialize(); err != nil {
		t.Fatalf("harness Initialize: %v", err)
	}
	t.Cleanup(func() { _ = hstore.Close() })
	return NewServer(store, hstore, ServerOptions{AuthBypass: true, Skills: reg})
}

func TestListSkills_RealRegistry(t *testing.T) {
	reg := skills.NewRegistry()
	if err := reg.Register(literature_search.NewSkill()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := newTestHTTPServer(t, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(out.Skills))
	}
	if out.Skills[0]["skill_id"] != "literature_search" {
		t.Fatalf("unexpected skill payload: %v", out.Skills[0])
	}
	if out.Skills[0]["title"] == "" || out.Skills[0]["description"] == "" {
		t.Fatalf("skill payload missing fields: %v", out.Skills[0])
	}
}

func TestListSkills_EmptyWithoutRegistry(t *testing.T) {
	srv := newTestHTTPServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Skills) != 0 {
		t.Fatalf("expected empty list, got %d", len(out.Skills))
	}
}

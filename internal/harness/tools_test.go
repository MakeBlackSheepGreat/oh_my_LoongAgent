package harness

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// newTestGovernor 构造允许 simulate.fs 的测试工具执行器。
func newTestGovernor(t *testing.T) *ToolGovernor {
	t.Helper()
	policy := Policy{
		AllowedToolNames:   []string{"simulate.fs"},
		AllowedPermissions: []Permission{PermWriteWorkspace},
		ApprovalRequiredFor: []RiskLevel{RiskHigh, RiskCritical},
		WorkspaceRoot:       t.TempDir(),
		AllowNetworkDomains: []string{"example.com"},
	}
	g := NewToolGovernor(policy)
	if err := g.Register(NewSimulateFSTool()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return g
}

func newTestAction(t *testing.T, toolName string, args map[string]any) *ActionContract {
	t.Helper()
	action, err := NewActionContract("act_1", "run_t1", toolName)
	if err != nil {
		t.Fatalf("NewActionContract: %v", err)
	}
	action.Arguments = args
	return action
}

func TestToolGovernor_RegisterConflict(t *testing.T) {
	g := NewToolGovernor(Policy{})
	if err := g.Register(NewSimulateFSTool()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	err := g.Register(NewSimulateFSTool())
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodeConflict {
		t.Fatalf("expected CONFLICT, got %v", err)
	}
}

func TestToolGovernor_ToolWhitelistDeny(t *testing.T) {
	g := NewToolGovernor(Policy{AllowedPermissions: []Permission{PermWriteWorkspace}})
	g.Register(NewSimulateFSTool())
	action := newTestAction(t, "simulate.fs", map[string]any{"op": "scan"})
	_, err := g.Execute(context.Background(), action, action.Arguments)
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodePermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED, got %v", err)
	}
}

func TestToolGovernor_PermissionDeny(t *testing.T) {
	g := NewToolGovernor(Policy{AllowedToolNames: []string{"simulate.fs"}})
	g.Register(NewSimulateFSTool())
	action := newTestAction(t, "simulate.fs", map[string]any{"op": "scan"})
	_, err := g.Execute(context.Background(), action, action.Arguments)
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodePermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED, got %v", err)
	}
}

func TestToolGovernor_PathEscapeDeny(t *testing.T) {
	g := newTestGovernor(t)
	escaped := filepath.Join(g.policy.WorkspaceRoot, "..", "outside")
	action := newTestAction(t, "simulate.fs", map[string]any{"op": "put", "path": escaped})
	_, err := g.Execute(context.Background(), action, action.Arguments)
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodePermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED, got %v", err)
	}
}

func TestToolGovernor_NetworkDomainDeny(t *testing.T) {
	g := NewToolGovernor(Policy{
		AllowedToolNames:   []string{"net.tool"},
		AllowedPermissions: []Permission{PermNetwork},
		AllowNetworkDomains: []string{"example.com"},
	})
	g.Register(netTool{})
	// 白名单外域名 → 拒绝
	action := newTestAction(t, "net.tool", map[string]any{"url": "http://evil.com/x"})
	_, err := g.Execute(context.Background(), action, action.Arguments)
	he, ok := err.(*HarnessError)
	if !ok || he.Code != ErrCodePermissionDenied {
		t.Fatalf("expected PERMISSION_DENIED for evil.com, got %v", err)
	}
	// 白名单子域 → 放行
	action = newTestAction(t, "net.tool", map[string]any{"url": "http://sub.example.com/x"})
	res, err := g.Execute(context.Background(), action, action.Arguments)
	if err != nil {
		t.Fatalf("expected pass for sub.example.com, got %v", err)
	}
	if !res.OK {
		t.Fatalf("expected ok result, got %s", res.Error)
	}
}

func TestToolGovernor_RequiresApproval(t *testing.T) {
	g := newTestGovernor(t)
	low := newTestAction(t, "simulate.fs", nil)
	if g.RequiresApproval(low) {
		t.Fatal("low risk tool should not require approval")
	}
	// 未知工具保守视为需审批
	unknown := newTestAction(t, "ghost.tool", nil)
	if !g.RequiresApproval(unknown) {
		t.Fatal("unknown tool should require approval conservatively")
	}
}

func TestSimulateFSTool(t *testing.T) {
	g := newTestGovernor(t)
	ws := g.policy.WorkspaceRoot
	aPath := filepath.Join(ws, "a.txt")
	bPath := filepath.Join(ws, "b.txt")
	// put
	put := newTestAction(t, "simulate.fs", map[string]any{"op": "put", "path": aPath, "content": "hi"})
	res, err := g.Execute(context.Background(), put, put.Arguments)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if !res.OK {
		t.Fatalf("put: %s", res.Error)
	}
	// scan
	scan := newTestAction(t, "simulate.fs", map[string]any{"op": "scan"})
	res, err = g.Execute(context.Background(), scan, scan.Arguments)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !res.OK {
		t.Fatalf("scan: %s", res.Error)
	}
	files, _ := res.Output["files"].([]string)
	if len(files) != 1 || files[0] != aPath {
		t.Fatalf("unexpected scan result: %v", files)
	}
	// move
	mv := newTestAction(t, "simulate.fs", map[string]any{"op": "move", "from": aPath, "to": bPath})
	res, err = g.Execute(context.Background(), mv, mv.Arguments)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if !res.OK {
		t.Fatalf("move: %s", res.Error)
	}
	// move 不存在 → 失败结果（非错误）
	mv = newTestAction(t, "simulate.fs", map[string]any{"op": "move", "from": "ghost", "to": "x"})
	res, err = g.Execute(context.Background(), mv, mv.Arguments)
	if err != nil || res.OK {
		t.Fatalf("expected failure result for missing source, got ok=%v err=%v", res.OK, err)
	}
	if !strings.Contains(res.Error, "source not found") {
		t.Fatalf("unexpected error message: %s", res.Error)
	}
	// 未知 op
	bad := newTestAction(t, "simulate.fs", map[string]any{"op": "explode"})
	res, err = g.Execute(context.Background(), bad, bad.Arguments)
	if err != nil || res.OK {
		t.Fatalf("expected failure result for unknown op, got ok=%v err=%v", res.OK, err)
	}
}

// netTool 网络权限测试工具。
type netTool struct{}

func (netTool) Name() string        { return "net.tool" }
func (netTool) Description() string { return "test network tool" }
func (netTool) Permission() Permission { return PermNetwork }
func (netTool) Risk() RiskLevel     { return RiskLow }
func (netTool) Execute(_ context.Context, _ map[string]any) (*ToolResult, error) {
	return NewToolResult(map[string]any{"ok": true}), nil
}

// TestToolGovernor_ConcurrentAccess 并发读写冒烟：注册表与执行器锁路径无数据竞争、无死锁。
func TestToolGovernor_ConcurrentAccess(t *testing.T) {
	g := NewToolGovernor(Policy{
		AllowedToolNames:   []string{"simulate.fs"},
		AllowedPermissions: []Permission{PermWriteWorkspace},
	})
	if err := g.Register(NewSimulateFSTool()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = g.List()
				_, _ = g.Lookup("simulate.fs")
				action, _ := NewActionContract("act_c", "run_c", "simulate.fs")
				action.Arguments = map[string]any{"op": "scan"}
				_, _ = g.Execute(context.Background(), action, action.Arguments)
				_ = g.RequiresApproval(action)
			}
		}()
	}
	wg.Wait()
}

package harness

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
)

// Tool 受限工具接口；Permission 与 Risk 用于执行前 Policy 检查。
type Tool interface {
	// Name 返回工具名（reTool 格式）。
	Name() string
	// Description 返回工具描述（供模型/审批预览）。
	Description() string
	// Permission 返回该工具所需的最小权限。
	Permission() Permission
	// Risk 返回该工具的风险等级。
	Risk() RiskLevel
	// Execute 执行工具。
	Execute(ctx context.Context, args map[string]any) (*ToolResult, error)
}

// ToolResult 工具执行结果。
type ToolResult struct {
	OK              bool           `json:"ok"`
	Output          map[string]any `json:"output"`
	Error           string         `json:"error,omitempty"`
	SideEffects     []string       `json:"side_effects,omitempty"`
	RollbackSummary string         `json:"rollback_summary,omitempty"`
}

// NewToolResult 构造成功结果。
func NewToolResult(output map[string]any) *ToolResult {
	return &ToolResult{OK: true, Output: output}
}

// NewToolFailure 构造失败结果。
func NewToolFailure(message string) *ToolResult {
	return &ToolResult{OK: false, Error: message}
}

// ToolGovernor 受限工具执行器：注册表 + 执行前 Policy 检查。
// 空白名单默认拒绝（最小权限）。并发安全。
type ToolGovernor struct {
	mu     sync.RWMutex
	tools  map[string]Tool
	policy Policy
}

// NewToolGovernor 构造工具执行器。
func NewToolGovernor(policy Policy) *ToolGovernor {
	return &ToolGovernor{tools: make(map[string]Tool), policy: policy}
}

// Register 注册工具；重复注册返回 CONFLICT。
func (g *ToolGovernor) Register(t Tool) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.tools[t.Name()]; ok {
		return NewHarnessError(ErrCodeConflict, fmt.Sprintf("tool already registered: %s", t.Name()), nil)
	}
	g.tools[t.Name()] = t
	return nil
}

// Lookup 查找工具；未知返回 NOT_FOUND。
func (g *ToolGovernor) Lookup(name string) (Tool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	t, ok := g.tools[name]
	if !ok {
		return nil, NewHarnessError(ErrCodeNotFound, fmt.Sprintf("tool not found: %s", name), nil)
	}
	return t, nil
}

// List 返回已注册工具名列表。
func (g *ToolGovernor) List() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, 0, len(g.tools))
	for name := range g.tools {
		names = append(names, name)
	}
	return names
}

// Execute 执行前完成 Policy 检查：工具白名单、权限、路径隔离、网络域白名单。
// 审批门（RiskHigh/Critical）由调用方（runtime）在策略选动作后检查，不在此处。
func (g *ToolGovernor) Execute(ctx context.Context, action *ActionContract, args map[string]any) (*ToolResult, error) {
	g.mu.RLock()
	tool, ok := g.tools[action.ToolName]
	policy := g.policy
	g.mu.RUnlock()
	if !ok {
		return nil, NewHarnessError(ErrCodeNotFound, fmt.Sprintf("tool not found: %s", action.ToolName), nil)
	}

	// 1) 工具名白名单：空白名单默认拒绝
	if !inStrings(action.ToolName, policy.AllowedToolNames) {
		return nil, ErrPermissionDenied(fmt.Sprintf("tool %s not in allowed tool names", action.ToolName))
	}
	// 2) 权限白名单
	if !inPerms(tool.Permission(), policy.AllowedPermissions) {
		return nil, ErrPermissionDenied(fmt.Sprintf("tool %s requires permission %s which is not allowed", action.ToolName, tool.Permission()))
	}
	// 3) 工作区路径隔离：拒绝逃逸 WorkspaceRoot
	if policy.WorkspaceRoot != "" {
		for key, val := range args {
			if !strings.Contains(key, "path") {
				continue
			}
			p, ok := val.(string)
			if !ok {
				continue
			}
			if err := checkPathInside(policy.WorkspaceRoot, p); err != nil {
				return nil, ErrPermissionDenied(fmt.Sprintf("path arg %q: %v", key, err))
			}
		}
	}
	// 4) 网络域白名单
	if len(policy.AllowNetworkDomains) > 0 {
		for key, val := range args {
			if !strings.Contains(key, "url") {
				continue
			}
			raw, ok := val.(string)
			if !ok {
				continue
			}
			if err := checkDomainAllowed(raw, policy.AllowNetworkDomains); err != nil {
				return nil, ErrPermissionDenied(fmt.Sprintf("url arg %q: %v", key, err))
			}
		}
	}
	return tool.Execute(ctx, args)
}

// RequiresApproval 判断动作是否需要人工审批（RiskHigh/RiskCritical 在 ApprovalRequiredFor 中）。
func (g *ToolGovernor) RequiresApproval(action *ActionContract) bool {
	tool, err := g.Lookup(action.ToolName)
	if err != nil {
		return true // 未知工具保守视为需审批
	}
	for _, level := range g.policy.ApprovalRequiredFor {
		if tool.Risk() == level {
			return true
		}
	}
	return false
}

// ---- 工具函数 ----

func inStrings(s string, list []string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func inPerms(p Permission, list []Permission) bool {
	for _, item := range list {
		if item == p {
			return true
		}
	}
	return false
}

// checkPathInside 校验 p 经 filepath.Abs 后仍位于 root 内（拒绝 .. 逃逸）。
func checkPathInside(root, p string) error {
	if p == "" {
		return fmt.Errorf("empty path")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absP, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absRoot, absP)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes workspace root: %s", p)
	}
	return nil
}

// checkDomainAllowed 校验 URL 的 host 在域名白名单内。
func checkDomainAllowed(raw string, domains []string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid url: %s", raw)
	}
	host := strings.ToLower(u.Hostname())
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if host == d || strings.HasSuffix(host, "."+d) {
			return nil
		}
	}
	return fmt.Errorf("host %s not in allowed domains", host)
}

// ---- 模拟回归工具（领域无关性验证样例） ----

// simulateFSTool 模拟文件系统整理工具：纯内存模拟，验证核心运行时不依赖领域对象。
type simulateFSTool struct {
	mu sync.Mutex
	// files 记录当前目录结构：相对路径 -> 模拟内容
	files map[string]string
}

// NewSimulateFSTool 构造模拟文件整理工具（task6 回归样例；task7 正式 file_organizer 挂接）。
func NewSimulateFSTool() Tool {
	return &simulateFSTool{files: map[string]string{}}
}

func (t *simulateFSTool) Name() string        { return "simulate.fs" }
func (t *simulateFSTool) Description() string { return "模拟文件系统：scan/move/copy，不触碰真实文件" }
func (t *simulateFSTool) Permission() Permission { return PermWriteWorkspace }
func (t *simulateFSTool) Risk() RiskLevel     { return RiskLow }

func (t *simulateFSTool) Execute(_ context.Context, args map[string]any) (*ToolResult, error) {
	op, _ := args["op"].(string)
	t.mu.Lock()
	defer t.mu.Unlock()
	switch op {
	case "scan":
		out := make([]string, 0, len(t.files))
		for p := range t.files {
			out = append(out, p)
		}
		return NewToolResult(map[string]any{"files": out}), nil
	case "put":
		p, _ := args["path"].(string)
		content, _ := args["content"].(string)
		if p == "" {
			return NewToolFailure("put requires path"), nil
		}
		t.files[p] = content
		return NewToolResult(map[string]any{"path": p, "bytes": len(content)}), nil
	case "move":
		from, _ := args["from"].(string)
		to, _ := args["to"].(string)
		content, ok := t.files[from]
		if !ok {
			return NewToolFailure(fmt.Sprintf("source not found: %s", from)), nil
		}
		delete(t.files, from)
		t.files[to] = content
		return NewToolResult(map[string]any{"from": from, "to": to}), nil
	default:
		return NewToolFailure(fmt.Sprintf("unknown op: %s", op)), nil
	}
}

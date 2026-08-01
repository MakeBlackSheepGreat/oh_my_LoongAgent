// Package skills 提供领域 Skill 的注册与执行适配层。
// 领域能力（论文搜索、全文归档、导出、文件整理等）全部位于本包及子包；
// 核心 harness 只提供 SkillManifest 契约、工具执行器与验证器注册表，不依赖任何领域对象（D-009）。
package skills

import (
	"context"
	"fmt"
	"sync"
	"time"

	"slim-agent/internal/harness"
	"slim-agent/internal/skills/adapters"
)

// Request Skill 执行请求：任务契约 + 领域输入。
type Request struct {
	// Task 任务契约（TaskID 即运行 ID，工件写入该运行）。
	Task *harness.TaskContract
	// Inputs 领域输入（研究问题、来源集合、导出要求等）。
	Inputs map[string]any
}

// Env 执行环境：由调用方装配注入；Skill 不持有全局状态。
type Env struct {
	// Store 核心工件库（归档与导出写入点）。
	Store *harness.HarnessStore
	// Tools 受限工具执行器（领域工具挂载点，可 nil）。
	Tools *harness.ToolGovernor
	// Validators 验证器注册表（领域验证器挂载点，可 nil）。
	Validators *harness.ValidatorRegistry
	// HTTP 共用的外部访问客户端。
	HTTP *adapters.Client
	// Now 可注入时钟（测试用）；nil 用 time.Now().UTC()。
	Now func() time.Time
}

// NowUTC 返回 Env 时钟当前 UTC 时间。
func (e *Env) NowUTC() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

// Result Skill 执行结果：工件、事件与验证结果。
type Result struct {
	Artifacts   []*harness.Artifact
	Events      []*harness.Event
	Validations []*harness.ValidatorResult
}

// AddArtifact 追加一个工件。
func (r *Result) AddArtifact(a *harness.Artifact) { r.Artifacts = append(r.Artifacts, a) }

// AddEvent 追加一个事件。
func (r *Result) AddEvent(ev *harness.Event) { r.Events = append(r.Events, ev) }

// AddValidation 追加一个验证结果。
func (r *Result) AddValidation(v *harness.ValidatorResult) { r.Validations = append(r.Validations, v) }

// Skill 领域 Skill 接口。
type Skill interface {
	// Manifest 返回 Skill 声明（skill_id/version/input_schema/output_kinds/...）。
	Manifest() *harness.SkillManifest
	// Execute 执行领域流程并产出工件、事件与验证结果。
	Execute(ctx context.Context, req *Request, env *Env) (*Result, error)
}

// Registry 并发安全的 Skill 注册表。
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewRegistry 构造空注册表。
func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]Skill)}
}

// Register 注册 Skill；重复注册返回 CONFLICT。
func (r *Registry) Register(s Skill) error {
	id := s.Manifest().SkillID
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.skills[id]; ok {
		return harness.NewHarnessError(harness.ErrCodeConflict,
			fmt.Sprintf("skill already registered: %s", id), nil)
	}
	r.skills[id] = s
	return nil
}

// Lookup 查找 Skill；未知返回 NOT_FOUND。
func (r *Registry) Lookup(id string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[id]
	if !ok {
		return nil, harness.NewHarnessError(harness.ErrCodeNotFound,
			fmt.Sprintf("skill not found: %s", id), nil)
	}
	return s, nil
}

// List 返回已注册 Skill 的 manifest 列表（顺序稳定）。
func (r *Registry) List() []*harness.SkillManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*harness.SkillManifest, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, s.Manifest())
	}
	return out
}

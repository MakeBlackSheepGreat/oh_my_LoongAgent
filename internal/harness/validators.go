package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Validator 验证器接口；验证结果落库并参与路由决策。
type Validator interface {
	// ID 返回验证器标识（reKind 格式）。
	ID() string
	// Version 返回验证器版本。
	Version() string
	// Validate 对运行状态与工件集合执行验证。
	Validate(ctx context.Context, run *RunState, artifacts map[string]*Artifact) (*ValidatorResult, error)
}

// ValidatorRegistry 验证器注册表；并发安全。
type ValidatorRegistry struct {
	mu         sync.RWMutex
	validators map[string]Validator
}

// NewValidatorRegistry 构造空注册表。
func NewValidatorRegistry() *ValidatorRegistry {
	return &ValidatorRegistry{validators: make(map[string]Validator)}
}

// Register 注册验证器；重复注册返回 CONFLICT。
func (r *ValidatorRegistry) Register(v Validator) error {
	id := v.ID()
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.validators[id]; ok {
		return NewHarnessError(ErrCodeConflict, fmt.Sprintf("validator already registered: %s", id), nil)
	}
	r.validators[id] = v
	return nil
}

// Lookup 查找验证器；未知返回 NOT_FOUND。
func (r *ValidatorRegistry) Lookup(id string) (Validator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.validators[id]
	if !ok {
		return nil, NewHarnessError(ErrCodeNotFound, fmt.Sprintf("validator not found: %s", id), nil)
	}
	return v, nil
}

// List 返回已注册验证器 ID（排序稳定）。
func (r *ValidatorRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.validators))
	for id := range r.validators {
		ids = append(ids, id)
	}
	return ids
}

// RunAll 对全部注册验证器执行验证并聚合结果。
// 任一验证器失败即整体失败；结果按 validator_id 排序，失败优先。
func (r *ValidatorRegistry) RunAll(ctx context.Context, run *RunState, artifacts map[string]*Artifact) ([]*ValidatorResult, error) {
	r.mu.RLock()
	validators := make([]Validator, 0, len(r.validators))
	for _, v := range r.validators {
		validators = append(validators, v)
	}
	r.mu.RUnlock()

	results := make([]*ValidatorResult, 0, len(validators))
	for _, v := range validators {
		vr, err := v.Validate(ctx, run, artifacts)
		if err != nil {
			return nil, err
		}
		results = append(results, vr)
	}
	// 排序：失败的在前（稳定决策顺序）
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && !results[j].Passed && results[j-1].Passed; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	return results, nil
}

// ---- 内建验证器 ----

// artifactExistsValidator 工件存在性验证：run.ArtifactIDs 全部存在于工件集合。
type artifactExistsValidator struct{}

// NewArtifactExistsValidator 构造工件存在性验证器。
func NewArtifactExistsValidator() Validator { return artifactExistsValidator{} }

func (artifactExistsValidator) ID() string      { return "artifact_exists" }
func (artifactExistsValidator) Version() string { return "v1" }

func (artifactExistsValidator) Validate(_ context.Context, run *RunState, artifacts map[string]*Artifact) (*ValidatorResult, error) {
	vr, err := NewValidatorResult("artifact_exists", run.RunID, true)
	if err != nil {
		return nil, err
	}
	for _, id := range run.ArtifactIDs {
		if _, ok := artifacts[id]; !ok {
			vr.Passed = false
			vr.Confidence = 1.0
			vr.Findings = append(vr.Findings, fmt.Sprintf("artifact missing: %s", id))
		}
	}
	vr.ArtifactIDs = append(vr.ArtifactIDs, run.ArtifactIDs...)
	return vr, nil
}

// jsonSchemaValidator JSON 合法性验证：kind 以 json 结尾或 metadata 声明 json_schema 的工件必须可解析。
// 内建版本仅校验合法 JSON（避免第三方 schema 库）；领域 Schema 校验由 Skill 注册专用验证器。
type jsonSchemaValidator struct{}

// NewJSONSchemaValidator 构造 JSON 合法性验证器。
func NewJSONSchemaValidator() Validator { return jsonSchemaValidator{} }

func (jsonSchemaValidator) ID() string      { return "json_schema" }
func (jsonSchemaValidator) Version() string { return "v1" }

func (jsonSchemaValidator) Validate(_ context.Context, run *RunState, artifacts map[string]*Artifact) (*ValidatorResult, error) {
	vr, err := NewValidatorResult("json_schema", run.RunID, true)
	if err != nil {
		return nil, err
	}
	for _, id := range run.ArtifactIDs {
		art, ok := artifacts[id]
		if !ok {
			continue // 存在性由 artifact_exists 负责
		}
		if !isJSONKind(art) {
			continue
		}
		content, err := readArtifactBytes(art)
		if err != nil {
			vr.Passed = false
			vr.Findings = append(vr.Findings, fmt.Sprintf("artifact %s unreadable: %v", id, err))
			continue
		}
		if !json.Valid(content) {
			vr.Passed = false
			vr.Confidence = 1.0
			vr.Findings = append(vr.Findings, fmt.Sprintf("artifact %s is not valid JSON", id))
		}
	}
	return vr, nil
}

// referenceIntegrityValidator 引用完整性验证：工件 metadata["references"] 指向的 artifact_id 必须存在于 run.ArtifactIDs。
type referenceIntegrityValidator struct{}

// NewReferenceIntegrityValidator 构造引用完整性验证器。
func NewReferenceIntegrityValidator() Validator { return referenceIntegrityValidator{} }

func (referenceIntegrityValidator) ID() string      { return "reference_integrity" }
func (referenceIntegrityValidator) Version() string { return "v1" }

func (referenceIntegrityValidator) Validate(_ context.Context, run *RunState, artifacts map[string]*Artifact) (*ValidatorResult, error) {
	vr, err := NewValidatorResult("reference_integrity", run.RunID, true)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]bool, len(run.ArtifactIDs))
	for _, id := range run.ArtifactIDs {
		existing[id] = true
	}
	for _, id := range run.ArtifactIDs {
		art, ok := artifacts[id]
		if !ok {
			continue
		}
		refs, err := artifactReferences(art)
		if err != nil {
			vr.Passed = false
			vr.Findings = append(vr.Findings, fmt.Sprintf("artifact %s: %v", id, err))
			continue
		}
		for _, ref := range refs {
			if !existing[ref] {
				vr.Passed = false
				vr.Findings = append(vr.Findings, fmt.Sprintf("artifact %s references missing artifact: %s", id, ref))
			}
		}
	}
	return vr, nil
}

// budgetValidator 预算约束验证：run.Usage 未超 Budget。
type budgetValidator struct{}

// NewBudgetValidator 构造预算约束验证器。
func NewBudgetValidator() Validator { return budgetValidator{} }

func (budgetValidator) ID() string      { return "budget_constraint" }
func (budgetValidator) Version() string { return "v1" }

func (budgetValidator) Validate(_ context.Context, run *RunState, _ map[string]*Artifact) (*ValidatorResult, error) {
	vr, err := NewValidatorResult("budget_constraint", run.RunID, true)
	if err != nil {
		return nil, err
	}
	b := run.Task.Budget
	if calls := int(run.Usage["model_calls"]); calls > b.MaxModelCalls {
		vr.Passed = false
		vr.Findings = append(vr.Findings, fmt.Sprintf("model_calls %d exceeds budget %d", calls, b.MaxModelCalls))
	}
	if calls := int(run.Usage["tool_calls"]); calls > b.MaxToolCalls {
		vr.Passed = false
		vr.Findings = append(vr.Findings, fmt.Sprintf("tool_calls %d exceeds budget %d", calls, b.MaxToolCalls))
	}
	if cost := run.Usage["cost_usd"]; b.MaxCostUSD != nil && cost > *b.MaxCostUSD {
		vr.Passed = false
		vr.Findings = append(vr.Findings, fmt.Sprintf("cost %.4f exceeds budget %.4f", cost, *b.MaxCostUSD))
	}
	return vr, nil
}

// ---- 工具函数 ----

// isJSONKind 判断工件是否为 JSON 类（kind 以 .json 结尾或包含 json 标记）。
func isJSONKind(art *Artifact) bool {
	if len(art.Kind) >= 5 && art.Kind[len(art.Kind)-5:] == ".json" {
		return true
	}
	ct, _ := art.Metadata["content_type"].(string)
	return ct == "application/json" || ct == "text/json"
}

// readArtifactBytes 读取工件内容；优先从元数据缓存，否则读存储。
func readArtifactBytes(art *Artifact) ([]byte, error) {
	if b, ok := art.Metadata["_bytes"].([]byte); ok {
		return b, nil
	}
	if s, ok := art.Metadata["_text"].(string); ok {
		return []byte(s), nil
	}
	return nil, errors.New("artifact content not available in memory")
}

// artifactReferences 提取工件 metadata["references"] 的 artifact_id 列表。
func artifactReferences(art *Artifact) ([]string, error) {
	raw, ok := art.Metadata["references"]
	if !ok {
		return nil, nil
	}
	switch refs := raw.(type) {
	case []any:
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			s, ok := ref.(string)
			if !ok {
				return nil, fmt.Errorf("reference must be a string, got %T", ref)
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		return refs, nil
	default:
		return nil, fmt.Errorf("references must be a list, got %T", raw)
	}
}

// AggregateValidation 便捷聚合：任一验证结果失败则失败。
func AggregateValidation(results []*ValidatorResult) *ValidatorResult {
	passed := true
	var findings []string
	for _, vr := range results {
		if !vr.Passed {
			passed = false
			findings = append(findings, vr.Findings...)
		}
	}
	return &ValidatorResult{
		ValidatorID:      "aggregate",
		ValidatorVersion: "v1",
		Passed:           passed,
		Confidence:       1.0,
		Findings:         findings,
		CreatedAt:        time.Now().UTC(),
	}
}

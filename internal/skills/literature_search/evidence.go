package literature_search

import (
	"context"
	"fmt"

	"slim-agent/internal/harness"
)

// registerEvidenceValidator 将证据核验验证器挂载进 harness 验证器注册表。
// 重复挂载由 Registry 的 CONFLICT 保护（调用方保证只注册一次）。
func registerEvidenceValidator(reg *harness.ValidatorRegistry) error {
	return reg.Register(evidenceValidator{})
}

// evidenceValidator 证据完整性核验：归档证据工件必须含来源 URL、哈希与候选 ID 元数据。
// 未核验/缺字段的引用返回验证失败与结构化 Findings。
type evidenceValidator struct{}

func (evidenceValidator) ID() string      { return "evidence_integrity" }
func (evidenceValidator) Version() string { return "v1" }

func (evidenceValidator) Validate(_ context.Context, run *harness.RunState, artifacts map[string]*harness.Artifact) (*harness.ValidatorResult, error) {
	vr, err := harness.NewValidatorResult("evidence_integrity", run.RunID, true)
	if err != nil {
		return nil, err
	}
	for _, id := range run.ArtifactIDs {
		art, ok := artifacts[id]
		if !ok || art.Kind != "evidence.archive" {
			continue
		}
		if s, _ := art.Metadata["source_url"].(string); s == "" {
			vr.Passed = false
			vr.Findings = append(vr.Findings, fmt.Sprintf("evidence %s missing source_url", id))
		}
		if s, _ := art.Metadata["sha256"].(string); s == "" {
			vr.Passed = false
			vr.Findings = append(vr.Findings, fmt.Sprintf("evidence %s missing sha256", id))
		}
		if s, _ := art.Metadata["candidate_id"].(string); s == "" {
			vr.Passed = false
			vr.Findings = append(vr.Findings, fmt.Sprintf("evidence %s missing candidate_id", id))
		}
	}
	return vr, nil
}

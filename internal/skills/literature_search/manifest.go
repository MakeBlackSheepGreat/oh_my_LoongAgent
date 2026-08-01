// Package literature_search 首发领域 Skill：跨四类来源搜索文献、去重溯源、
// 归档开放全文、核验证据并导出报告。领域能力全部位于本包（D-009）。
package literature_search

import (
	"context"
	"fmt"
	"strings"

	"slim-agent/internal/harness"
	"slim-agent/internal/harness/errs"
	"slim-agent/internal/skills"
	"slim-agent/internal/skills/adapters"
)

const (
	// SkillID 领域 Skill 标识。
	SkillID = "literature_search"
	// Version 版本。
	Version = "v1"
	// defaultMaxResults 默认最大结果数。
	defaultMaxResults = 20
	// maxResultsLimit 单来源结果上限。
	maxResultsLimit = 100
)

// 来源与导出格式白名单。
var (
	allowedSources = map[string]bool{"arxiv": true, "openalex": true, "crossref": true, "acl": true}
	allowedExports = map[string]bool{"markdown": true, "json": true, "csv": true, "manifest": true}

	// defaultSourceBases 官方接口基地址；测试通过 source_base_urls 覆盖。
	defaultSourceBases = map[string]string{
		"arxiv":    "https://export.arxiv.org",
		"openalex": "https://api.openalex.org",
		"crossref": "https://api.crossref.org",
		"acl":      "", // ACL Anthology 暂无公开搜索 API，留空表示跳过
	}
)

// Skill literature_search 主实现。
type Skill struct{}

// NewSkill 构造 literature_search Skill。
func NewSkill() skills.Skill { return &Skill{} }

// Manifest 返回 Skill 声明。
func (s *Skill) Manifest() *harness.SkillManifest {
	return &harness.SkillManifest{
		SkillID:   SkillID,
		Version:   Version,
		Title:     "Literature Search",
		Description: "跨 arXiv/OpenAlex/Crossref 搜索文献（ACL 暂不支持公开 API），去重溯源、归档开放全文、核验证据并导出报告。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"research_question": map[string]any{"type": "string"},
				"sources":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"max_results":       map[string]any{"type": "integer"},
				"export":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"research_question"},
		},
		OutputArtifactKinds: []string{
			"evidence.archive", "report.markdown", "report.json", "report.csv", "archive.manifest",
		},
		RequiredTools:      []string{},
		RequiredValidators: []string{"evidence_integrity"},
		DefaultBudget:      harness.DefaultBudget(),
		Metadata:           map[string]any{},
	}
}

// searchInput 校验后的搜索输入。
type searchInput struct {
	ResearchQuestion string
	Sources          []string
	MaxResults       int
	Exports          []string
	SourceBases      map[string]string
}

// parseInput 校验并归一化任务输入；失败返回 VALIDATION_ERROR。
func parseInput(inputs map[string]any) (*searchInput, error) {
	if inputs == nil {
		inputs = map[string]any{}
	}
	raw, ok := inputs["research_question"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, errs.ErrValidation("research_question is required and must be non-empty", nil)
	}
	if len(raw) > 10000 {
		return nil, errs.ErrValidation("research_question too long", nil)
	}

	in := &searchInput{
		ResearchQuestion: strings.TrimSpace(raw),
		Sources:          sortedKeys(allowedSources),
		MaxResults:       defaultMaxResults,
		Exports:          []string{"markdown", "json", "manifest"},
		SourceBases:      map[string]string{},
	}

	if v, ok := inputs["max_results"]; ok {
		n, err := toInt(v)
		if err != nil || n < 1 || n > maxResultsLimit {
			return nil, errs.ErrValidation(fmt.Sprintf("max_results out of range [1,%d]", maxResultsLimit), nil)
		}
		in.MaxResults = n
	}
	if v, ok := inputs["sources"]; ok {
		srcs, err := toStringSlice(v)
		if err != nil {
			return nil, errs.ErrValidation("sources must be a list of strings", err)
		}
		if len(srcs) == 0 {
			return nil, errs.ErrValidation("sources must be non-empty", nil)
		}
		for _, src := range srcs {
			if !allowedSources[src] {
				return nil, errs.ErrValidation(fmt.Sprintf("unknown source: %s", src), nil)
			}
		}
		in.Sources = srcs
	}
	if v, ok := inputs["export"]; ok {
		exps, err := toStringSlice(v)
		if err != nil {
			return nil, errs.ErrValidation("export must be a list of strings", err)
		}
		for _, e := range exps {
			if !allowedExports[e] {
				return nil, errs.ErrValidation(fmt.Sprintf("unknown export format: %s", e), nil)
			}
		}
		if len(exps) > 0 {
			in.Exports = exps
		}
	}
	if v, ok := inputs["source_base_urls"]; ok {
		bases, ok := v.(map[string]any)
		if !ok {
			return nil, errs.ErrValidation("source_base_urls must be an object", nil)
		}
		for k, val := range bases {
			s, ok := val.(string)
			if !ok {
				return nil, errs.ErrValidation(fmt.Sprintf("source_base_urls[%s] must be a string", k), nil)
			}
			in.SourceBases[k] = strings.TrimRight(s, "/")
		}
	}
	return in, nil
}

// Execute 执行搜索→去重→归档→核验→导出全流程。
func (s *Skill) Execute(ctx context.Context, req *skills.Request, env *skills.Env) (*skills.Result, error) {
	result := &skills.Result{}
	in, err := parseInput(req.Inputs)
	if err != nil {
		return nil, err
	}
	if env == nil || env.Store == nil {
		return nil, errs.ErrValidation("skill env with store is required", nil)
	}
	runID := req.Task.TaskID
	client := env.HTTP
	if client == nil {
		client = adapters.NewClient(0, 0)
	}

	// 1) 来源搜索（单来源失败降级继续，全部失败才报错）
	var records []Record
	var sourceErrs []string
	for _, name := range in.Sources {
		base, ok := in.SourceBases[name]
		if !ok {
			base = defaultSourceBases[name]
		}
		if base == "" {
			sourceErrs = append(sourceErrs, name+": no base url configured")
			continue
		}
		src, err := NewSource(name, base, client)
		if err != nil {
			sourceErrs = append(sourceErrs, err.Error())
			continue
		}
		got, err := src.Search(ctx, in.ResearchQuestion, in.MaxResults)
		if err != nil {
			sourceErrs = append(sourceErrs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		records = append(records, got...)
		result.AddEvent(newEvent(runID, "source_searched", fmt.Sprintf("source %s returned %d records", name, len(got)),
			map[string]any{"source": name, "count": len(got)}))
	}
	if len(records) == 0 {
		if len(sourceErrs) > 0 {
			return nil, errs.NewHarnessError(errs.ErrCodeProviderUnavailable,
				"all sources failed: "+strings.Join(sourceErrs, "; "), nil)
		}
		return nil, errs.ErrValidation("no records found", nil)
	}

	// 2) 去重与溯源
	cands := Deduplicate(records)
	result.AddEvent(newEvent(runID, "dedup_completed",
		fmt.Sprintf("%d records merged into %d candidates", len(records), len(cands)), nil))

	// 3) 开放全文归档
	archived := 0
	for _, c := range cands {
		if c.OpenAccessURL == "" {
			continue
		}
		art, err := ArchiveFullText(ctx, client, env.Store, runID, c, env.NowUTC())
		if err != nil {
			result.AddEvent(newEvent(runID, "archive_failed",
				fmt.Sprintf("archive failed for %s: %v", c.ID, err), map[string]any{"candidate_id": c.ID}))
			continue
		}
		c.Evidence = append(c.Evidence, "archived:"+art.ArtifactID)
		result.AddArtifact(art)
		archived++
	}
	result.AddEvent(newEvent(runID, "archive_completed",
		fmt.Sprintf("%d/%d candidates archived", archived, len(cands)), nil))

	// 4) 证据核验（验证器挂在 Env 上，执行后汇总）
	if env.Validators != nil {
		if err := registerEvidenceValidator(env.Validators); err != nil {
			return nil, err
		}
		state, gerr := env.Store.GetRun(runID)
		if gerr != nil {
			return nil, gerr
		}
		artifacts, aerr := loadRunArtifacts(env.Store, state)
		if aerr != nil {
			return nil, aerr
		}
		results, verr := env.Validators.RunAll(ctx, state, artifacts)
		if verr != nil {
			return nil, verr
		}
		for _, vr := range results {
			result.AddValidation(vr)
		}
	}

	// 5) 导出
	exports := buildExports(cands, archived)
	for _, format := range in.Exports {
		content, err := exports.render(format, req)
		if err != nil {
			result.AddEvent(newEvent(runID, "export_failed", fmt.Sprintf("export %s: %v", format, err), nil))
			continue
		}
		ai, err := harness.NewArtifactInput(exportArtifactID(format), "report."+format, []byte(content))
		if err != nil {
			return nil, err
		}
		art, err := env.Store.PutArtifact(runID, ai)
		if err != nil {
			return nil, err
		}
		result.AddArtifact(art)
		result.AddEvent(newEvent(runID, "export_completed", fmt.Sprintf("export %s written", format),
			map[string]any{"artifact_id": art.ArtifactID, "bytes": art.SizeBytes}))
	}
	return result, nil
}

// Package file_organizer 领域无关性回归样例：模拟文件整理 Skill。
// 只依赖 harness 工件库与标准库，证明核心运行时不依赖任何垂直领域对象（D-009）。
package file_organizer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"slim-agent/internal/harness"
	"slim-agent/internal/harness/errs"
	"slim-agent/internal/skills"
)

const (
	// SkillID 领域 Skill 标识。
	SkillID = "file_organizer"
	// Version 版本。
	Version = "v1"
)

// Skill file_organizer 主实现。
type Skill struct{}

// NewSkill 构造 file_organizer Skill。
func NewSkill() skills.Skill { return &Skill{} }

// Manifest 返回 Skill 声明。
func (s *Skill) Manifest() *harness.SkillManifest {
	return &harness.SkillManifest{
		SkillID:   SkillID,
		Version:   Version,
		Title:     "File Organizer",
		Description: "扫描目录并按规则分类文件，产出整理计划（只读模拟，不触碰工作区外路径）。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspace_dir": map[string]any{"type": "string"},
			},
			"required": []string{"workspace_dir"},
		},
		OutputArtifactKinds: []string{"organize.plan"},
		RequiredTools:       []string{"simulate.fs"},
		RequiredValidators:  []string{"plan_integrity"},
		DefaultBudget:       harness.DefaultBudget(),
		Metadata:            map[string]any{},
	}
}

// Execute 扫描目录→按扩展名分类→输出整理计划工件。
func (s *Skill) Execute(ctx context.Context, req *skills.Request, env *skills.Env) (*skills.Result, error) {
	result := &skills.Result{}
	if env == nil || env.Store == nil {
		return nil, errs.ErrValidation("skill env with store is required", nil)
	}
	dir, _ := req.Inputs["workspace_dir"].(string)
	if strings.TrimSpace(dir) == "" {
		return nil, errs.ErrValidation("workspace_dir is required and must be non-empty", nil)
	}
	content, err := buildPlan(ctx, dir)
	if err != nil {
		return nil, err
	}
	ai, err := harness.NewArtifactInput("organize_plan", "organize.plan", content)
	if err != nil {
		return nil, err
	}
	art, err := env.Store.PutArtifact(req.Task.TaskID, ai)
	if err != nil {
		return nil, err
	}
	result.AddArtifact(art)
	result.AddEvent(&harness.Event{
		Sequence:  1,
		RunID:     req.Task.TaskID,
		Kind:      "plan_written",
		Message:   fmt.Sprintf("organize plan written with %d files", planFileCount(content)),
		Payload:   map[string]any{"artifact_id": art.ArtifactID},
		Timestamp: time.Now().UTC(),
	})
	return result, nil
}

// fileEntry 单个文件的整理条目。
type fileEntry struct {
	Path      string `json:"path"`
	Ext       string `json:"ext"`
	SizeBytes int64  `json:"size_bytes"`
}

// buildPlan 只读扫描目录并生成按分类分组的整理计划（JSON）。
func buildPlan(ctx context.Context, root string) ([]byte, error) {
	byCategory := map[string][]fileEntry{}
	total := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		total++
		ext := filepath.Ext(d.Name())
		info, ierr := d.Info()
		var size int64
		if ierr == nil {
			size = info.Size()
		}
		cat := categoryFor(ext)
		byCategory[cat] = append(byCategory[cat], fileEntry{Path: path, Ext: ext, SizeBytes: size})
		return nil
	})
	if err != nil {
		return nil, errs.NewHarnessError(errs.ErrCodeInternal, "scan directory failed", err)
	}
	out := struct {
		Root        string               `json:"root"`
		GeneratedAt string               `json:"generated_at"`
		TotalFiles  int                  `json:"total_files"`
		ByCategory  map[string][]fileEntry `json:"by_category"`
	}{
		Root:        root,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		TotalFiles:  total,
		ByCategory:  byCategory,
	}
	return json.MarshalIndent(out, "", "  ")
}

// categoryFor 按扩展名分类（确定性规则）。
func categoryFor(ext string) string {
	switch strings.ToLower(ext) {
	case ".pdf", ".doc", ".docx", ".txt":
		return "documents"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "images"
	case ".py", ".go", ".ts", ".js", ".java":
		return "code"
	case ".csv", ".xlsx", ".json":
		return "data"
	default:
		return "other"
	}
}

// planFileCount 从计划 JSON 提取文件数（事件消息用）。
func planFileCount(content []byte) int {
	var probe struct {
		TotalFiles int `json:"total_files"`
	}
	_ = json.Unmarshal(content, &probe)
	return probe.TotalFiles
}

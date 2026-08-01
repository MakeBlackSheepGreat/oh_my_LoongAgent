package file_organizer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"slim-agent/internal/harness"
	"slim-agent/internal/harness/errs"
	"slim-agent/internal/skills"
)

func TestSkill_Execute_ProducesPlan(t *testing.T) {
	store := harness.NewHarnessStore(t.TempDir())
	if err := store.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	// 临时工作区（只读扫描，不触碰工作区外路径）
	ws := t.TempDir()
	for name, content := range map[string]string{
		"report.pdf": "pdf-bytes", "photo.png": "img-bytes",
		"main.go": "go-bytes", "data.csv": "csv-bytes", "notes.txt": "txt-bytes",
	} {
		if err := os.WriteFile(filepath.Join(ws, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	task, err := harness.NewTaskContract("run_org1", SkillID, "organize my files")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	env := &skills.Env{Store: store}
	skill := NewSkill()
	result, err := skill.Execute(context.Background(), &skills.Request{
		Task:   task,
		Inputs: map[string]any{"workspace_dir": ws},
	}, env)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected 1 plan artifact, got %d", len(result.Artifacts))
	}
	art := result.Artifacts[0]
	if art.Kind != "organize.plan" {
		t.Fatalf("unexpected kind: %s", art.Kind)
	}
	content, err := store.ReadArtifact(task.TaskID, art.ArtifactID)
	if err != nil {
		t.Fatalf("ReadArtifact: %v", err)
	}
	plan := string(content)
	for _, want := range []string{`"documents"`, `"images"`, `"code"`, `"data"`, `"total_files": 5`} {
		if !strings.Contains(plan, want) {
			t.Fatalf("plan missing %q:\n%s", want, plan)
		}
	}
}

func TestSkill_Execute_MissingWorkspaceDir(t *testing.T) {
	store := harness.NewHarnessStore(t.TempDir())
	if err := store.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	task, err := harness.NewTaskContract("run_org2", SkillID, "organize")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	skill := NewSkill()
	_, err = skill.Execute(context.Background(), &skills.Request{Task: task, Inputs: map[string]any{}},
		&skills.Env{Store: store})
	he, ok := err.(*errs.HarnessError)
	if !ok || he.Code != errs.ErrCodeValidation {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestCategoryFor(t *testing.T) {
	cases := map[string]string{
		".pdf": "documents", ".txt": "documents",
		".png": "images", ".jpg": "images",
		".go": "code", ".py": "code",
		".csv": "data", ".json": "data",
		".xyz": "other",
	}
	for ext, want := range cases {
		if got := categoryFor(ext); got != want {
			t.Fatalf("categoryFor(%s) = %s, want %s", ext, got, want)
		}
	}
}

func TestManifest(t *testing.T) {
	m := NewSkill().Manifest()
	if m.SkillID != SkillID || m.Version != Version {
		t.Fatalf("unexpected manifest identity: %s/%s", m.SkillID, m.Version)
	}
	if len(m.OutputArtifactKinds) != 1 || m.OutputArtifactKinds[0] != "organize.plan" {
		t.Fatalf("unexpected output kinds: %v", m.OutputArtifactKinds)
	}
}

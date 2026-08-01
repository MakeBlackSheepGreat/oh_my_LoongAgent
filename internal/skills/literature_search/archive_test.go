package literature_search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"slim-agent/internal/harness"
	"slim-agent/internal/skills"
	"slim-agent/internal/skills/adapters"
)

// newSkillEnv 构造带内存工件库的 Skill 执行环境。
func newSkillEnv(t *testing.T) (*skills.Env, *harness.HarnessStore) {
	t.Helper()
	store := harness.NewHarnessStore(t.TempDir())
	if err := store.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	env := &skills.Env{
		Store:      store,
		Validators: harness.NewValidatorRegistry(),
		HTTP:       adapters.NewClient(0, 0),
		Now:        func() time.Time { return time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) },
	}
	return env, store
}

// newLitTask 构造 literature_search 任务契约。
func newLitTask(t *testing.T, runID string) *harness.TaskContract {
	t.Helper()
	task, err := harness.NewTaskContract(runID, SkillID, "search for papers about MoE")
	if err != nil {
		t.Fatalf("NewTaskContract: %v", err)
	}
	return task
}

func TestArchiveFullText(t *testing.T) {
	env, store := newSkillEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("full text content of the paper"))
	}))
	defer srv.Close()

	task := newLitTask(t, "run_arc1")
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	c := &Candidate{ID: "arxiv:A1", OpenAccessURL: srv.URL}
	art, err := ArchiveFullText(context.Background(), env.HTTP, store, task.TaskID, c, env.NowUTC())
	if err != nil {
		t.Fatalf("ArchiveFullText: %v", err)
	}
	if art == nil {
		t.Fatal("expected archived artifact")
	}
	if art.Kind != "evidence.archive" {
		t.Fatalf("unexpected kind: %s", art.Kind)
	}
	if art.Metadata["sha256"] == "" || art.Metadata["source_url"] != srv.URL {
		t.Fatalf("missing metadata: %+v", art.Metadata)
	}
	// 工件自动追加到 run.ArtifactIDs
	state, _ := store.GetRun(task.TaskID)
	if len(state.ArtifactIDs) != 1 || state.ArtifactIDs[0] != art.ArtifactID {
		t.Fatalf("artifact not registered on run: %v", state.ArtifactIDs)
	}
}

func TestArchiveFullText_UnknownMIME(t *testing.T) {
	env, store := newSkillEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0xde, 0xad, 0xbe, 0xef})
	}))
	defer srv.Close()

	task := newLitTask(t, "run_arc2")
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	c := &Candidate{ID: "arxiv:A2", OpenAccessURL: srv.URL}
	art, err := ArchiveFullText(context.Background(), env.HTTP, store, task.TaskID, c, env.NowUTC())
	if err != nil {
		t.Fatalf("ArchiveFullText: %v", err)
	}
	if art != nil {
		t.Fatal("expected nil artifact for unknown MIME")
	}
}

func TestEvidenceValidator(t *testing.T) {
	env, store := newSkillEnv(t)
	task := newLitTask(t, "run_ev1")
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if err := registerEvidenceValidator(env.Validators); err != nil {
		t.Fatalf("register: %v", err)
	}

	// 完整证据工件 → 通过
	ai, err := harness.NewArtifactInput("ev_ok", "evidence.archive", []byte("x"))
	if err != nil {
		t.Fatalf("NewArtifactInput: %v", err)
	}
	ai.Metadata = map[string]any{
		"source_url": "https://example.org/1.pdf", "sha256": "aaa", "candidate_id": "arxiv:A1",
	}
	if _, err := store.PutArtifact(task.TaskID, ai); err != nil {
		t.Fatalf("PutArtifact: %v", err)
	}
	state, _ := store.GetRun(task.TaskID)
	artifacts, err := loadRunArtifacts(store, state)
	if err != nil {
		t.Fatalf("loadRunArtifacts: %v", err)
	}
	results, err := env.Validators.RunAll(context.Background(), state, artifacts)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if !results[0].Passed {
		t.Fatalf("expected pass, got %v", results[0].Findings)
	}

	// 缺 sha256 与 candidate_id → 失败
	ai2, err := harness.NewArtifactInput("ev_bad", "evidence.archive", []byte("y"))
	if err != nil {
		t.Fatalf("NewArtifactInput: %v", err)
	}
	ai2.Metadata = map[string]any{"source_url": "https://example.org/2.pdf"}
	if _, err := store.PutArtifact(task.TaskID, ai2); err != nil {
		t.Fatalf("PutArtifact: %v", err)
	}
	state, _ = store.GetRun(task.TaskID)
	artifacts, err = loadRunArtifacts(store, state)
	if err != nil {
		t.Fatalf("loadRunArtifacts: %v", err)
	}
	results, err = env.Validators.RunAll(context.Background(), state, artifacts)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if results[0].Passed {
		t.Fatal("expected failure for incomplete evidence")
	}
	if len(results[0].Findings) == 0 {
		t.Fatal("expected structured findings")
	}
}

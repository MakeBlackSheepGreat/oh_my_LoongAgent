package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *HarnessStore {
	t.Helper()
	dir := t.TempDir()
	store := NewHarnessStore(dir)
	if err := store.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestCreateRunAndGetRun(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_001", "lit_search", "find papers about MoE")
	state, err := s.CreateRun(task)
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if state.RunID != "run_001" {
		t.Fatalf("run_id: %s", state.RunID)
	}
	if state.Status != StatusCreated {
		t.Fatalf("status: %s", state.Status)
	}
	// 重复创建应冲突
	_, err = s.CreateRun(task)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected conflict, got %v", err)
	}
	// 获取
	got, err := s.GetRun("run_001")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.RunID != "run_001" {
		t.Fatalf("got run_id: %s", got.RunID)
	}
	// 未找到
	_, err = s.GetRun("nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestTransitionRunStateMachine(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_002", "lit", "objective")
	s.CreateRun(task)
	// created -> running (合法)
	_, err := s.TransitionRun("run_002", StatusRunning, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("created->running: %v", err)
	}
	// running -> waiting (合法)
	_, err = s.TransitionRun("run_002", StatusWaiting, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("running->waiting: %v", err)
	}
	// waiting -> running (合法)
	_, err = s.TransitionRun("run_002", StatusRunning, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("waiting->running: %v", err)
	}
	// running -> completed (合法)
	_, err = s.TransitionRun("run_002", StatusCompleted, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("running->completed: %v", err)
	}
	// completed -> running (非法)
	_, err = s.TransitionRun("run_002", StatusRunning, nil, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "INVALID_STATUS_TRANSITION") {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestTransitionRunVersionGuard(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_003", "lit", "obj")
	s.CreateRun(task)
	wrongVer := 99
	_, err := s.TransitionRun("run_003", StatusRunning, &wrongVer, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "CONFLICT") {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestPutArtifactAndRead(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_004", "lit", "obj")
	s.CreateRun(task)
	content := []byte("test artifact content")
	ai, _ := NewArtifactInput("art_1", "report", content)
	art, err := s.PutArtifact("run_004", ai)
	if err != nil {
		t.Fatalf("PutArtifact: %v", err)
	}
	if art.SizeBytes != len(content) {
		t.Fatalf("size: %d", art.SizeBytes)
	}
	// 读取并校验完整性
	read, err := s.ReadArtifact("run_004", "art_1")
	if err != nil {
		t.Fatalf("ReadArtifact: %v", err)
	}
	if string(read) != string(content) {
		t.Fatalf("content mismatch")
	}
	// 工件 ID 出现在状态中
	state, _ := s.GetRun("run_004")
	if len(state.ArtifactIDs) != 1 || state.ArtifactIDs[0] != "art_1" {
		t.Fatalf("artifact_ids: %v", state.ArtifactIDs)
	}
}

func TestPutArtifactBudgetExceeded(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_005", "lit", "obj")
	task.Budget.MaxArtifactBytes = 10 // 极小预算
	s.CreateRun(task)
	content := []byte("this is way too long for the budget")
	ai, _ := NewArtifactInput("art_1", "report", content)
	_, err := s.PutArtifact("run_005", ai)
	if err == nil || !strings.Contains(err.Error(), "BUDGET_EXCEEDED") {
		t.Fatalf("expected budget exceeded, got %v", err)
	}
}

func TestArtifactDependencyGraph(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_006", "lit", "obj")
	s.CreateRun(task)
	// art_1 无父
	c1 := []byte("content1")
	ai1, _ := NewArtifactInput("art_1", "raw", c1)
	s.PutArtifact("run_006", ai1)
	// art_2 父为 art_1
	c2 := []byte("content2")
	ai2, _ := NewArtifactInput("art_2", "derived", c2)
	ai2.ParentArtifactIDs = []string{"art_1"}
	s.PutArtifact("run_006", ai2)
	graph, err := s.ArtifactDependencyGraph("run_006")
	if err != nil {
		t.Fatalf("ArtifactDependencyGraph: %v", err)
	}
	if len(graph) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph))
	}
	if len(graph["art_2"]) != 1 || graph["art_2"][0] != "art_1" {
		t.Fatalf("art_2 parents: %v", graph["art_2"])
	}
	// 父不存在应报错
	c3 := []byte("c3")
	ai3, _ := NewArtifactInput("art_3", "d", c3)
	ai3.ParentArtifactIDs = []string{"nonexistent"}
	_, err = s.PutArtifact("run_006", ai3)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected parent not found, got %v", err)
	}
}

func TestEventsAfter(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_007", "lit", "obj")
	s.CreateRun(task)
	s.AppendEvent("run_007", "info", "first", nil)
	s.AppendEvent("run_007", "info", "second", nil)
	events, err := s.EventsAfter("run_007", 0)
	if err != nil {
		t.Fatalf("EventsAfter: %v", err)
	}
	// run_created + 2 events = 3
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	after1, _ := s.EventsAfter("run_007", 1)
	if len(after1) != 2 {
		t.Fatalf("expected 2 events after seq 1, got %d", len(after1))
	}
}

func TestRecordValidatorResult(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_008", "lit", "obj")
	s.CreateRun(task)
	vr, _ := NewValidatorResult("syntax_check", "run_008", true)
	s.RecordValidatorResult(vr)
	results, _ := s.ValidatorResults("run_008")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Fatal("should pass")
	}
}

func TestRecordError(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_009", "lit", "obj")
	s.CreateRun(task)
	er, _ := NewErrorRecord("TIMEOUT_ERROR", "timed out", true)
	s.RecordError("run_009", er)
	errs, _ := s.Errors("run_009")
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !errs[0].Recoverable {
		t.Fatal("should be recoverable")
	}
}

func TestReplayManifest(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_010", "lit", "obj")
	s.CreateRun(task)
	c := []byte("content")
	ai, _ := NewArtifactInput("art_1", "report", c)
	s.PutArtifact("run_010", ai)
	manifest, err := s.ReplayManifest("run_010")
	if err != nil {
		t.Fatalf("ReplayManifest: %v", err)
	}
	if manifest["format_version"] != "harness-replay-v1" {
		t.Fatalf("format_version: %v", manifest["format_version"])
	}
	// 写文件
	path, err := s.WriteReplayManifest("run_010")
	if err != nil {
		t.Fatalf("WriteReplayManifest: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join("run_010", "replay_manifest.json")) {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestStateVersions(t *testing.T) {
	s := newTestStore(t)
	task, _ := NewTaskContract("run_011", "lit", "obj")
	s.CreateRun(task)
	s.TransitionRun("run_011", StatusRunning, nil, nil, nil, nil, nil)
	s.TransitionRun("run_011", StatusCompleted, nil, nil, nil, nil, nil)
	versions, err := s.StateVersions("run_011")
	if err != nil {
		t.Fatalf("StateVersions: %v", err)
	}
	// created(1) + running(2) + completed(3) = 3 versions
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
	if versions[0].StateVersion != 1 || versions[2].StateVersion != 3 {
		t.Fatalf("version order: %d, %d", versions[0].StateVersion, versions[2].StateVersion)
	}
}

func TestListRuns(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 3; i++ {
		task, _ := NewTaskContract("run_l"+string(rune('A'+i)), "lit", "obj")
		s.CreateRun(task)
	}
	runs, err := s.ListRuns(10)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
}

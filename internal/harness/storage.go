package harness

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// allowedTransitions 运行状态转换表。
var allowedTransitions = map[RunStatus]map[RunStatus]bool{
	StatusCreated:   {StatusRunning: true, StatusFailed: true, StatusCancelled: true},
	StatusRunning:   {StatusWaiting: true, StatusCompleted: true, StatusFailed: true, StatusCancelled: true},
	StatusWaiting:   {StatusRunning: true, StatusFailed: true, StatusCancelled: true},
	StatusCompleted: {},
	StatusFailed:    {},
	StatusCancelled: {},
}

// HarnessStore 单进程持久化边界；状态快照只追加，工件按 SHA-256 内容寻址。
type HarnessStore struct {
	dataDir  string
	dbPath   string
	blobsDir string
	runsDir  string
	db       *sql.DB
}

// NewHarnessStore 构造存储实例。
func NewHarnessStore(dataDir string) *HarnessStore {
	return &HarnessStore{
		dataDir:  dataDir,
		dbPath:   filepath.Join(dataDir, "harness.sqlite3"),
		blobsDir: filepath.Join(dataDir, "blobs", "sha256"),
		runsDir:  filepath.Join(dataDir, "runs"),
	}
}

// Initialize 建立目录与数据库 schema。
func (s *HarnessStore) Initialize() error {
	if err := os.MkdirAll(s.blobsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.runsDir, 0o755); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return err
	}
	s.db = db
	schema := `
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
		CREATE TABLE IF NOT EXISTS runs (
			run_id TEXT PRIMARY KEY,
			task_json TEXT NOT NULL,
			state_json TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS state_versions (
			run_id TEXT NOT NULL,
			state_version INTEGER NOT NULL,
			state_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (run_id, state_version),
			FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS events (
			run_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			event_json TEXT NOT NULL,
			PRIMARY KEY (run_id, sequence),
			FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS artifacts (
			run_id TEXT NOT NULL,
			artifact_id TEXT NOT NULL,
			artifact_json TEXT NOT NULL,
			PRIMARY KEY (run_id, artifact_id),
			FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS validator_results (
			run_id TEXT NOT NULL,
			validator_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			result_json TEXT NOT NULL,
			PRIMARY KEY (run_id, validator_id, created_at),
			FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS errors (
			run_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			error_json TEXT NOT NULL,
			PRIMARY KEY (run_id, sequence),
			FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_events_run_sequence ON events(run_id, sequence);
		CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts(run_id);
	`
	_, err = db.Exec(schema)
	return err
}

// Close 关闭数据库连接。
func (s *HarnessStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ensureInitialized 懒初始化。
func (s *HarnessStore) ensureInitialized() error {
	if s.db == nil {
		return s.Initialize()
	}
	return nil
}

// CreateRun 持久化新接受的任务；任务 ID 即运行 ID。
func (s *HarnessStore) CreateRun(task *TaskContract) (*RunState, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, err
	}
	state, err := NewRunState(task)
	if err != nil {
		return nil, err
	}
	taskJSON, _ := json.Marshal(task)
	stateJSON, _ := json.Marshal(state)
	now := state.CreatedAt.Format(time.RFC3339Nano)
	_, err = s.db.Exec(
		`INSERT INTO runs(run_id, task_json, state_json, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		state.RunID, string(taskJSON), string(stateJSON), string(state.Status), now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, NewHarnessError("CONFLICT", fmt.Sprintf("run already exists: %s", task.TaskID), err)
		}
		return nil, err
	}
	_, err = s.db.Exec(
		`INSERT INTO state_versions(run_id, state_version, state_json, created_at) VALUES (?, ?, ?, ?)`,
		state.RunID, state.StateVersion, string(stateJSON), now,
	)
	if err != nil {
		return nil, err
	}
	s.AppendEvent(state.RunID, "run_created", "Task contract accepted", map[string]any{"skill_id": task.SkillID})
	return state, nil
}

// GetRun 获取运行状态。
func (s *HarnessStore) GetRun(runID string) (*RunState, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, err
	}
	var stateJSON string
	err := s.db.QueryRow(`SELECT state_json FROM runs WHERE run_id = ?`, runID).Scan(&stateJSON)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("run", runID)
	}
	if err != nil {
		return nil, err
	}
	var state RunState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// ListRuns 列出运行（按创建时间降序）。
func (s *HarnessStore) ListRuns(limit int) ([]*RunState, error) {
	if limit < 1 || limit > 10000 {
		return nil, errors.New("limit must be between 1 and 10000")
	}
	if err := s.ensureInitialized(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT state_json FROM runs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*RunState
	for rows.Next() {
		var stateJSON string
		if err := rows.Scan(&stateJSON); err != nil {
			return nil, err
		}
		var state RunState
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return nil, err
		}
		result = append(result, &state)
	}
	return result, rows.Err()
}

// StateVersions 返回不可变快照（按版本升序）。
func (s *HarnessStore) StateVersions(runID string) ([]*RunState, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT state_json FROM state_versions WHERE run_id = ? ORDER BY state_version`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*RunState
	for rows.Next() {
		var stateJSON string
		if err := rows.Scan(&stateJSON); err != nil {
			return nil, err
		}
		var state RunState
		if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
			return nil, err
		}
		result = append(result, &state)
	}
	return result, rows.Err()
}

// TransitionRun 在状态与版本守卫下创建下一状态快照。
func (s *HarnessStore) TransitionRun(
	runID string,
	status RunStatus,
	expectedStateVersion *int,
	activeActionID *string,
	artifactIDs []string,
	usage map[string]float64,
	lastError *ErrorRecord,
) (*RunState, error) {
	current, err := s.GetRun(runID)
	if err != nil {
		return nil, err
	}
	if expectedStateVersion != nil && current.StateVersion != *expectedStateVersion {
		return nil, NewHarnessError("CONFLICT",
			fmt.Sprintf("expected state version %d, found %d for %s", *expectedStateVersion, current.StateVersion, runID), nil)
	}
	if status != current.Status && !allowedTransitions[current.Status][status] {
		return nil, NewHarnessError("INVALID_STATUS_TRANSITION",
			fmt.Sprintf("cannot transition run %s from %s to %s", runID, current.Status, status), nil)
	}
	next := *current
	next.Status = status
	next.StateVersion = current.StateVersion + 1
	if activeActionID != nil {
		next.ActiveActionID = *activeActionID
	} else {
		next.ActiveActionID = ""
	}
	if artifactIDs != nil {
		next.ArtifactIDs = artifactIDs
	}
	if usage != nil {
		next.Usage = usage
	}
	if lastError != nil {
		next.LastError = lastError
	}
	next.UpdatedAt = utcNow()
	if err := s.persistNextState(&next, current.StateVersion); err != nil {
		return nil, err
	}
	s.AppendEvent(runID, "state_changed", fmt.Sprintf("Run transitioned to %s", status),
		map[string]any{"state_version": next.StateVersion})
	return &next, nil
}

// AppendEvent 追加事件到数据库与 JSONL 文件。
func (s *HarnessStore) AppendEvent(runID, kind, message string, payload map[string]any) (*Event, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	var maxSeq int
	err = tx.QueryRow(`SELECT COALESCE(MAX(sequence), 0) + 1 FROM events WHERE run_id = ?`, runID).Scan(&maxSeq)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	event, err := NewEvent(maxSeq, runID, kind, message)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if payload != nil {
		event.Payload = payload
	}
	eventJSON, _ := json.Marshal(event)
	_, err = tx.Exec(`INSERT INTO events(run_id, sequence, event_json) VALUES (?, ?, ?)`,
		runID, event.Sequence, string(eventJSON))
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	// 追加到 JSONL 文件
	eventPath := filepath.Join(s.runsDir, runID, "events.jsonl")
	os.MkdirAll(filepath.Dir(eventPath), 0o755)
	f, err := os.OpenFile(eventPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return event, nil // 数据库已写入，文件失败不阻塞
	}
	defer f.Close()
	f.WriteString(string(eventJSON) + "\n")
	return event, nil
}

// EventsAfter 返回指定序号之后的事件。
func (s *HarnessStore) EventsAfter(runID string, afterSequence int) ([]*Event, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT event_json FROM events WHERE run_id = ? AND sequence > ? ORDER BY sequence`,
		runID, afterSequence,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Event
	for rows.Next() {
		var eventJSON string
		if err := rows.Scan(&eventJSON); err != nil {
			return nil, err
		}
		var event Event
		if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
			return nil, err
		}
		result = append(result, &event)
	}
	return result, rows.Err()
}

// PutArtifact 按 SHA-256 写入字节并链接不可变元数据到运行。
func (s *HarnessStore) PutArtifact(runID string, in *ArtifactInput) (*Artifact, error) {
	state, err := s.GetRun(runID)
	if err != nil {
		return nil, err
	}
	usedBytes, err := s.artifactBytes(runID)
	if err != nil {
		return nil, err
	}
	if usedBytes+len(in.Content) > state.Task.Budget.MaxArtifactBytes {
		return nil, ErrBudgetExceeded("artifact would exceed the run artifact-byte budget")
	}
	if err := s.assertParentArtifacts(runID, in.ParentArtifactIDs); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(in.Content)
	digest := hex.EncodeToString(sum[:])
	blobPath := s.blobPath(digest)
	if err := s.writeBlobIfMissing(blobPath, in.Content); err != nil {
		return nil, err
	}
	relURI, _ := filepath.Rel(s.dataDir, blobPath)
	relURI = filepath.ToSlash(relURI)
	artifact, err := ArtifactFromInput(runID, in, relURI)
	if err != nil {
		return nil, err
	}
	artJSON, _ := json.Marshal(artifact)
	_, err = s.db.Exec(
		`INSERT INTO artifacts(run_id, artifact_id, artifact_json) VALUES (?, ?, ?)`,
		runID, artifact.ArtifactID, string(artJSON),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, NewHarnessError("CONFLICT",
				fmt.Sprintf("artifact already exists in run %s: %s", runID, artifact.ArtifactID), err)
		}
		return nil, err
	}
	// 更新状态：追加 artifact_id 并递增版本
	current, err := s.GetRun(runID)
	if err != nil {
		return nil, err
	}
	next := *current
	next.StateVersion = current.StateVersion + 1
	next.ArtifactIDs = append(next.ArtifactIDs, artifact.ArtifactID)
	next.UpdatedAt = utcNow()
	if err := s.persistNextState(&next, current.StateVersion); err != nil {
		return nil, err
	}
	s.AppendEvent(runID, "artifact_stored", fmt.Sprintf("Stored artifact %s", artifact.ArtifactID),
		map[string]any{"artifact_id": artifact.ArtifactID, "kind": artifact.Kind, "sha256": artifact.SHA256})
	return artifact, nil
}

// GetArtifact 获取工件元数据。
func (s *HarnessStore) GetArtifact(runID, artifactID string) (*Artifact, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	var artJSON string
	err := s.db.QueryRow(
		`SELECT artifact_json FROM artifacts WHERE run_id = ? AND artifact_id = ?`,
		runID, artifactID,
	).Scan(&artJSON)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("artifact", artifactID)
	}
	if err != nil {
		return nil, err
	}
	var art Artifact
	if err := json.Unmarshal([]byte(artJSON), &art); err != nil {
		return nil, err
	}
	return &art, nil
}

// ListArtifacts 列出运行的工件（按 ID 排序）。
func (s *HarnessStore) ListArtifacts(runID string) ([]*Artifact, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT artifact_json FROM artifacts WHERE run_id = ? ORDER BY artifact_id`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Artifact
	for rows.Next() {
		var artJSON string
		if err := rows.Scan(&artJSON); err != nil {
			return nil, err
		}
		var art Artifact
		if err := json.Unmarshal([]byte(artJSON), &art); err != nil {
			return nil, err
		}
		result = append(result, &art)
	}
	return result, rows.Err()
}

// ArtifactDependencyGraph 返回每个工件及其父工件 ID。
func (s *HarnessStore) ArtifactDependencyGraph(runID string) (map[string][]string, error) {
	arts, err := s.ListArtifacts(runID)
	if err != nil {
		return nil, err
	}
	graph := make(map[string][]string, len(arts))
	for _, a := range arts {
		graph[a.ArtifactID] = a.ParentArtifactIDs
	}
	return graph, nil
}

// ReadArtifact 读取工件字节并校验完整性。
func (s *HarnessStore) ReadArtifact(runID, artifactID string) ([]byte, error) {
	art, err := s.GetArtifact(runID, artifactID)
	if err != nil {
		return nil, err
	}
	path := s.safeDataPath(art.StorageURI)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, NewHarnessError("ARTIFACT_MISSING",
			fmt.Sprintf("artifact bytes are missing: %s", artifactID), err)
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	if digest != art.SHA256 || len(content) != art.SizeBytes {
		return nil, NewHarnessError("ARTIFACT_INTEGRITY",
			fmt.Sprintf("artifact integrity check failed: %s", artifactID), nil)
	}
	return content, nil
}

// RecordValidatorResult 记录验证器结果。
func (s *HarnessStore) RecordValidatorResult(result *ValidatorResult) (*ValidatorResult, error) {
	if err := s.requireRun(result.RunID); err != nil {
		return nil, err
	}
	if err := s.assertParentArtifacts(result.RunID, result.ArtifactIDs); err != nil {
		return nil, err
	}
	resultJSON, _ := json.Marshal(result)
	_, err := s.db.Exec(
		`INSERT INTO validator_results(run_id, validator_id, created_at, result_json) VALUES (?, ?, ?, ?)`,
		result.RunID, result.ValidatorID, result.CreatedAt.Format(time.RFC3339Nano), string(resultJSON),
	)
	if err != nil {
		return nil, err
	}
	s.AppendEvent(result.RunID, "validator_completed",
		fmt.Sprintf("Validator %s completed", result.ValidatorID),
		map[string]any{"passed": result.Passed, "confidence": result.Confidence})
	return result, nil
}

// ValidatorResults 列出运行的验证器结果。
func (s *HarnessStore) ValidatorResults(runID string) ([]*ValidatorResult, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT result_json FROM validator_results WHERE run_id = ? ORDER BY created_at, validator_id`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ValidatorResult
	for rows.Next() {
		var rJSON string
		if err := rows.Scan(&rJSON); err != nil {
			return nil, err
		}
		var vr ValidatorResult
		if err := json.Unmarshal([]byte(rJSON), &vr); err != nil {
			return nil, err
		}
		result = append(result, &vr)
	}
	return result, rows.Err()
}

// RecordError 记录错误。
func (s *HarnessStore) RecordError(runID string, er *ErrorRecord) (*ErrorRecord, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	var maxSeq int
	err = tx.QueryRow(`SELECT COALESCE(MAX(sequence), 0) + 1 FROM errors WHERE run_id = ?`, runID).Scan(&maxSeq)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	erJSON, _ := json.Marshal(er)
	_, err = tx.Exec(`INSERT INTO errors(run_id, sequence, error_json) VALUES (?, ?, ?)`,
		runID, maxSeq, string(erJSON))
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.AppendEvent(runID, "error_recorded", er.Message,
		map[string]any{"code": er.Code, "recoverable": er.Recoverable})
	return er, nil
}

// Errors 列出运行的错误记录。
func (s *HarnessStore) Errors(runID string) ([]*ErrorRecord, error) {
	if err := s.requireRun(runID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT error_json FROM errors WHERE run_id = ? ORDER BY sequence`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ErrorRecord
	for rows.Next() {
		var eJSON string
		if err := rows.Scan(&eJSON); err != nil {
			return nil, err
		}
		var er ErrorRecord
		if err := json.Unmarshal([]byte(eJSON), &er); err != nil {
			return nil, err
		}
		result = append(result, &er)
	}
	return result, rows.Err()
}

// ReplayManifest 返回离线轨迹回放所需的全部可观察输入和结果。
func (s *HarnessStore) ReplayManifest(runID string) (map[string]any, error) {
	state, err := s.GetRun(runID)
	if err != nil {
		return nil, err
	}
	versions, err := s.StateVersions(runID)
	if err != nil {
		return nil, err
	}
	events, err := s.EventsAfter(runID, 0)
	if err != nil {
		return nil, err
	}
	arts, err := s.ListArtifacts(runID)
	if err != nil {
		return nil, err
	}
	vrs, err := s.ValidatorResults(runID)
	if err != nil {
		return nil, err
	}
	errs, err := s.Errors(runID)
	if err != nil {
		return nil, err
	}
	versJSON := make([]any, len(versions))
	for i, v := range versions {
		versJSON[i] = v
	}
	evJSON := make([]any, len(events))
	for i, e := range events {
		evJSON[i] = e
	}
	artJSON := make([]any, len(arts))
	for i, a := range arts {
		artJSON[i] = a
	}
	vrJSON := make([]any, len(vrs))
	for i, v := range vrs {
		vrJSON[i] = v
	}
	erJSON := make([]any, len(errs))
	for i, e := range errs {
		erJSON[i] = e
	}
	return map[string]any{
		"format_version":     "harness-replay-v1",
		"task":               state.Task,
		"states":             versJSON,
		"events":             evJSON,
		"artifacts":          artJSON,
		"validator_results":  vrJSON,
		"errors":             erJSON,
	}, nil
}

// WriteReplayManifest 将回放清单写入 JSON 文件。
func (s *HarnessStore) WriteReplayManifest(runID string) (string, error) {
	manifest, err := s.ReplayManifest(runID)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(s.runsDir, runID, "replay_manifest.json")
	os.MkdirAll(filepath.Dir(dest), 0o755)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	return dest, os.WriteFile(dest, data, 0o644)
}

// ---- 内部辅助 ----

func (s *HarnessStore) persistNextState(state *RunState, expectedVersion int) error {
	stateJSON, _ := json.Marshal(state)
	now := state.UpdatedAt.Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		`UPDATE runs SET state_json = ?, status = ?, updated_at = ?
		 WHERE run_id = ? AND json_extract(state_json, '$.state_version') = ?`,
		string(stateJSON), string(state.Status), now, state.RunID, expectedVersion,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows != 1 {
		return NewHarnessError("CONFLICT",
			fmt.Sprintf("state changed concurrently for run: %s", state.RunID), nil)
	}
	_, err = s.db.Exec(
		`INSERT INTO state_versions(run_id, state_version, state_json, created_at) VALUES (?, ?, ?, ?)`,
		state.RunID, state.StateVersion, string(stateJSON), now,
	)
	return err
}

func (s *HarnessStore) artifactBytes(runID string) (int, error) {
	arts, err := s.ListArtifacts(runID)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, a := range arts {
		total += a.SizeBytes
	}
	return total, nil
}

func (s *HarnessStore) assertParentArtifacts(runID string, ids []string) error {
	for _, id := range ids {
		var exists int
		err := s.db.QueryRow(
			`SELECT 1 FROM artifacts WHERE run_id = ? AND artifact_id = ?`, runID, id,
		).Scan(&exists)
		if err == sql.ErrNoRows {
			return ErrNotFound("parent artifact", id)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *HarnessStore) requireRun(runID string) error {
	_, err := s.GetRun(runID)
	return err
}

func (s *HarnessStore) writeBlobIfMissing(path string, content []byte) error {
	os.MkdirAll(filepath.Dir(path), 0o755)
	if _, err := os.Stat(path); err == nil {
		existing, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		exSum := sha256.Sum256(existing)
		newSum := sha256.Sum256(content)
		if hex.EncodeToString(exSum[:]) != hex.EncodeToString(newSum[:]) {
			return NewHarnessError("ARTIFACT_INTEGRITY",
				fmt.Sprintf("content-addressed blob collision at %s", filepath.Base(path)), nil)
		}
		return nil
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			os.Remove(tmp)
			return s.writeBlobIfMissing(path, content)
		}
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close() // Windows 上 rename 前必须关闭
	return os.Rename(tmp, path)
}

func (s *HarnessStore) blobPath(digest string) string {
	return filepath.Join(s.blobsDir, digest[:2], digest[2:4], digest)
}

func (s *HarnessStore) safeDataPath(relativeURI string) string {
	candidate := filepath.Join(s.dataDir, filepath.FromSlash(relativeURI))
	abs, _ := filepath.Abs(candidate)
	dataAbs, _ := filepath.Abs(s.dataDir)
	if !strings.HasPrefix(abs, dataAbs) {
		panic(NewHarnessError("PATH_DENIED", "artifact URI escapes the store data directory", nil))
	}
	return abs
}

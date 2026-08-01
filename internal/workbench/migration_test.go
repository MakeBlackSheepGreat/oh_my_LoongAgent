package workbench

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// newLegacyStore 构造旧结构（无 username/password_hash 列）的 accounts 表并插入旧账户。
func newLegacyStore(t *testing.T) *WorkbenchStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewWorkbenchStore(db)
	// 旧 schema：accounts 无 username/password_hash；其余表缺失（由 InitAll 补建）。
	_, err = db.Exec(`
		CREATE TABLE accounts (
			account_id   TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			status       TEXT NOT NULL DEFAULT 'active',
			locale       TEXT NOT NULL DEFAULT 'zh-CN',
			created_at   TEXT NOT NULL
		);
		INSERT INTO accounts(account_id, display_name, status, locale, created_at) VALUES
			('AAAAAAAAAAAAAAAAAAAAAAAAAA', 'alice', 'active', 'zh-CN', '2026-01-01T00:00:00Z'),
			('BBBBBBBBBBBBBBBBBBBBBBBBBB', 'alice', 'active', 'zh-CN', '2026-01-02T00:00:00Z'),
			('CCCCCCCCCCCCCCCCCCCCCCCCCC', '测试用户', 'active', 'en', '2026-01-03T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	return store
}

func TestMigrateAccountColumns_AddsColumnsAndBackfills(t *testing.T) {
	store := newLegacyStore(t)
	ctx := context.Background()
	if err := store.InitAll(ctx); err != nil {
		t.Fatalf("InitAll on legacy schema: %v", err)
	}

	// 列已补齐
	cols, err := store.tableColumns(ctx, "accounts")
	if err != nil {
		t.Fatalf("tableColumns: %v", err)
	}
	seen := map[string]bool{}
	for _, c := range cols {
		seen[c] = true
	}
	if !seen["username"] || !seen["password_hash"] {
		t.Fatalf("expected username/password_hash columns, got %v", cols)
	}

	// 回填结果：display_name → username；冲突去重加 -2；中文映射为下划线；password_hash 空
	rows, err := store.db.QueryContext(ctx,
		`SELECT username, display_name, password_hash FROM accounts ORDER BY created_at`)
	if err != nil {
		t.Fatalf("query accounts: %v", err)
	}
	defer rows.Close()
	type row struct {
		username, displayName, hash string
	}
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.username, &r.displayName, &r.hash); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	want := []row{
		{username: "alice", displayName: "alice", hash: ""},
		{username: "alice-2", displayName: "alice", hash: ""},
		{username: "____", displayName: "测试用户", hash: ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}

	// 唯一索引存在：插入重复 username 应失败
	_, err = store.db.ExecContext(ctx,
		`INSERT INTO accounts(account_id, username, display_name, password_hash, status, locale, created_at)
		 VALUES ('DDDDDDDDDDDDDDDDDDDDDDDD', 'alice', 'dup', 'x', 'active', 'zh-CN', '2026-01-04T00:00:00Z')`)
	if err == nil {
		t.Fatal("expected unique index to reject duplicate username")
	}
}

func TestMigrateAccountColumns_Idempotent(t *testing.T) {
	store := newLegacyStore(t)
	ctx := context.Background()
	if err := store.InitAll(ctx); err != nil {
		t.Fatalf("first InitAll: %v", err)
	}
	// 第二次调用幂等：不报错、不回滚已回填的 username
	if err := store.InitAll(ctx); err != nil {
		t.Fatalf("second InitAll must be idempotent: %v", err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM accounts WHERE username = ''`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no accounts with empty username after migration, got %d", count)
	}
}

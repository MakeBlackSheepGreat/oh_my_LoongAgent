package workbench

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestStore 构造测试用内存 SQLite WorkbenchStore；风格对齐 providers/audit_test.go。
func newTestStore(t *testing.T) *WorkbenchStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewWorkbenchStore(db)
	if err := store.InitAll(context.Background()); err != nil {
		t.Fatalf("InitAll: %v", err)
	}
	return store
}

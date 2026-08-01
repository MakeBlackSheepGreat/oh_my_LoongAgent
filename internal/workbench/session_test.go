package workbench

import (
	"context"
	"testing"
	"time"
)

// TestCreateSession_Success 验证会话创建。
func TestCreateSession_Success(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	sess, err := store.CreateSession(context.Background(), acc.AccountID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.SessionID == "" {
		t.Fatal("session_id empty")
	}
	if sess.AccountID != acc.AccountID {
		t.Fatalf("account_id: got %q", sess.AccountID)
	}
	if !sess.ExpiresAt.After(sess.CreatedAt) {
		t.Fatal("expires_at should be after created_at")
	}
}

// TestCreateSession_EmptyAccountID 验证空 account_id 被拒绝。
func TestCreateSession_EmptyAccountID(t *testing.T) {
	store := newTestStore(t)
	_, err := store.CreateSession(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty account_id")
	}
}

// TestGetSession_NotFound 验证不存在返回 nil。
func TestGetSession_NotFound(t *testing.T) {
	store := newTestStore(t)
	sess, err := store.GetSession(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess != nil {
		t.Fatalf("expected nil, got %+v", sess)
	}
}

// TestGetSession_Found 验证查询已存在会话。
func TestGetSession_Found(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	created, _ := store.CreateSession(context.Background(), acc.AccountID)
	got, err := store.GetSession(context.Background(), created.SessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got == nil || got.SessionID != created.SessionID {
		t.Fatalf("expected %s, got %+v", created.SessionID, got)
	}
}

// TestGetSession_Expired 验证过期会话返回 nil。
func TestGetSession_Expired(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	// 直接插入一个已过期的会话
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO sessions(session_id, account_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		"expired_session_id", acc.AccountID,
		time.Now().Add(-2*time.Hour).Format(time.RFC3339Nano),
		time.Now().Add(-1*time.Hour).Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}
	sess, err := store.GetSession(context.Background(), "expired_session_id")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess != nil {
		t.Fatal("expected nil for expired session")
	}
}

// TestDeleteSession 验证删除会话。
func TestDeleteSession(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	sess, _ := store.CreateSession(context.Background(), acc.AccountID)
	if err := store.DeleteSession(context.Background(), sess.SessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	got, _ := store.GetSession(context.Background(), sess.SessionID)
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

package workbench

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"slim-agent/internal/harness"
)

// 会话过期时间：7 天。
const sessionTTL = 7 * 24 * time.Hour

// Session 会话契约；SessionID 为 crypto/rand 生成的 32 字节 hex 编码。
type Session struct {
	SessionID string    `json:"session_id"`
	AccountID string    `json:"account_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InitSessionsTable 创建 sessions 表（幂等）。
func (s *WorkbenchStore) InitSessionsTable(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS sessions (
			session_id  TEXT PRIMARY KEY,
			account_id  TEXT NOT NULL,
			created_at  TEXT NOT NULL,
			expires_at  TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_account ON sessions(account_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
	`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return harness.NewHarnessError(
			harness.ErrCodeInternal,
			"failed to initialize sessions table",
			err,
		)
	}
	return nil
}

// newSessionID 生成 32 字节随机数的 hex 编码（64 字符）；crypto/rand 保证密码学安全。
func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession 为指定账户创建会话；生成 crypto/rand session_id，7 天过期。
func (s *WorkbenchStore) CreateSession(ctx context.Context, accountID string) (*Session, error) {
	if accountID == "" {
		return nil, harness.ErrValidation("account_id must be non-empty", nil)
	}
	sessionID, err := newSessionID()
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to generate session id", err)
	}
	now := time.Now().UTC()
	sess := &Session{
		SessionID: sessionID,
		AccountID: accountID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions(session_id, account_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		sess.SessionID, sess.AccountID, sess.CreatedAt.Format(time.RFC3339Nano), sess.ExpiresAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to create session", err)
	}
	return sess, nil
}

// GetSession 按 session_id 查询会话；过期或不存在返回 nil。
func (s *WorkbenchStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	var (
		sess      Session
		createdAt string
		expiresAt string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT session_id, account_id, created_at, expires_at FROM sessions WHERE session_id = ?`,
		sessionID,
	).Scan(&sess.SessionID, &sess.AccountID, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get session", err)
	}
	sess.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	sess.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, nil
	}
	return &sess, nil
}

// DeleteSession 删除会话；不存在不报错。
func (s *WorkbenchStore) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE session_id = ?`, sessionID)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to delete session", err)
	}
	return nil
}

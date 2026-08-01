package workbench

import (
	"context"
	"database/sql"
	"time"

	"slim-agent/internal/harness"
)

// Conversation 会话契约；绑定账户与可选项目。
type Conversation struct {
	ConversationID string    `json:"conversation_id"`
	AccountID      string    `json:"account_id"`
	ProjectID      string    `json:"project_id,omitempty"`
	Title          string    `json:"title"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateConversation 在当前账户下创建会话。
func (a *AccountScoped) CreateConversation(ctx context.Context, conversationID, projectID, title string) (*Conversation, error) {
	if conversationID == "" || title == "" {
		return nil, harness.ErrValidation("conversation_id and title must be non-empty", nil)
	}
	now := time.Now().UTC()
	c := &Conversation{
		ConversationID: conversationID,
		AccountID:      a.accountID,
		ProjectID:      projectID,
		Title:          title,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_, err := a.store.db.ExecContext(ctx,
		`INSERT INTO conversations(conversation_id, account_id, project_id, title, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ConversationID, c.AccountID, c.ProjectID, c.Title,
		c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to create conversation", err)
	}
	return c, nil
}

// GetConversation 按 ID 查询会话；跨账户返回 nil, nil（对应 404）。
func (a *AccountScoped) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	var (
		c         Conversation
		createdAt string
		updatedAt string
	)
	err := a.store.db.QueryRowContext(ctx,
		`SELECT conversation_id, account_id, project_id, title, created_at, updated_at
		 FROM conversations WHERE conversation_id = ? AND account_id = ?`,
		conversationID, a.accountID,
	).Scan(&c.ConversationID, &c.AccountID, &c.ProjectID, &c.Title, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get conversation", err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return &c, nil
}

// ListConversations 列出当前账户的会话，按 updated_at 降序。
func (a *AccountScoped) ListConversations(ctx context.Context) ([]*Conversation, error) {
	rows, err := a.store.db.QueryContext(ctx,
		`SELECT conversation_id, account_id, project_id, title, created_at, updated_at
		 FROM conversations WHERE account_id = ? ORDER BY updated_at DESC`,
		a.accountID,
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list conversations", err)
	}
	defer rows.Close()
	var result []*Conversation
	for rows.Next() {
		var (
			c         Conversation
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&c.ConversationID, &c.AccountID, &c.ProjectID, &c.Title, &createdAt, &updatedAt); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan conversation", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		result = append(result, &c)
	}
	if result == nil {
		result = []*Conversation{}
	}
	return result, rows.Err()
}

// UpdateConversationTitle 更新会话标题。
func (a *AccountScoped) UpdateConversationTitle(ctx context.Context, conversationID, title string) error {
	if title == "" {
		return harness.ErrValidation("title must be non-empty", nil)
	}
	res, err := a.store.db.ExecContext(ctx,
		`UPDATE conversations SET title = ?, updated_at = ? WHERE conversation_id = ? AND account_id = ?`,
		title, time.Now().UTC().Format(time.RFC3339Nano), conversationID, a.accountID,
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to update conversation", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("conversation", conversationID)
	}
	return nil
}

// DeleteConversation 删除会话及其消息（事务内先删消息再删会话）。
func (a *AccountScoped) DeleteConversation(ctx context.Context, conversationID string) error {
	tx, err := a.store.db.BeginTx(ctx, nil)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to begin transaction", err)
	}
	defer tx.Rollback()

	// 删除会话下的消息
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM messages WHERE conversation_id = ? AND account_id = ?`,
		conversationID, a.accountID,
	); err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to delete messages", err)
	}

	// 删除会话
	res, err := tx.ExecContext(ctx,
		`DELETE FROM conversations WHERE conversation_id = ? AND account_id = ?`,
		conversationID, a.accountID,
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to delete conversation", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("conversation", conversationID)
	}
	return tx.Commit()
}

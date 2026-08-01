package workbench

import (
	"context"
	"regexp"
	"time"

	"slim-agent/internal/harness"
)

// reRole 消息角色白名单（与 providers.ChatMessage 对齐）。
var reRole = regexp.MustCompile(`^(system|user|assistant|tool)$`)

// Message 会话消息契约。
type Message struct {
	MessageID      string    `json:"message_id"`
	AccountID      string    `json:"account_id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// AppendMessage 追加一条消息到会话；消息 ID 由调用方生成（客户端或服务端 ULID）。
func (a *AccountScoped) AppendMessage(ctx context.Context, messageID, conversationID, role, content string) (*Message, error) {
	if messageID == "" || conversationID == "" {
		return nil, harness.ErrValidation("message_id and conversation_id must be non-empty", nil)
	}
	if !reRole.MatchString(role) {
		return nil, harness.ErrValidation("invalid role, must be system|user|assistant|tool", nil)
	}
	if content == "" || len(content) > 100000 {
		return nil, harness.ErrValidation("content length out of range [1,100000]", nil)
	}
	// 校验会话属于当前账户
	c, err := a.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, harness.ErrNotFound("conversation", conversationID)
	}
	m := &Message{
		MessageID:      messageID,
		AccountID:      a.accountID,
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		CreatedAt:      time.Now().UTC(),
	}
	_, err = a.store.db.ExecContext(ctx,
		`INSERT INTO messages(message_id, account_id, conversation_id, role, content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		m.MessageID, m.AccountID, m.ConversationID, m.Role, m.Content, m.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to append message", err)
	}
	return m, nil
}

// ListMessages 按 created_at 升序列出会话消息，分页 limit/offset。
func (a *AccountScoped) ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := a.store.db.QueryContext(ctx,
		`SELECT message_id, account_id, conversation_id, role, content, created_at
		 FROM messages WHERE conversation_id = ? AND account_id = ?
		 ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		conversationID, a.accountID, limit, offset,
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list messages", err)
	}
	defer rows.Close()
	var result []*Message
	for rows.Next() {
		var (
			m         Message
			createdAt string
		)
		if err := rows.Scan(&m.MessageID, &m.AccountID, &m.ConversationID, &m.Role, &m.Content, &createdAt); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan message", err)
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		result = append(result, &m)
	}
	if result == nil {
		result = []*Message{}
	}
	return result, rows.Err()
}

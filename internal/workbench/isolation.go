package workbench

import (
	"context"
	"database/sql"
	"time"

	"slim-agent/internal/harness"
)

// Project 项目契约。
type Project struct {
	ProjectID   string    `json:"project_id"`
	AccountID   string    `json:"account_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AccountScoped 按账户隔离的查询包装器；自动附加 WHERE account_id = ?。
// 跨账户访问返回 nil, nil（对应 HTTP 404，不泄露资源存在性）。
type AccountScoped struct {
	store     *WorkbenchStore
	accountID string
}

// NewAccountScoped 构造 AccountScoped；accountID 为当前会话账户。
func NewAccountScoped(store *WorkbenchStore, accountID string) *AccountScoped {
	return &AccountScoped{store: store, accountID: accountID}
}

// AccountID 返回当前 scope 绑定的账户 ID。
func (a *AccountScoped) AccountID() string { return a.accountID }

// CreateProject 在当前账户下创建项目。
func (a *AccountScoped) CreateProject(ctx context.Context, projectID, name, description string) (*Project, error) {
	now := time.Now().UTC()
	p := &Project{
		ProjectID:   projectID,
		AccountID:   a.accountID,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := a.store.db.ExecContext(ctx,
		`INSERT INTO projects(project_id, account_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		p.ProjectID, p.AccountID, p.Name, p.Description, p.CreatedAt.Format(time.RFC3339Nano), p.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to create project", err)
	}
	return p, nil
}

// GetProject 按 ID 查询项目；account_id 不匹配返回 nil, nil（对应 HTTP 404）。
func (a *AccountScoped) GetProject(ctx context.Context, projectID string) (*Project, error) {
	var (
		p         Project
		createdAt string
		updatedAt string
	)
	err := a.store.db.QueryRowContext(ctx,
		`SELECT project_id, account_id, name, description, created_at, updated_at
		 FROM projects WHERE project_id = ? AND account_id = ?`,
		projectID, a.accountID,
	).Scan(&p.ProjectID, &p.AccountID, &p.Name, &p.Description, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get project", err)
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return &p, nil
}

// UpdateProject 更新项目名称与描述。
func (a *AccountScoped) UpdateProject(ctx context.Context, projectID, name, description string) error {
	if name == "" {
		return harness.ErrValidation("name must be non-empty", nil)
	}
	res, err := a.store.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, description = ?, updated_at = ?
		 WHERE project_id = ? AND account_id = ?`,
		name, description, time.Now().UTC().Format(time.RFC3339Nano), projectID, a.accountID,
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to update project", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("project", projectID)
	}
	return nil
}

// DeleteProject 删除项目（会话与消息由应用层级联处理，此处仅删项目）。
func (a *AccountScoped) DeleteProject(ctx context.Context, projectID string) error {
	res, err := a.store.db.ExecContext(ctx,
		`DELETE FROM projects WHERE project_id = ? AND account_id = ?`,
		projectID, a.accountID,
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to delete project", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("project", projectID)
	}
	return nil
}

// ListProjects 列出当前账户的项目，按 created_at 降序。
func (a *AccountScoped) ListProjects(ctx context.Context) ([]*Project, error) {
	rows, err := a.store.db.QueryContext(ctx,
		`SELECT project_id, account_id, name, description, created_at, updated_at
		 FROM projects WHERE account_id = ? ORDER BY created_at DESC`,
		a.accountID,
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list projects", err)
	}
	defer rows.Close()
	var result []*Project
	for rows.Next() {
		var (
			p         Project
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(&p.ProjectID, &p.AccountID, &p.Name, &p.Description, &createdAt, &updatedAt); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan project", err)
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		result = append(result, &p)
	}
	if result == nil {
		result = []*Project{}
	}
	return result, rows.Err()
}

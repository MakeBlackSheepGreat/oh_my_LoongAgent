// Package workbench 实现账户、会话、数据隔离与计量等应用层能力。
// 契约风格对齐 internal/harness/contracts.go：校验在 Validate 方法中执行；
// 所有 DB 操作接受 context.Context，共享状态用 sync.Mutex 保护。
package workbench

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"slim-agent/internal/harness"
)

// ---- 预编译正则 ----

var (
	// reULID 匹配 26 字符 Crockford Base32 编码的 ULID。
	reULID     = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	reLocale   = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,4})?$`)
	// reUsername 登录名规则：3-32 位字母/数字/下划线/连字符。
	reUsername = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)
)

// ---- 枚举 ----

// AccountStatus 账户状态。
type AccountStatus string

const (
	AccountActive   AccountStatus = "active"
	AccountDisabled AccountStatus = "disabled"
)

// 默认 locale 常量。
const defaultLocale = "zh-CN"

// ---- 结构体 ----

// Account 用户账户契约；AccountID 为 ULID，按时间排序利于索引。
// Username 为登录名；PasswordHash 仅用于存储层校验，绝不随 JSON 返回（json:"-"）。
type Account struct {
	AccountID    string        `json:"account_id"`
	Username     string        `json:"username"`
	DisplayName  string        `json:"display_name"`
	Status       AccountStatus `json:"status"`
	Locale       string        `json:"locale"`
	CreatedAt    time.Time     `json:"created_at"`
	PasswordHash string        `json:"-"`
}

// Validate 校验账户字段；AccountID 需为合法 ULID，DisplayName 与 Locale 非空。
func (a *Account) Validate() error {
	if !reULID.MatchString(a.AccountID) {
		return fmt.Errorf("account_id invalid: %s", a.AccountID)
	}
	if a.DisplayName == "" || len(a.DisplayName) > 64 {
		return errors.New("display_name length out of range [1,64]")
	}
	if a.Status != AccountActive && a.Status != AccountDisabled {
		return fmt.Errorf("status invalid: %s", a.Status)
	}
	if !reLocale.MatchString(a.Locale) {
		return fmt.Errorf("locale invalid: %s", a.Locale)
	}
	return nil
}

// ---- Account CRUD（WorkbenchStore 方法） ----

// CreateAccount 生成 ULID account_id 并写入 accounts 表；时间复杂度 O(1)。
// passwordHash 为 PBKDF2 哈希（调用方生成）；username 需满足 reUsername 且全局唯一。
func (s *WorkbenchStore) CreateAccount(ctx context.Context, username, displayName, locale, passwordHash string) (*Account, error) {
	if displayName == "" {
		return nil, harness.ErrValidation("display_name must be non-empty", nil)
	}
	if !reUsername.MatchString(username) {
		return nil, harness.ErrValidation("username must be 3-32 characters of [A-Za-z0-9_-]", nil)
	}
	if passwordHash == "" {
		return nil, harness.ErrValidation("password is required", nil)
	}
	if locale == "" {
		locale = defaultLocale
	}
	acc := &Account{
		AccountID:    newULID(),
		Username:     username,
		DisplayName:  displayName,
		Status:       AccountActive,
		Locale:       locale,
		CreatedAt:    time.Now().UTC(),
		PasswordHash: passwordHash,
	}
	if err := acc.Validate(); err != nil {
		return nil, harness.ErrValidation("account validation failed", err)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO accounts(account_id, username, display_name, password_hash, status, locale, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		acc.AccountID, acc.Username, acc.DisplayName, acc.PasswordHash, string(acc.Status), acc.Locale, acc.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, harness.NewHarnessError(harness.ErrCodeConflict, "username already exists", nil)
		}
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to create account", err)
	}
	return acc, nil
}

// GetAccount 按 account_id 查询账户（含密码哈希，仅内部使用）；不存在返回 nil, nil（对应 HTTP 404）。
func (s *WorkbenchStore) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	var (
		acc       Account
		status    string
		createdAt string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT account_id, username, display_name, password_hash, status, locale, created_at FROM accounts WHERE account_id = ?`,
		accountID,
	).Scan(&acc.AccountID, &acc.Username, &acc.DisplayName, &acc.PasswordHash, &status, &acc.Locale, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get account", err)
	}
	acc.Status = AccountStatus(status)
	acc.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &acc, nil
}

// GetAccountByUsername 按登录名查询账户（含密码哈希，仅内部使用）；不存在返回 nil, nil。
func (s *WorkbenchStore) GetAccountByUsername(ctx context.Context, username string) (*Account, error) {
	var (
		acc       Account
		status    string
		createdAt string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT account_id, username, display_name, password_hash, status, locale, created_at FROM accounts WHERE username = ?`,
		username,
	).Scan(&acc.AccountID, &acc.Username, &acc.DisplayName, &acc.PasswordHash, &status, &acc.Locale, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get account by username", err)
	}
	acc.Status = AccountStatus(status)
	acc.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &acc, nil
}

// ListAccounts 列出全部账户，按 created_at 升序；时间复杂度 O(n)。
func (s *WorkbenchStore) ListAccounts(ctx context.Context) ([]*Account, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT account_id, username, display_name, status, locale, created_at FROM accounts ORDER BY created_at`,
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list accounts", err)
	}
	defer rows.Close()
	var result []*Account
	for rows.Next() {
		var (
			acc       Account
			status    string
			createdAt string
		)
		if err := rows.Scan(&acc.AccountID, &acc.Username, &acc.DisplayName, &status, &acc.Locale, &createdAt); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan account", err)
		}
		acc.Status = AccountStatus(status)
		acc.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		result = append(result, &acc)
	}
	if result == nil {
		result = []*Account{}
	}
	return result, rows.Err()
}

// EnsureDefaultAccount 检查账户是否存在；若已存在则返回第一个账户，否则返回 nil,nil。
// 不自动创建默认账户——改由前端引导用户注册首个账户。
func (s *WorkbenchStore) EnsureDefaultAccount(ctx context.Context) (*Account, error) {
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	if len(accounts) > 0 {
		return accounts[0], nil
	}
	return nil, nil
}

// UpdateAccountLocale 更新账户 locale 偏好；locale 由调用方校验（{zh-CN,en}）。
// AccountID 来自鉴权 context（AccountScoped，只能改当前登录账户）。
func (s *WorkbenchStore) UpdateAccountLocale(ctx context.Context, accountID, locale string) (*Account, error) {
	acc, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, harness.ErrNotFound("account", accountID)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE accounts SET locale = ? WHERE account_id = ?`, locale, accountID)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to update account locale", err)
	}
	acc.Locale = locale
	return acc, nil
}

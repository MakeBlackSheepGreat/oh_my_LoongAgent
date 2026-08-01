package workbench

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"slim-agent/internal/harness"
)

// ---- 预编译正则 ----

var (
	reProfileID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	reBaseURL   = regexp.MustCompile(`^https?://[^\s]+$`)
)

// ---- 枚举 ----

// ProfileScope 供应商档案作用域。
type ProfileScope string

const (
	ScopeAccount ProfileScope = "account"
	ScopeSystem  ProfileScope = "system"
)

// ---- 结构体 ----

// ProviderProfile 供应商档案契约；密钥从环境变量读取，不落库（APIKeyEnv 仅存变量名）。
type ProviderProfile struct {
	ProfileID   string       `json:"profile_id"`
	AccountID   string       `json:"account_id"`
	ProviderID  string       `json:"provider_id"`
	DisplayName string       `json:"display_name"`
	BaseURL     string       `json:"base_url"`
	ModelID     string       `json:"model_id"`
	APIKeyEnv   string       `json:"api_key_env"`
	Scope       ProfileScope `json:"scope"`
	IsActive    bool         `json:"is_active"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Validate 校验档案字段。
func (p *ProviderProfile) Validate() error {
	if !reProfileID.MatchString(p.ProfileID) {
		return fmt.Errorf("profile_id invalid: %s", p.ProfileID)
	}
	if p.AccountID == "" {
		return errors.New("account_id must be non-empty")
	}
	if p.ProviderID == "" {
		return errors.New("provider_id must be non-empty")
	}
	if p.DisplayName == "" || len(p.DisplayName) > 256 {
		return errors.New("display_name length out of range [1,256]")
	}
	if !reBaseURL.MatchString(p.BaseURL) {
		return fmt.Errorf("base_url invalid: %s", p.BaseURL)
	}
	if p.ModelID == "" {
		return errors.New("model_id must be non-empty")
	}
	if p.APIKeyEnv == "" {
		return errors.New("api_key_env must be non-empty")
	}
	if p.Scope != ScopeAccount && p.Scope != ScopeSystem {
		return fmt.Errorf("scope invalid: %s", p.Scope)
	}
	return nil
}

// ---- ProviderProfile CRUD（AccountScoped 方法） ----

// CreateProfile 在当前账户下创建供应商档案；默认 scope=account。
func (a *AccountScoped) CreateProfile(ctx context.Context, profileID, providerID, displayName, baseURL, modelID, apiKeyEnv string) (*ProviderProfile, error) {
	p := &ProviderProfile{
		ProfileID:   profileID,
		AccountID:   a.accountID,
		ProviderID:  providerID,
		DisplayName: displayName,
		BaseURL:     baseURL,
		ModelID:     modelID,
		APIKeyEnv:   apiKeyEnv,
		Scope:       ScopeAccount,
		IsActive:    false,
		CreatedAt:   time.Now().UTC(),
	}
	if err := p.Validate(); err != nil {
		return nil, harness.ErrValidation("profile validation failed", err)
	}
	_, err := a.store.db.ExecContext(ctx,
		`INSERT INTO provider_profiles(profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ProfileID, p.AccountID, p.ProviderID, p.DisplayName, p.BaseURL, p.ModelID, p.APIKeyEnv,
		string(p.Scope), boolToInt(p.IsActive), p.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to create profile", err)
	}
	return p, nil
}

// GetProfile 按 ID 查询档案；account scope 档案跨账户返回 nil, nil。
// system scope 档案所有账户可见。
func (a *AccountScoped) GetProfile(ctx context.Context, profileID string) (*ProviderProfile, error) {
	var (
		p         ProviderProfile
		scope     string
		isActive  int
		createdAt string
	)
	err := a.store.db.QueryRowContext(ctx,
		`SELECT profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at
		 FROM provider_profiles
		 WHERE profile_id = ? AND (account_id = ? OR scope = ?)`,
		profileID, a.accountID, string(ScopeSystem),
	).Scan(&p.ProfileID, &p.AccountID, &p.ProviderID, &p.DisplayName, &p.BaseURL, &p.ModelID, &p.APIKeyEnv,
		&scope, &isActive, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to get profile", err)
	}
	p.Scope = ProfileScope(scope)
	p.IsActive = isActive != 0
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &p, nil
}

// ListProfiles 列出当前账户可见的档案：account scope（属主）+ system scope（公共）。
func (a *AccountScoped) ListProfiles(ctx context.Context) ([]*ProviderProfile, error) {
	rows, err := a.store.db.QueryContext(ctx,
		`SELECT profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at
		 FROM provider_profiles
		 WHERE account_id = ? OR scope = ?
		 ORDER BY created_at DESC`,
		a.accountID, string(ScopeSystem),
	)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to list profiles", err)
	}
	defer rows.Close()
	var result []*ProviderProfile
	for rows.Next() {
		var (
			p         ProviderProfile
			scope     string
			isActive  int
			createdAt string
		)
		if err := rows.Scan(&p.ProfileID, &p.AccountID, &p.ProviderID, &p.DisplayName, &p.BaseURL,
			&p.ModelID, &p.APIKeyEnv, &scope, &isActive, &createdAt); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan profile", err)
		}
		p.Scope = ProfileScope(scope)
		p.IsActive = isActive != 0
		p.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		result = append(result, &p)
	}
	if result == nil {
		result = []*ProviderProfile{}
	}
	return result, rows.Err()
}

// ActivateProfile 激活档案；确保同账户同 provider 只有一个激活档案（事务内重置其他）。
func (a *AccountScoped) ActivateProfile(ctx context.Context, profileID string) error {
	tx, err := a.store.db.BeginTx(ctx, nil)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to begin transaction", err)
	}
	defer tx.Rollback()

	// 查询目标档案，确认可见性
	var (
		accountID string
		providerID string
		scope     string
	)
	err = tx.QueryRowContext(ctx,
		`SELECT account_id, provider_id, scope FROM provider_profiles
		 WHERE profile_id = ? AND (account_id = ? OR scope = ?)`,
		profileID, a.accountID, string(ScopeSystem),
	).Scan(&accountID, &providerID, &scope)
	if err == sql.ErrNoRows {
		return harness.ErrNotFound("provider_profile", profileID)
	}
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to query profile", err)
	}

	// 重置同账户同 provider 的其他激活档案
	_, err = tx.ExecContext(ctx,
		`UPDATE provider_profiles SET is_active = 0 WHERE account_id = ? AND provider_id = ?`,
		a.accountID, providerID,
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to reset active profiles", err)
	}

	// 激活目标档案
	_, err = tx.ExecContext(ctx,
		`UPDATE provider_profiles SET is_active = 1 WHERE profile_id = ?`,
		profileID,
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to activate profile", err)
	}
	return tx.Commit()
}

// DeleteProfile 删除档案；仅 account scope 且属主可删；system scope 不可删（返回错误）。
func (a *AccountScoped) DeleteProfile(ctx context.Context, profileID string) error {
	res, err := a.store.db.ExecContext(ctx,
		`DELETE FROM provider_profiles WHERE profile_id = ? AND account_id = ? AND scope = ?`,
		profileID, a.accountID, string(ScopeAccount),
	)
	if err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to delete profile", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return harness.ErrNotFound("provider_profile", profileID)
	}
	return nil
}

// boolToInt 将 bool 转为 SQLite INTEGER（0/1）。
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

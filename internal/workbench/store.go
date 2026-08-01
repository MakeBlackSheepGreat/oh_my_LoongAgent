package workbench

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"slim-agent/internal/harness"

	_ "modernc.org/sqlite"
)

// WorkbenchStore 应用层存储边界；复用 *sql.DB，不持有所有权。
// 所有方法接受 context.Context；表创建幂等（IF NOT EXISTS）。
type WorkbenchStore struct {
	db *sql.DB
}

// NewWorkbenchStore 构造 WorkbenchStore；调用方负责传入已打开的 *sql.DB 并在结束时关闭。
func NewWorkbenchStore(db *sql.DB) *WorkbenchStore {
	return &WorkbenchStore{db: db}
}

// DB 暴露底层 *sql.DB，供需要直接访问的组件（如 MeterRecorder）复用。
func (s *WorkbenchStore) DB() *sql.DB { return s.db }

// InitAccountsTable 创建 accounts 表（幂等）。
// account_id 为 ULID 主键；username 为登录名（唯一，可为空串兼容旧数据）；
// password_hash 为 PBKDF2 哈希（空串表示未设置密码，不可登录）；created_at 索引支持按时间排序查询。
func (s *WorkbenchStore) InitAccountsTable(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS accounts (
			account_id     TEXT PRIMARY KEY,
			username       TEXT NOT NULL DEFAULT '',
			display_name   TEXT NOT NULL,
			password_hash  TEXT NOT NULL DEFAULT '',
			status         TEXT NOT NULL DEFAULT 'active',
			locale         TEXT NOT NULL DEFAULT 'zh-CN',
			created_at     TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_accounts_created_at
			ON accounts(created_at);
	`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return harness.NewHarnessError(
			harness.ErrCodeInternal,
			"failed to initialize accounts table",
			err,
		)
	}
	// 旧库迁移必须先补列，再建 username 唯一索引（旧表无该列时索引创建会失败）。
	if err := s.migrateAccountColumns(ctx); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username) WHERE username <> ''`); err != nil {
		return harness.NewHarnessError(harness.ErrCodeInternal, "failed to create accounts.username index", err)
	}
	return nil
}

// migrateAccountColumns 旧库迁移：accounts 表缺少 username/password_hash 列时补充。
// SQLite 不支持 ADD COLUMN IF NOT EXISTS，需先查 PRAGMA table_info。
// 旧账户 username 回填为 display_name（冲突时追加 -2/-3… 后缀），password_hash 留空（未设密码，不可登录）。
func (s *WorkbenchStore) migrateAccountColumns(ctx context.Context) error {
	cols, err := s.tableColumns(ctx, "accounts")
	if err != nil {
		return err
	}
	hasUsername := false
	hasPasswordHash := false
	for _, c := range cols {
		switch c {
		case "username":
			hasUsername = true
		case "password_hash":
			hasPasswordHash = true
		}
	}
	if !hasUsername {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE accounts ADD COLUMN username TEXT NOT NULL DEFAULT ''`); err != nil {
			return harness.NewHarnessError(harness.ErrCodeInternal, "failed to add accounts.username column", err)
		}
	}
	if !hasPasswordHash {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE accounts ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`); err != nil {
			return harness.NewHarnessError(harness.ErrCodeInternal, "failed to add accounts.password_hash column", err)
		}
	}
	if !hasUsername {
		// 回填 username：display_name 去空格截断；冲突时追加 -2/-3…
		rows, err := s.db.QueryContext(ctx,
			`SELECT account_id, display_name FROM accounts WHERE username = '' ORDER BY created_at`)
		if err != nil {
			return harness.NewHarnessError(harness.ErrCodeInternal, "failed to list legacy accounts", err)
		}
		defer rows.Close()
		type legacy struct {
			id, name string
		}
		var legacyList []legacy
		for rows.Next() {
			var l legacy
			if err := rows.Scan(&l.id, &l.name); err != nil {
				return harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan legacy account", err)
			}
			legacyList = append(legacyList, l)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, l := range legacyList {
			base := strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
					return r
				}
				return '_'
			}, strings.TrimSpace(l.name))
			if len(base) > 29 {
				// 预留 -2/-3…/ -10 后缀空间，保证回填 username 满足 3-32 位规则。
				base = base[:29]
			}
			if base == "" {
				base = "user"
			}
			if len(base) < 3 {
				// 过短（1-2 字符）不满足 3-32 位规则，回退到 "user"（唯一冲突由下方去重循环处理）。
				base = "user"
			}
			username := base
			for i := 2; ; i++ {
				var exists string
				err := s.db.QueryRowContext(ctx,
					`SELECT account_id FROM accounts WHERE username = ? AND account_id <> ?`,
					username, l.id).Scan(&exists)
				if err == sql.ErrNoRows {
					break
				}
				if err != nil {
					return harness.NewHarnessError(harness.ErrCodeInternal, "failed to check username uniqueness", err)
				}
				username = fmt.Sprintf("%s-%d", base, i)
			}
			if _, err := s.db.ExecContext(ctx,
				`UPDATE accounts SET username = ? WHERE account_id = ?`, username, l.id); err != nil {
				return harness.NewHarnessError(harness.ErrCodeInternal, "failed to backfill username", err)
			}
		}
	}
	return nil
}

// tableColumns 返回表的列名列表。
func (s *WorkbenchStore) tableColumns(ctx context.Context, table string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to read table info", err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var (
			cid       int
			name, typ string
			notnull   int
			dflt      any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, harness.NewHarnessError(harness.ErrCodeInternal, "failed to scan table info", err)
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

// InitWorkbenchSchema 创建全部领域表（幂等）。
// 所有领域表含 account_id 外键并建立索引；provider_profiles 含 scope 字段。
func (s *WorkbenchStore) InitWorkbenchSchema(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS projects (
			project_id   TEXT PRIMARY KEY,
			account_id   TEXT NOT NULL,
			name         TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_projects_account ON projects(account_id);

		CREATE TABLE IF NOT EXISTS conversations (
			conversation_id TEXT PRIMARY KEY,
			account_id      TEXT NOT NULL,
			project_id      TEXT NOT NULL DEFAULT '',
			title           TEXT NOT NULL DEFAULT '',
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_conversations_account ON conversations(account_id);

		CREATE TABLE IF NOT EXISTS provider_profiles (
			profile_id   TEXT PRIMARY KEY,
			account_id   TEXT NOT NULL,
			provider_id  TEXT NOT NULL,
			display_name TEXT NOT NULL,
			base_url     TEXT NOT NULL,
			model_id     TEXT NOT NULL,
			api_key_env  TEXT NOT NULL,
			scope        TEXT NOT NULL DEFAULT 'account',
			is_active    INTEGER NOT NULL DEFAULT 0,
			created_at   TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_provider_profiles_account ON provider_profiles(account_id);
		CREATE INDEX IF NOT EXISTS idx_provider_profiles_scope ON provider_profiles(scope);

		CREATE TABLE IF NOT EXISTS task_drafts (
			draft_id      TEXT PRIMARY KEY,
			account_id    TEXT NOT NULL,
			conversation_id TEXT NOT NULL DEFAULT '',
			objective     TEXT NOT NULL,
			skill_id      TEXT NOT NULL DEFAULT '',
			status        TEXT NOT NULL DEFAULT 'draft',
			run_id        TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_task_drafts_account ON task_drafts(account_id);

		CREATE TABLE IF NOT EXISTS messages (
			message_id      TEXT PRIMARY KEY,
			account_id      TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			role            TEXT NOT NULL,
			content         TEXT NOT NULL,
			created_at      TEXT NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_messages_account ON messages(account_id);
		CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
	`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return harness.NewHarnessError(
			harness.ErrCodeInternal,
			"failed to initialize workbench schema",
			err,
		)
	}
	return nil
}

// InitAll 初始化 accounts 表 + 领域表 + sessions 表 + usage 表；供服务启动时一次性调用。
func (s *WorkbenchStore) InitAll(ctx context.Context) error {
	if err := s.InitAccountsTable(ctx); err != nil {
		return fmt.Errorf("init accounts: %w", err)
	}
	if err := s.InitSessionsTable(ctx); err != nil {
		return fmt.Errorf("init sessions: %w", err)
	}
	if err := s.InitWorkbenchSchema(ctx); err != nil {
		return fmt.Errorf("init workbench schema: %w", err)
	}
	if err := s.InitUsageTable(ctx); err != nil {
		return fmt.Errorf("init usage table: %w", err)
	}
	return nil
}

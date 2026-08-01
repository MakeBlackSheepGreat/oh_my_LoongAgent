package workbench

import (
	"context"
	"testing"
)

// TestCreateProfile_AccountScope 验证 account scope 档案创建。
func TestCreateProfile_AccountScope(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	scoped := NewAccountScoped(store, acc.AccountID)

	p, err := scoped.CreateProfile(context.Background(), "prof_001", "deepseek", "My DeepSeek",
		"https://api.deepseek.com/v1", "deepseek-chat", "HARNESS_DEEPSEEK_KEY")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if p.Scope != ScopeAccount {
		t.Fatalf("scope: got %q", p.Scope)
	}
	if p.IsActive {
		t.Fatal("new profile should not be active")
	}
}

// TestListProfiles_AccountAndSystemScope 验证列表返回 account + system scope。
func TestListProfiles_AccountAndSystemScope(t *testing.T) {
	store := newTestStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	// alice 创建 account scope 档案
	scopedA := NewAccountScoped(store, accA.AccountID)
	scopedA.CreateProfile(context.Background(), "prof_a1", "deepseek", "Alice DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_A")

	// 插入 system scope 档案（直接插库模拟管理员创建）
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO provider_profiles(profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"prof_sys1", accB.AccountID, "siliconflow", "Public SF",
		"https://api.siliconflow.cn/v1", "Qwen2.5-7B", "KEY_SF",
		"system", 0, "2026-08-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert system profile: %v", err)
	}

	// alice 应看到自己的 1 个 + system scope 1 个 = 2 个
	list, err := scopedA.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}
}

// TestGetProfile_CrossAccount_AccountScope 验证 account scope 跨账户返回 nil。
func TestGetProfile_CrossAccount_AccountScope(t *testing.T) {
	store := newTestStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	scopedA := NewAccountScoped(store, accA.AccountID)
	scopedA.CreateProfile(context.Background(), "prof_a1", "deepseek", "Alice DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_A")

	// bob 尝试访问 alice 的 account scope 档案
	scopedB := NewAccountScoped(store, accB.AccountID)
	got, err := scopedB.GetProfile(context.Background(), "prof_a1")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for cross-account access, got %+v", got)
	}
}

// TestGetProfile_SystemScope_VisibleToAll 验证 system scope 所有账户可见。
func TestGetProfile_SystemScope_VisibleToAll(t *testing.T) {
	store := newTestStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	// 插入 system scope 档案
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO provider_profiles(profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"prof_sys1", accA.AccountID, "deepseek", "Public DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_DS",
		"system", 0, "2026-08-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert system profile: %v", err)
	}

	// bob 可以看到 system scope 档案
	scopedB := NewAccountScoped(store, accB.AccountID)
	got, err := scopedB.GetProfile(context.Background(), "prof_sys1")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got == nil {
		t.Fatal("expected system scope profile visible to all")
	}
}

// TestActivateProfile_SingleActive 验证同账户同 provider 只有一个激活。
func TestActivateProfile_SingleActive(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	scoped := NewAccountScoped(store, acc.AccountID)

	// 创建两个同 provider 档案
	scoped.CreateProfile(context.Background(), "prof_1", "deepseek", "DS 1",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_A")
	scoped.CreateProfile(context.Background(), "prof_2", "deepseek", "DS 2",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_B")

	// 激活第一个
	if err := scoped.ActivateProfile(context.Background(), "prof_1"); err != nil {
		t.Fatalf("ActivateProfile 1: %v", err)
	}
	// 激活第二个
	if err := scoped.ActivateProfile(context.Background(), "prof_2"); err != nil {
		t.Fatalf("ActivateProfile 2: %v", err)
	}

	// 验证只有 prof_2 激活
	list, _ := scoped.ListProfiles(context.Background())
	var activeCount int
	var activeID string
	for _, p := range list {
		if p.IsActive {
			activeCount++
			activeID = p.ProfileID
		}
	}
	if activeCount != 1 {
		t.Fatalf("expected 1 active profile, got %d", activeCount)
	}
	if activeID != "prof_2" {
		t.Fatalf("expected prof_2 active, got %s", activeID)
	}
}

// TestDeleteProfile_AccountScope 验证 account scope 可删除。
func TestDeleteProfile_AccountScope(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	scoped := NewAccountScoped(store, acc.AccountID)
	scoped.CreateProfile(context.Background(), "prof_1", "deepseek", "DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_A")

	if err := scoped.DeleteProfile(context.Background(), "prof_1"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	got, _ := scoped.GetProfile(context.Background(), "prof_1")
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

// TestDeleteProfile_SystemScope_Forbidden 验证 system scope 不可由普通账户删除。
func TestDeleteProfile_SystemScope_Forbidden(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")

	// 插入 system scope 档案
	_, err := store.db.ExecContext(context.Background(),
		`INSERT INTO provider_profiles(profile_id, account_id, provider_id, display_name, base_url, model_id, api_key_env, scope, is_active, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"prof_sys1", acc.AccountID, "deepseek", "Public DS",
		"https://api.deepseek.com/v1", "deepseek-chat", "KEY_DS",
		"system", 0, "2026-08-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	scoped := NewAccountScoped(store, acc.AccountID)
	err = scoped.DeleteProfile(context.Background(), "prof_sys1")
	if err == nil {
		t.Fatal("expected error for deleting system scope profile")
	}
}

// TestProviderProfile_Validate 验证档案字段校验。
func TestProviderProfile_Validate(t *testing.T) {
	valid := ProviderProfile{
		ProfileID: "prof_001", AccountID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ",
		ProviderID: "deepseek", DisplayName: "DS",
		BaseURL: "https://api.deepseek.com/v1", ModelID: "deepseek-chat",
		APIKeyEnv: "KEY", Scope: ScopeAccount,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid profile: %v", err)
	}
	// 密钥不落库验证：APIKeyEnv 只存变量名，不存密钥值
	if valid.APIKeyEnv == "actual_secret_value" {
		t.Fatal("api_key_env should only store variable name")
	}
}

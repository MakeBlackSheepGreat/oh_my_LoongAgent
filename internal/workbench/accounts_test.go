package workbench

import (
	"context"
	"regexp"
	"testing"
)

// TestAccount_Validate 校验 Account 字段。
func TestAccount_Validate(t *testing.T) {
	tests := []struct {
		name    string
		account  Account
		wantErr bool
	}{
		{"valid", Account{AccountID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ", DisplayName: "alice", Status: AccountActive, Locale: "zh-CN"}, false},
		{"invalid_ulid", Account{AccountID: "invalid", DisplayName: "alice", Status: AccountActive, Locale: "zh-CN"}, true},
		{"empty_name", Account{AccountID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ", DisplayName: "", Status: AccountActive, Locale: "zh-CN"}, true},
		{"invalid_status", Account{AccountID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ", DisplayName: "alice", Status: "banned", Locale: "zh-CN"}, true},
		{"invalid_locale", Account{AccountID: "01HXY9KP1ZQR4C8N7T3V6RWYBJ", DisplayName: "alice", Status: AccountActive, Locale: "123"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewULID 验证 ULID 格式与唯一性。
func TestNewULID(t *testing.T) {
	id1 := newULID()
	id2 := newULID()
	if !regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`).MatchString(id1) {
		t.Fatalf("id1 not valid ULID: %s", id1)
	}
	if id1 == id2 {
		t.Fatal("two ULIDs should differ")
	}
}

// TestCreateAccount_Success 验证账户创建。
func TestCreateAccount_Success(t *testing.T) {
	store := newTestStore(t)
	acc, err := store.CreateAccount(context.Background(), "alice", "alice", "en", "test-hash")
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if acc.DisplayName != "alice" {
		t.Fatalf("display_name: got %q", acc.DisplayName)
	}
	if acc.Status != AccountActive {
		t.Fatalf("status: got %q", acc.Status)
	}
	if acc.Locale != "en" {
		t.Fatalf("locale: got %q", acc.Locale)
	}
	if acc.AccountID == "" {
		t.Fatal("account_id empty")
	}
}

// TestCreateAccount_EmptyName 验证空名称被拒绝。
func TestCreateAccount_EmptyName(t *testing.T) {
	store := newTestStore(t)
	_, err := store.CreateAccount(context.Background(), "", "", "en", "test-hash")
	if err == nil {
		t.Fatal("expected error for empty display_name")
	}
}

// TestGetAccount_NotFound 验证不存在返回 nil, nil。
func TestGetAccount_NotFound(t *testing.T) {
	store := newTestStore(t)
	acc, err := store.GetAccount(context.Background(), "01HXY9KP1ZQR4C8N7T3V6RWYBJ")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acc != nil {
		t.Fatalf("expected nil, got %+v", acc)
	}
}

// TestGetAccount_Found 验证查询已存在账户。
func TestGetAccount_Found(t *testing.T) {
	store := newTestStore(t)
	created, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")
	got, err := store.GetAccount(context.Background(), created.AccountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got == nil || got.AccountID != created.AccountID {
		t.Fatalf("expected %s, got %+v", created.AccountID, got)
	}
}

// TestListAccounts_OrderByCreatedAt 验证按 created_at 升序。
func TestListAccounts_OrderByCreatedAt(t *testing.T) {
	store := newTestStore(t)
	a1, _ := store.CreateAccount(context.Background(), "first", "first", "zh-CN", "test-hash")
	a2, _ := store.CreateAccount(context.Background(), "second", "second", "zh-CN", "test-hash")
	list, err := store.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
	if list[0].AccountID != a1.AccountID {
		t.Fatalf("expected first account %s, got %s", a1.AccountID, list[0].AccountID)
	}
	if list[1].AccountID != a2.AccountID {
		t.Fatalf("expected second account %s, got %s", a2.AccountID, list[1].AccountID)
	}
}

// TestEnsureDefaultAccount_NoAccounts 验证无账户时返回 nil,nil。
func TestEnsureDefaultAccount_NoAccounts(t *testing.T) {
	store := newTestStore(t)
	acc, err := store.EnsureDefaultAccount(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultAccount: %v", err)
	}
	if acc != nil {
		t.Fatalf("expected nil, got %+v", acc)
	}
}

// TestEnsureDefaultAccount_AlreadyExists 验证已有账户时返回第一个账户，不重复创建。
func TestEnsureDefaultAccount_AlreadyExists(t *testing.T) {
	store := newTestStore(t)
	created, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	got, err := store.EnsureDefaultAccount(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultAccount: %v", err)
	}
	if got.AccountID != created.AccountID {
		t.Fatalf("expected account %s, got %s", created.AccountID, got.AccountID)
	}
}

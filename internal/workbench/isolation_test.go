package workbench

import (
	"context"
	"testing"
)

// TestCreateProject_SameAccount 验证同账户可创建并查询项目。
func TestCreateProject_SameAccount(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	scoped := NewAccountScoped(store, acc.AccountID)

	p, err := scoped.CreateProject(context.Background(), "proj_001", "my project", "desc")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ProjectID != "proj_001" {
		t.Fatalf("project_id: got %q", p.ProjectID)
	}
	if p.AccountID != acc.AccountID {
		t.Fatalf("account_id: got %q", p.AccountID)
	}
}

// TestGetProject_SameAccount 验证同账户可查询项目。
func TestGetProject_SameAccount(t *testing.T) {
	store := newTestStore(t)
	acc, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	scoped := NewAccountScoped(store, acc.AccountID)
	scoped.CreateProject(context.Background(), "proj_001", "my project", "desc")

	got, err := scoped.GetProject(context.Background(), "proj_001")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got == nil {
		t.Fatal("expected project, got nil")
	}
	if got.Name != "my project" {
		t.Fatalf("name: got %q", got.Name)
	}
}

// TestGetProject_CrossAccount_ReturnsNil 验证跨账户访问返回 nil, nil（不泄露存在性）。
func TestGetProject_CrossAccount_ReturnsNil(t *testing.T) {
	store := newTestStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	// alice 创建项目
	scopedA := NewAccountScoped(store, accA.AccountID)
	scopedA.CreateProject(context.Background(), "proj_001", "alice project", "secret")

	// bob 尝试访问 alice 的项目
	scopedB := NewAccountScoped(store, accB.AccountID)
	got, err := scopedB.GetProject(context.Background(), "proj_001")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for cross-account access, got %+v", got)
	}
}

// TestListProjects_OnlyOwnAccount 验证列表只返回当前账户项目。
func TestListProjects_OnlyOwnAccount(t *testing.T) {
	store := newTestStore(t)
	accA, _ := store.CreateAccount(context.Background(), "alice", "alice", "zh-CN", "test-hash")
	accB, _ := store.CreateAccount(context.Background(), "bob", "bob", "zh-CN", "test-hash")

	scopedA := NewAccountScoped(store, accA.AccountID)
	scopedA.CreateProject(context.Background(), "proj_a1", "alice project", "")
	scopedA.CreateProject(context.Background(), "proj_a2", "alice project 2", "")

	scopedB := NewAccountScoped(store, accB.AccountID)
	scopedB.CreateProject(context.Background(), "proj_b1", "bob project", "")

	// alice 只看到自己的 2 个项目
	listA, err := scopedA.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects A: %v", err)
	}
	if len(listA) != 2 {
		t.Fatalf("expected 2 projects for alice, got %d", len(listA))
	}

	// bob 只看到自己的 1 个项目
	listB, err := scopedB.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects B: %v", err)
	}
	if len(listB) != 1 {
		t.Fatalf("expected 1 project for bob, got %d", len(listB))
	}
	if listB[0].ProjectID != "proj_b1" {
		t.Fatalf("project_id: got %q", listB[0].ProjectID)
	}
}

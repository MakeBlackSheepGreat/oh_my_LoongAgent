package skills_test

import (
	"testing"

	"slim-agent/internal/harness"
	"slim-agent/internal/skills"
	"slim-agent/internal/skills/literature_search"
)

func TestRegistry_RegisterLookupList(t *testing.T) {
	reg := skills.NewRegistry()
	if err := reg.Register(literature_search.NewSkill()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	s, err := reg.Lookup("literature_search")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if s.Manifest().SkillID != "literature_search" {
		t.Fatalf("unexpected skill: %s", s.Manifest().SkillID)
	}
	manifests := reg.List()
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
	if manifests[0].Title == "" {
		t.Fatal("manifest title should be non-empty")
	}
}

func TestRegistry_DuplicateConflict(t *testing.T) {
	reg := skills.NewRegistry()
	if err := reg.Register(literature_search.NewSkill()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	err := reg.Register(literature_search.NewSkill())
	he, ok := err.(*harness.HarnessError)
	if !ok || he.Code != harness.ErrCodeConflict {
		t.Fatalf("expected CONFLICT, got %v", err)
	}
}

func TestRegistry_LookupUnknown(t *testing.T) {
	reg := skills.NewRegistry()
	_, err := reg.Lookup("ghost")
	he, ok := err.(*harness.HarnessError)
	if !ok || he.Code != harness.ErrCodeNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := skills.NewRegistry()
	if err := reg.Register(literature_search.NewSkill()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 50; j++ {
				_ = reg.List()
				_, _ = reg.Lookup("literature_search")
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

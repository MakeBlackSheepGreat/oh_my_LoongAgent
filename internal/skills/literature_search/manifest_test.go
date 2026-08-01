package literature_search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"slim-agent/internal/harness"
	"slim-agent/internal/harness/errs"
	"slim-agent/internal/skills"
)

func TestParseInput_Defaults(t *testing.T) {
	in, err := parseInput(map[string]any{"research_question": "  mixture of experts  "})
	if err != nil {
		t.Fatalf("parseInput: %v", err)
	}
	if in.ResearchQuestion != "mixture of experts" {
		t.Fatalf("unexpected question: %q", in.ResearchQuestion)
	}
	if len(in.Sources) != 4 {
		t.Fatalf("expected 4 default sources, got %v", in.Sources)
	}
	if in.MaxResults != defaultMaxResults {
		t.Fatalf("expected default max_results %d, got %d", defaultMaxResults, in.MaxResults)
	}
	if len(in.Exports) != 3 {
		t.Fatalf("expected 3 default exports, got %v", in.Exports)
	}
}

func TestParseInput_Validation(t *testing.T) {
	cases := []map[string]any{
		{}, // 缺 research_question
		{"research_question": "   "},
		{"research_question": "q", "sources": []any{"ghost"}},
		{"research_question": "q", "max_results": 0},
		{"research_question": "q", "max_results": 1000},
		{"research_question": "q", "export": []any{"xlsx"}},
	}
	for i, inputs := range cases {
		_, err := parseInput(inputs)
		he, ok := err.(*errs.HarnessError)
		if !ok || he.Code != errs.ErrCodeValidation {
			t.Fatalf("case %d: expected VALIDATION_ERROR, got %v", i, err)
		}
	}
}

func TestParseInput_Overrides(t *testing.T) {
	in, err := parseInput(map[string]any{
		"research_question": "q",
		"sources":           []string{"arxiv", "acl"},
		"max_results":       7,
		"export":            []string{"csv"},
		"source_base_urls":  map[string]any{"arxiv": "http://x/"},
	})
	if err != nil {
		t.Fatalf("parseInput: %v", err)
	}
	if len(in.Sources) != 2 || in.MaxResults != 7 || len(in.Exports) != 1 || in.Exports[0] != "csv" {
		t.Fatalf("unexpected overrides: %+v", in)
	}
	if in.SourceBases["arxiv"] != "http://x" {
		t.Fatalf("unexpected base url trim: %q", in.SourceBases["arxiv"])
	}
}

func TestExecute_FullFlow(t *testing.T) {
	env, store := newSkillEnv(t)
	openAlex := `{"results": [
		{"id": "https://openalex.org/W9", "display_name": "A Survey of Mixture of Experts",
		 "publication_year": 2023,
		 "authorships": [{"author": {"display_name": "Jane Doe"}}],
		 "abstract_inverted_index": {"moe": [0]},
		 "open_access": {"oa_url": ""}}
	]}`
	var fullTextServed bool
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "api/query"):
			feed := fmt.Sprintf(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2301.00001</id>
    <title>A Survey of Mixture of Experts</title>
    <summary>We survey MoE architectures.</summary>
    <published>2023-01-01T00:00:00Z</published>
    <author><name>Jane Doe</name></author>
    <link rel="related" href="http://arxiv.org/abs/2301.00001"/>
    <link rel="alternate" href="%s/pdf/2301.00001"/>
  </entry>
</feed>`, srv.URL)
			w.Header().Set("Content-Type", "application/atom+xml")
			_, _ = w.Write([]byte(feed))
		case strings.Contains(r.URL.Path, "works"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(openAlex))
		default:
			fullTextServed = true
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("full text of the MoE survey"))
		}
	}))
	defer srv.Close()

	task := newLitTask(t, "run_lit1")
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	req := &skills.Request{
		Task: task,
		Inputs: map[string]any{
			"research_question": "survey of mixture of experts",
			"sources":           []string{"arxiv", "openalex"},
			"source_base_urls":  map[string]any{"arxiv": srv.URL, "openalex": srv.URL},
			"max_results":       5,
			"export":            []string{"markdown", "json", "csv", "manifest"},
		},
	}
	skill := NewSkill()
	result, err := skill.Execute(context.Background(), req, env)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// 两条记录同 DOI/标题 → 合并为 1 个候选；归档 1 份证据 + 4 份导出
	if len(result.Artifacts) != 5 {
		for _, ev := range result.Events {
			t.Logf("event: %s %s %v", ev.Kind, ev.Message, ev.Payload)
		}
		t.Fatalf("expected 5 artifacts (1 evidence + 4 exports), got %d", len(result.Artifacts))
	}
	if !fullTextServed {
		t.Fatal("expected full text archive download")
	}
	if len(result.Validations) == 0 {
		t.Fatal("expected validation results")
	}
	// 验证器结果应全部通过（归档证据元数据完整）
	for _, vr := range result.Validations {
		if vr.ValidatorID == "evidence_integrity" && !vr.Passed {
			t.Fatalf("evidence_integrity should pass, got %v", vr.Findings)
		}
	}
	// 事件可回放
	if len(result.Events) == 0 {
		t.Fatal("expected events")
	}
	// 工件库落库
	state, _ := store.GetRun(task.TaskID)
	if len(state.ArtifactIDs) != 5 {
		t.Fatalf("expected 5 artifacts on run, got %v", state.ArtifactIDs)
	}
}

func TestExecute_InvalidInput(t *testing.T) {
	env, store := newSkillEnv(t)
	task := newLitTask(t, "run_lit2")
	if _, err := store.CreateRun(task); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	skill := NewSkill()
	_, err := skill.Execute(context.Background(), &skills.Request{Task: task, Inputs: map[string]any{}}, env)
	he, ok := err.(*harness.HarnessError)
	if !ok || he.Code != harness.ErrCodeValidation {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

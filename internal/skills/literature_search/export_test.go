package literature_search

import (
	"encoding/json"
	"strings"
	"testing"

	"slim-agent/internal/skills"
)

func TestExportFormats(t *testing.T) {
	cands := []*Candidate{
		{ID: "arxiv:A1", Title: "Paper One", Authors: []string{"Jane Doe"}, Year: 2023,
			DOI: "10.1000/one", Sources: []string{"arxiv"}, Evidence: []string{"arxiv:A1"},
			URL: "https://example.org/1", OpenAccessURL: "https://example.org/1.pdf"},
	}
	task := newLitTask(t, "run_exp1")
	e := buildExports(cands, 1)
	req := &skills.Request{Task: task, Inputs: map[string]any{"research_question": "MoE survey"}}

	// Markdown
	md, err := e.render("markdown", req)
	if err != nil {
		t.Fatalf("markdown render: %v", err)
	}
	for _, want := range []string{"# 文献搜索报告", "Paper One", "MoE survey", "Jane Doe", "arxiv"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}

	// JSON
	j, err := e.render("json", req)
	if err != nil {
		t.Fatalf("json render: %v", err)
	}
	var decoded struct {
		Total      int `json:"total"`
		Archived   int `json:"archived"`
		Candidates []jsonCandidate `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(j), &decoded); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if decoded.Total != 1 || decoded.Archived != 1 || len(decoded.Candidates) != 1 {
		t.Fatalf("unexpected json payload: %+v", decoded)
	}

	// CSV
	csv, err := e.render("csv", req)
	if err != nil {
		t.Fatalf("csv render: %v", err)
	}
	if !strings.Contains(csv, "id,title,year,doi") || !strings.Contains(csv, "Paper One") {
		t.Fatalf("csv render:\n%s", csv)
	}

	// Manifest
	man, err := e.render("manifest", req)
	if err != nil {
		t.Fatalf("manifest render: %v", err)
	}
	if !strings.Contains(man, `"task_id": "run_exp1"`) || !strings.Contains(man, `"total_candidates": 1`) {
		t.Fatalf("manifest render:\n%s", man)
	}

	// 未知格式
	if _, err := e.render("xlsx", req); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestExportArtifactID(t *testing.T) {
	cases := map[string]string{
		"markdown": "report_markdown",
		"json":     "report_json",
		"csv":      "report_csv",
		"manifest": "archive_manifest",
		"other":    "archive_manifest",
	}
	for format, want := range cases {
		if got := exportArtifactID(format); got != want {
			t.Fatalf("exportArtifactID(%s) = %s, want %s", format, got, want)
		}
	}
}

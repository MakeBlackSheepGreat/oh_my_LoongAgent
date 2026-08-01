package literature_search

import (
	"testing"
)

func TestDeduplicate_SameDOI(t *testing.T) {
	records := []Record{
		{Source: "arxiv", ID: "A1", DOI: "10.1000/one", Title: "Paper One"},
		{Source: "crossref", ID: "C1", DOI: "10.1000/one", Title: "Paper One"},
	}
	cands := Deduplicate(records)
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	c := cands[0]
	if c.DOI != "10.1000/one" {
		t.Fatalf("expected DOI 10.1000/one, got %s", c.DOI)
	}
	if len(c.Sources) != 2 {
		t.Fatalf("expected both sources retained, got %v", c.Sources)
	}
	if len(c.Evidence) != 2 {
		t.Fatalf("expected 2 evidence entries, got %v", c.Evidence)
	}
}

func TestDeduplicate_SameSourceID(t *testing.T) {
	records := []Record{
		{Source: "arxiv", ID: "arXiv:2301.00001", Title: "Paper A"},
		{Source: "arxiv", ID: "arXiv:2301.00001", Title: "Paper A (updated)"},
	}
	cands := Deduplicate(records)
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	if cands[0].Sources[0] != "arxiv" {
		t.Fatalf("unexpected source: %v", cands[0].Sources)
	}
}

func TestDeduplicate_TitleSimilarity(t *testing.T) {
	records := []Record{
		{Source: "arxiv", ID: "A1", Title: "Attention Is All You Need"},
		{Source: "openalex", ID: "W1", Title: "attention is all you need!"},
		{Source: "crossref", ID: "C1", Title: "Completely Different Topic"},
	}
	cands := Deduplicate(records)
	if len(cands) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(cands))
	}
	// 标题相似的合并为一个
	found := false
	for _, c := range cands {
		if len(c.Sources) == 2 && c.Title == "Attention Is All You Need" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected title-similar records merged: %+v", cands)
	}
}

func TestDeduplicate_DistinctTitles(t *testing.T) {
	records := []Record{
		{Source: "arxiv", ID: "A1", Title: "Transformers"},
		{Source: "openalex", ID: "W1", Title: "Mixture of Experts"},
	}
	cands := Deduplicate(records)
	if len(cands) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(cands))
	}
}

func TestTitleSimilar(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Attention Is All You Need", "attention is all you need!", true},
		{"MoE Survey", "MoE Survey", true},
		{"Transformers", "Mixture of Experts", false},
		{"", "", true},
		{"", "x", false},
	}
	for _, c := range cases {
		if got := titleSimilar(c.a, c.b); got != c.want {
			t.Fatalf("titleSimilar(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

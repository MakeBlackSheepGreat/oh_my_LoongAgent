package literature_search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"slim-agent/internal/skills/adapters"
)

func testClient() *adapters.Client { return adapters.NewClient(0, 0) }

func TestArxivSource_Search(t *testing.T) {
	feed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>http://arxiv.org/abs/2301.00001</id>
    <title>Attention Is All You Need Revisited</title>
    <summary>We revisit the transformer architecture.</summary>
    <published>2023-01-01T00:00:00Z</published>
    <author><name>Jane Doe</name></author>
    <author><name>John Smith</name></author>
    <link rel="related" href="http://arxiv.org/abs/2301.00001"/>
    <link rel="alternate" href="http://arxiv.org/pdf/2301.00001"/>
  </entry>
  <entry>
    <id>http://arxiv.org/abs/2301.00002</id>
    <title>MoE Survey</title>
    <summary>A survey of mixture of experts.</summary>
    <published>2023-02-01T00:00:00Z</published>
    <author><name>Alice Wu</name></author>
    <link rel="related" href="http://arxiv.org/abs/2301.00002"/>
  </entry>
</feed>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "api/query") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = w.Write([]byte(feed))
	}))
	defer srv.Close()

	src, err := NewSource("arxiv", srv.URL, testClient())
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	records, err := src.Search(context.Background(), "transformers", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	r0 := records[0]
	if r0.Source != "arxiv" || r0.Title != "Attention Is All You Need Revisited" {
		t.Fatalf("unexpected record: %+v", r0)
	}
	if r0.Year != 2023 {
		t.Fatalf("expected year 2023, got %d", r0.Year)
	}
	if len(r0.Authors) != 2 || r0.Authors[0] != "Jane Doe" {
		t.Fatalf("unexpected authors: %v", r0.Authors)
	}
	if r0.OpenAccessURL != "http://arxiv.org/pdf/2301.00001" {
		t.Fatalf("unexpected open access url: %s", r0.OpenAccessURL)
	}
}

func TestOpenAlexSource_Search(t *testing.T) {
	body := `{"results": [
		{"id": "https://openalex.org/W1", "display_name": "A Survey of MoE",
		 "publication_year": 2024, "doi": "https://doi.org/10.1000/xyz",
		 "authorships": [{"author": {"display_name": "Alice Wu"}}],
		 "abstract_inverted_index": {"a": [0], "survey": [1], "of": [2], "moe": [3]},
		 "open_access": {"oa_url": "https://example.org/full.txt"}}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "works") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	src, err := NewSource("openalex", srv.URL, testClient())
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	records, err := src.Search(context.Background(), "mixture of experts", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.DOI != "10.1000/xyz" {
		t.Fatalf("expected DOI 10.1000/xyz, got %s", r.DOI)
	}
	if r.Abstract != "a survey of moe" {
		t.Fatalf("expected reconstructed abstract, got %q", r.Abstract)
	}
	if r.OpenAccessURL != "https://example.org/full.txt" {
		t.Fatalf("unexpected oa url: %s", r.OpenAccessURL)
	}
}

func TestCrossrefSource_Search(t *testing.T) {
	body := `{"message": {"items": [
		{"DOI": "10.1000/abc", "title": ["Deep Learning Advances"],
		 "author": [{"family": "Doe", "given": "Jane"}],
		 "issued": {"date-parts": [[2022, 3, 1]]},
		 "URL": "https://doi.org/10.1000/abc", "abstract": "<jats:p>An abstract.</jats:p>"}
	]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "works") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	src, err := NewSource("crossref", srv.URL, testClient())
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	records, err := src.Search(context.Background(), "deep learning", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Title != "Deep Learning Advances" || r.Year != 2022 {
		t.Fatalf("unexpected record: %+v", r)
	}
	if len(r.Authors) != 1 || r.Authors[0] != "Jane Doe" {
		t.Fatalf("unexpected authors: %v", r.Authors)
	}
}

func TestACLSource_Search(t *testing.T) {
	body := `{"results": [
		{"id": "P18-1001", "title": "Neural Machine Translation", "authors": ["Bob Lee"],
		 "year": 2018, "doi": "10.18653/v1/P18-1001",
		 "abstract": "We present a neural approach.", "url": "https://aclanthology.org/P18-1001/",
		 "open_access_url": "https://aclanthology.org/P18-1001.pdf"}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "search") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	src, err := NewSource("acl", srv.URL, testClient())
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	records, err := src.Search(context.Background(), "machine translation", 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(records) != 1 || records[0].ID != "P18-1001" {
		t.Fatalf("unexpected records: %+v", records)
	}
}

func TestNewSource_Unknown(t *testing.T) {
	_, err := NewSource("ghost", "http://x", testClient())
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestSource_NetworkError(t *testing.T) {
	// 连接被拒绝的端口
	src, _ := NewSource("crossref", "http://127.0.0.1:1", testClient())
	_, err := src.Search(context.Background(), "x", 5)
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "PROVIDER_UNAVAILABLE") {
		t.Fatalf("expected PROVIDER_UNAVAILABLE mapping, got %v", err)
	}
}

package literature_search

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"slim-agent/internal/skills"
)

// exportArtifactID 导出工件的稳定 ID（reID 格式）。
func exportArtifactID(format string) string {
	switch format {
	case "markdown":
		return "report_markdown"
	case "json":
		return "report_json"
	case "csv":
		return "report_csv"
	default:
		return "archive_manifest"
	}
}

// exports 按格式渲染导出内容；数据来自去重后的候选与归档统计。
type exports struct {
	cands    []*Candidate
	archived int
}

func buildExports(cands []*Candidate, archived int) *exports {
	return &exports{cands: cands, archived: archived}
}

// render 返回指定格式的导出内容。
func (e *exports) render(format string, req *skills.Request) (string, error) {
	switch format {
	case "markdown":
		return e.renderMarkdown(req), nil
	case "json":
		return e.renderJSON(req), nil
	case "csv":
		return e.renderCSV(), nil
	case "manifest":
		return e.renderManifest(req), nil
	default:
		return "", fmt.Errorf("unknown export format: %s", format)
	}
}

// renderMarkdown 结构化证据包报告。
func (e *exports) renderMarkdown(req *skills.Request) string {
	var b strings.Builder
	b.WriteString("# 文献搜索报告\n\n")
	b.WriteString("**研究问题**：" + researchQuestion(req) + "\n\n")
	b.WriteString(fmt.Sprintf("**候选数**：%d　**已归档证据**：%d\n\n", len(e.cands), e.archived))
	for i, c := range e.cands {
		b.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, c.Title))
		if len(c.Authors) > 0 {
			b.WriteString("作者：" + strings.Join(c.Authors, ", ") + "\n\n")
		}
		b.WriteString(fmt.Sprintf("- 年份：%d\n", c.Year))
		if c.DOI != "" {
			b.WriteString("- DOI：" + c.DOI + "\n")
		}
		b.WriteString("- 来源：" + strings.Join(c.Sources, ", ") + "\n")
		if c.URL != "" {
			b.WriteString("- 链接：" + c.URL + "\n")
		}
		if len(c.Evidence) > 0 {
			b.WriteString("- 证据：" + strings.Join(c.Evidence, "; ") + "\n")
		}
		if c.Abstract != "" {
			b.WriteString("\n" + c.Abstract + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

type jsonCandidate struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Authors       []string `json:"authors"`
	Year          int      `json:"year"`
	DOI           string   `json:"doi"`
	Sources       []string `json:"sources"`
	Evidence      []string `json:"evidence"`
	URL           string   `json:"url"`
	OpenAccessURL string   `json:"open_access_url"`
}

// renderJSON 机器可读全量导出。
func (e *exports) renderJSON(req *skills.Request) string {
	out := struct {
		ResearchQuestion string          `json:"research_question"`
		Total            int             `json:"total"`
		Archived         int             `json:"archived"`
		Candidates       []jsonCandidate `json:"candidates"`
	}{
		ResearchQuestion: researchQuestion(req),
		Total:            len(e.cands),
		Archived:         e.archived,
		Candidates:       make([]jsonCandidate, 0, len(e.cands)),
	}
	for _, c := range e.cands {
		out.Candidates = append(out.Candidates, jsonCandidate{
			ID: c.ID, Title: c.Title, Authors: c.Authors, Year: c.Year, DOI: c.DOI,
			Sources: c.Sources, Evidence: c.Evidence, URL: c.URL, OpenAccessURL: c.OpenAccessURL,
		})
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data)
}

// renderCSV Excel 兼容表格（UTF-8 CSV，Excel 可直接打开）。
func (e *exports) renderCSV() string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"id", "title", "year", "doi", "authors", "sources", "evidence", "url"})
	for _, c := range e.cands {
		_ = w.Write([]string{
			c.ID, c.Title, fmt.Sprint(c.Year), c.DOI,
			strings.Join(c.Authors, "; "), strings.Join(c.Sources, "; "),
			strings.Join(c.Evidence, "; "), c.URL,
		})
	}
	w.Flush()
	return b.String()
}

// renderManifest 可回放归档清单。
func (e *exports) renderManifest(req *skills.Request) string {
	type manifestEntry struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Year     int      `json:"year"`
		DOI      string   `json:"doi"`
		Sources  []string `json:"sources"`
		Evidence []string `json:"evidence"`
	}
	out := struct {
		TaskID           string          `json:"task_id"`
		ResearchQuestion string          `json:"research_question"`
		GeneratedAt      string          `json:"generated_at"`
		TotalCandidates  int             `json:"total_candidates"`
		ArchivedEvidence int             `json:"archived_evidence"`
		Candidates       []manifestEntry `json:"candidates"`
	}{
		TaskID:           req.Task.TaskID,
		ResearchQuestion: researchQuestion(req),
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		TotalCandidates:  len(e.cands),
		ArchivedEvidence: e.archived,
		Candidates:       make([]manifestEntry, 0, len(e.cands)),
	}
	for _, c := range e.cands {
		out.Candidates = append(out.Candidates, manifestEntry{
			ID: c.ID, Title: c.Title, Year: c.Year, DOI: c.DOI,
			Sources: c.Sources, Evidence: c.Evidence,
		})
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data)
}

// researchQuestion 从原始输入提取研究问题。
func researchQuestion(req *skills.Request) string {
	if s, ok := req.Inputs["research_question"].(string); ok {
		return s
	}
	return ""
}

package literature_search

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"slim-agent/internal/harness/errs"
	"slim-agent/internal/skills/adapters"
)

// Record 单个来源返回的结构化文献记录。
type Record struct {
	Source        string
	ID            string
	DOI           string
	Title         string
	Authors       []string
	Year          int
	Abstract      string
	URL           string
	OpenAccessURL string
}

// Source 文献来源客户端；四类来源只实现"搜索→结构化记录"的最小适配。
type Source interface {
	Name() string
	Search(ctx context.Context, query string, limit int) ([]Record, error)
}

// NewSource 按名称构造来源客户端。
func NewSource(name, baseURL string, client *adapters.Client) (Source, error) {
	switch name {
	case "arxiv":
		return &arxivSource{base: baseURL, client: client}, nil
	case "openalex":
		return &openAlexSource{base: baseURL, client: client}, nil
	case "crossref":
		return &crossrefSource{base: baseURL, client: client}, nil
	case "acl":
		return &aclSource{base: baseURL, client: client}, nil
	default:
		return nil, fmt.Errorf("unknown source: %s", name)
	}
}

// ---- arXiv（Atom XML） ----

type arxivSource struct {
	base   string
	client *adapters.Client
}

func (s *arxivSource) Name() string { return "arxiv" }

type atomFeed struct {
	Entries []struct {
		ID        string `xml:"id"`
		Title     string `xml:"title"`
		Summary   string `xml:"summary"`
		Published string `xml:"published"`
		Authors   []struct {
			Name string `xml:"name"`
		} `xml:"author"`
		Links []struct {
			Rel  string `xml:"rel,attr"`
			Href string `xml:"href,attr"`
		} `xml:"link"`
	} `xml:"entry"`
}

func (s *arxivSource) Search(ctx context.Context, query string, limit int) ([]Record, error) {
	u := s.base + "/api/query?search_query=" + url.QueryEscape("all:"+query) + "&max_results=" + strconv.Itoa(limit)
	body, err := s.client.Do(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, errs.ErrValidation("arxiv: parse atom feed failed", err)
	}
	records := make([]Record, 0, len(feed.Entries))
	for _, e := range feed.Entries {
		rec := Record{
			Source:   "arxiv",
			ID:       e.ID,
			Title:    strings.TrimSpace(e.Title),
			Abstract: strings.TrimSpace(e.Summary),
			Year:     yearFromString(e.Published),
		}
		for _, a := range e.Authors {
			if a.Name != "" {
				rec.Authors = append(rec.Authors, a.Name)
			}
		}
		for _, l := range e.Links {
			switch l.Rel {
			case "related":
				rec.URL = l.Href
			case "alternate":
				if strings.Contains(l.Href, "/pdf") && rec.OpenAccessURL == "" {
					rec.OpenAccessURL = strings.TrimSuffix(l.Href, ".pdf")
				}
			}
		}
		if rec.URL == "" {
			rec.URL = "https://arxiv.org/abs/" + strings.TrimPrefix(e.ID, "http://arxiv.org/abs/")
		}
		records = append(records, rec)
	}
	return records, nil
}

// ---- OpenAlex（JSON） ----

type openAlexSource struct {
	base   string
	client *adapters.Client
}

func (s *openAlexSource) Name() string { return "openalex" }

type openAlexResponse struct {
	Results []struct {
		ID        string `json:"id"`
		Title     string `json:"display_name"`
		Year      int    `json:"publication_year"`
		DOI       string `json:"doi"`
		Authorships []struct {
			Author struct {
				DisplayName string `json:"display_name"`
			} `json:"author"`
		} `json:"authorships"`
		AbstractInv map[string][]int `json:"abstract_inverted_index"`
		OpenAccess  struct {
			OAURL string `json:"oa_url"`
		} `json:"open_access"`
	} `json:"results"`
}

func (s *openAlexSource) Search(ctx context.Context, query string, limit int) ([]Record, error) {
	u := s.base + "/works?search=" + url.QueryEscape(query) + "&per-page=" + strconv.Itoa(limit)
	var resp openAlexResponse
	if err := s.client.DoJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(resp.Results))
	for _, r := range resp.Results {
		rec := Record{
			Source:        "openalex",
			ID:            r.ID,
			Title:         strings.TrimSpace(r.Title),
			Year:          r.Year,
			DOI:           doiFromURL(r.DOI),
			URL:           r.ID,
			Abstract:      reconstructAbstract(r.AbstractInv),
			OpenAccessURL: r.OpenAccess.OAURL,
		}
		for _, a := range r.Authorships {
			if a.Author.DisplayName != "" {
				rec.Authors = append(rec.Authors, a.Author.DisplayName)
			}
		}
		records = append(records, rec)
	}
	return records, nil
}

// reconstructAbstract 由 OpenAlex inverted index 还原摘要文本。
func reconstructAbstract(inv map[string][]int) string {
	if len(inv) == 0 {
		return ""
	}
	type wordAt struct {
		pos int
		w   string
	}
	words := make([]wordAt, 0)
	for w, poss := range inv {
		for _, p := range poss {
			words = append(words, wordAt{pos: p, w: w})
		}
	}
	sort.Slice(words, func(i, j int) bool { return words[i].pos < words[j].pos })
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = w.w
	}
	return strings.Join(parts, " ")
}

// ---- Crossref（JSON） ----

type crossrefSource struct {
	base   string
	client *adapters.Client
}

func (s *crossrefSource) Name() string { return "crossref" }

type crossrefResponse struct {
	Message struct {
		Items []struct {
			DOI      string   `json:"DOI"`
			Title    []string `json:"title"`
			Abstract string   `json:"abstract"`
			URL      string   `json:"URL"`
			Author   []struct {
				Family string `json:"family"`
				Given  string `json:"given"`
			} `json:"author"`
			Issued struct {
				DateParts [][]int `json:"date-parts"`
			} `json:"issued"`
		} `json:"items"`
	} `json:"message"`
}

func (s *crossrefSource) Search(ctx context.Context, query string, limit int) ([]Record, error) {
	u := s.base + "/works?query=" + url.QueryEscape(query) + "&rows=" + strconv.Itoa(limit)
	var resp crossrefResponse
	if err := s.client.DoJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(resp.Message.Items))
	for _, item := range resp.Message.Items {
		rec := Record{
			Source:   "crossref",
			ID:       item.DOI,
			DOI:      item.DOI,
			Title:    strings.TrimSpace(firstNonEmpty(item.Title)),
			Abstract: strings.TrimSpace(item.Abstract),
			URL:      item.URL,
			Year:     yearFromDateParts(item.Issued.DateParts),
		}
		for _, a := range item.Author {
			name := strings.TrimSpace(a.Given + " " + a.Family)
			if name != "" {
				rec.Authors = append(rec.Authors, name)
			}
		}
		records = append(records, rec)
	}
	return records, nil
}

// ---- ACL（通用 JSON；ACL Anthology 暂无公开 API，协议预留） ----

type aclSource struct {
	base   string
	client *adapters.Client
}

func (s *aclSource) Name() string { return "acl" }

type aclResponse struct {
	Results []struct {
		ID            string   `json:"id"`
		Title         string   `json:"title"`
		Authors       []string `json:"authors"`
		Year          int      `json:"year"`
		DOI           string   `json:"doi"`
		Abstract      string   `json:"abstract"`
		URL           string   `json:"url"`
		OpenAccessURL string   `json:"open_access_url"`
	} `json:"results"`
}

func (s *aclSource) Search(ctx context.Context, query string, limit int) ([]Record, error) {
	u := s.base + "/search?query=" + url.QueryEscape(query) + "&limit=" + strconv.Itoa(limit)
	var resp aclResponse
	if err := s.client.DoJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(resp.Results))
	for _, r := range resp.Results {
		records = append(records, Record{
			Source:        "acl",
			ID:            r.ID,
			Title:         strings.TrimSpace(r.Title),
			Authors:       r.Authors,
			Year:          r.Year,
			DOI:           r.DOI,
			Abstract:      r.Abstract,
			URL:           r.URL,
			OpenAccessURL: r.OpenAccessURL,
		})
	}
	return records, nil
}

// ---- 小工具 ----

// yearFromString 从 RFC3339/ISO 日期字符串提取年份。
func yearFromString(s string) int {
	for i := 0; i+4 <= len(s); i++ {
		if isDigit(s[i]) && isDigit(s[i+1]) && isDigit(s[i+2]) && isDigit(s[i+3]) {
			y, err := strconv.Atoi(s[i : i+4])
			if err == nil && y >= 1900 && y <= 2100 {
				return y
			}
		}
	}
	return 0
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// yearFromDateParts 从 [[year, month, day]] 提取年份。
func yearFromDateParts(parts [][]int) int {
	if len(parts) > 0 && len(parts[0]) > 0 {
		return parts[0][0]
	}
	return 0
}

// doiFromURL 从 "https://doi.org/10.x" 提取 "10.x"。
func doiFromURL(s string) string {
	if s == "" {
		return ""
	}
	if i := strings.LastIndex(s, "/10."); i >= 0 {
		return s[i+1:]
	}
	return s
}

func firstNonEmpty(ss []string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

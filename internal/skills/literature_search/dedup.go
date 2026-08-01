package literature_search

import "strings"

// titleSimilarityThreshold 标题相似度合并阈值（归一化 Jaccard，可配置常量）。
const titleSimilarityThreshold = 0.9

// Candidate 去重合并后的文献候选；保留全部来源与证据链。
type Candidate struct {
	ID            string
	Title         string
	Authors       []string
	Year          int
	DOI           string
	Abstract      string
	URL           string
	OpenAccessURL string
	// Sources 该候选命中的全部来源。
	Sources []string
	// Evidence 证据链（如 "arxiv:arXiv:2301.00001"、"archived:ev_abc123"）。
	Evidence []string
}

// Deduplicate 分层去重：
//  1. 非空 DOI 相同 → 直接合并（优先级最高）
//  2. 同来源 ID 相同 → 直接合并
//  3. 归一化标题 Jaccard ≥ 阈值 → 合并
//
// 每个候选保留完整来源列表与证据链。
func Deduplicate(records []Record) []*Candidate {
	seen := make([]bool, len(records))
	var cands []*Candidate

	// 第一层：DOI
	byDOI := make(map[string][]int)
	for i, r := range records {
		if r.DOI != "" {
			byDOI[r.DOI] = append(byDOI[r.DOI], i)
		}
	}
	for _, idxs := range byDOI {
		for _, i := range idxs {
			seen[i] = true
		}
		cands = append(cands, mergeIndices(records, idxs))
	}

	// 第二层：同来源 ID（仅当同一 ID 出现多次才合并；单条留给标题相似层）
	byKey := make(map[string][]int)
	for i, r := range records {
		if seen[i] || r.ID == "" {
			continue
		}
		byKey[r.Source+"|"+r.ID] = append(byKey[r.Source+"|"+r.ID], i)
	}
	for _, idxs := range byKey {
		if len(idxs) < 2 {
			continue
		}
		for _, i := range idxs {
			seen[i] = true
		}
		cands = append(cands, mergeIndices(records, idxs))
	}

	// 第三层：标题相似度（贪心聚类）
	var rest []int
	for i := range records {
		if !seen[i] {
			rest = append(rest, i)
		}
	}
	for len(rest) > 0 {
		head := rest[0]
		rest = rest[1:]
		group := []int{head}
		for j := len(rest) - 1; j >= 0; j-- {
			if titleSimilar(records[head].Title, records[rest[j]].Title) {
				group = append(group, rest[j])
				rest = append(rest[:j], rest[j+1:]...)
			}
		}
		cands = append(cands, mergeIndices(records, group))
	}
	return cands
}

// mergeIndices 将一组记录合并为一个候选。
func mergeIndices(records []Record, idxs []int) *Candidate {
	c := &Candidate{}
	sources := map[string]bool{}
	for _, i := range idxs {
		r := records[i]
		if c.Title == "" {
			c.Title = r.Title
		}
		if c.Abstract == "" {
			c.Abstract = r.Abstract
		}
		if c.URL == "" {
			c.URL = r.URL
		}
		if c.OpenAccessURL == "" {
			c.OpenAccessURL = r.OpenAccessURL
		}
		if c.DOI == "" {
			c.DOI = r.DOI
		}
		if c.ID == "" {
			c.ID = r.Source + ":" + r.ID
		}
		if c.Year == 0 {
			c.Year = r.Year
		}
		c.Authors = mergeStrings(c.Authors, r.Authors)
		if !sources[r.Source] {
			sources[r.Source] = true
			c.Sources = append(c.Sources, r.Source)
		}
		c.Evidence = append(c.Evidence, r.Source+":"+r.ID)
	}
	return c
}

// mergeStrings 有序合并去重字符串列表。
func mergeStrings(dst, src []string) []string {
	seen := make(map[string]bool, len(dst)+len(src))
	for _, s := range dst {
		seen[s] = true
	}
	for _, s := range src {
		if s != "" && !seen[s] {
			seen[s] = true
			dst = append(dst, s)
		}
	}
	return dst
}

// titleSimilar 标题归一化后的 Jaccard 相似度是否达到阈值。
func titleSimilar(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	return jaccard(normalizeTitle(a), normalizeTitle(b)) >= titleSimilarityThreshold
}

// normalizeTitle 小写、去标点、压缩空白，返回词序列。
func normalizeTitle(t string) []string {
	var b strings.Builder
	b.Grow(len(t))
	for _, r := range t {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

// jaccard 两个词集合的 Jaccard 相似度。
func jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	setA := make(map[string]bool, len(a))
	for _, w := range a {
		setA[w] = true
	}
	setB := make(map[string]bool, len(b))
	for _, w := range b {
		setB[w] = true
	}
	inter := 0
	for w := range setA {
		if setB[w] {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

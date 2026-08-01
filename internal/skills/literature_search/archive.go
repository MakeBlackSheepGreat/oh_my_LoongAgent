package literature_search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"slim-agent/internal/harness"
	"slim-agent/internal/skills/adapters"
)

// maxArchiveBytes 全文归档大小上限（50 MiB，spec task7.5）。
const maxArchiveBytes = 50 << 20

// allowedArchiveMIMEs 可归档 MIME 白名单。
var allowedArchiveMIMEs = map[string]bool{
	"application/pdf": true,
	"text/plain":      true,
	"text/html":       true,
}

// ArchiveFullText 下载候选开放全文并写入工件库。
// 返回 nil 工件表示不可归档（未知 MIME/空内容），不视为错误；
// 工件 metadata 保存 MIME、大小、SHA-256、版权状态、来源 URL 与候选 ID。
func ArchiveFullText(ctx context.Context, client *adapters.Client, store *harness.HarnessStore,
	runID string, c *Candidate, now time.Time) (*harness.Artifact, error) {
	if c.OpenAccessURL == "" {
		return nil, nil
	}
	body, mime, err := client.DoWithLimit(ctx, http.MethodGet, c.OpenAccessURL, nil, maxArchiveBytes)
	if err != nil {
		return nil, err
	}
	mime = strings.ToLower(strings.TrimSpace(mime))
	if mime == "" {
		mime = sniffMIME(body)
	}
	if !allowedArchiveMIMEs[mime] || len(body) == 0 {
		return nil, nil
	}
	sum := sha256.Sum256(body)
	artifactID := fmt.Sprintf("ev_%s", hex.EncodeToString(sum[:6]))
	ai, err := harness.NewArtifactInput(artifactID, "evidence.archive", body)
	if err != nil {
		return nil, err
	}
	ai.ContentType = mime
	ai.Metadata = map[string]any{
		"source_url":       c.OpenAccessURL,
		"mime":             mime,
		"size_bytes":       len(body),
		"sha256":           hex.EncodeToString(sum[:]),
		"copyright_status": "unknown",
		"candidate_id":     c.ID,
		"archived_at":      now.Format(time.RFC3339),
	}
	return store.PutArtifact(runID, ai)
}

// sniffMIME 按内容前缀嗅探 MIME（响应头缺失时兜底）。
func sniffMIME(body []byte) string {
	head := body
	if len(head) > 512 {
		head = head[:512]
	}
	low := strings.ToLower(string(head))
	switch {
	case strings.HasPrefix(low, "%pdf"):
		return "application/pdf"
	case strings.Contains(low, "<html") || strings.HasPrefix(low, "<!doctype html"):
		return "text/html"
	case strings.Contains(low, "\n"):
		return "text/plain"
	default:
		return ""
	}
}

package literature_search

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"slim-agent/internal/harness"
)

// sortedKeys 返回 map 键的有序切片（决定来源默认顺序，稳定可复现）。
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// toInt 兼容 int/float64/json.Number 的整数提取。
func toInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case float64:
		return int(n), nil
	case string:
		return strconv.Atoi(n)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// toStringSlice 将 []any 或 []string 归一化为 []string。
func toStringSlice(v any) ([]string, error) {
	switch s := v.(type) {
	case []string:
		return s, nil
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("list item is not a string: %T", item)
			}
			out = append(out, str)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected list, got %T", v)
	}
}

// newEvent 构造结果事件（sequence 由调用方持久化时分配，此处仅保证校验通过）。
func newEvent(runID, kind, message string, payload map[string]any) *harness.Event {
	return &harness.Event{
		Sequence:  1,
		RunID:     runID,
		Kind:      kind,
		Message:   message,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}
}

// loadRunArtifacts 加载运行工件并注入内容，保留原有元数据（供验证器读取）。
func loadRunArtifacts(store *harness.HarnessStore, state *harness.RunState) (map[string]*harness.Artifact, error) {
	artifacts := make(map[string]*harness.Artifact, len(state.ArtifactIDs))
	for _, id := range state.ArtifactIDs {
		art, err := store.GetArtifact(state.RunID, id)
		if err != nil {
			return nil, err
		}
		content, err := store.ReadArtifact(state.RunID, id)
		if err != nil {
			return nil, err
		}
		md := make(map[string]any, len(art.Metadata)+1)
		for k, v := range art.Metadata {
			md[k] = v
		}
		md["_bytes"] = content
		art.Metadata = md
		artifacts[id] = art
	}
	return artifacts, nil
}

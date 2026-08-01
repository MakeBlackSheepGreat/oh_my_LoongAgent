package workbench

import (
	"encoding/json"
	"net/http"

	"slim-agent/internal/harness"
	"slim-agent/internal/providers"
)

// createProfileRequest 创建供应商档案请求体。
// 仅创建 account scope；system scope 由初始化或 CLI 注入（DeleteProfile 已保护）。
type createProfileRequest struct {
	ProfileID   string `json:"profile_id"`
	ProviderID  string `json:"provider_id"`
	DisplayName string `json:"display_name"`
	BaseURL     string `json:"base_url"`
	ModelID     string `json:"model_id"`
	APIKeyEnv   string `json:"api_key_env"`
}

// handleListPresets GET /api/providers/presets — 返回预设供应商列表。
func (s *Server) handleListPresets(w http.ResponseWriter, r *http.Request) {
	presets := providers.ListPresets()
	writeJSON(w, http.StatusOK, map[string]any{"presets": presets})
}

// handleListProfiles GET /api/providers
func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	list, err := NewAccountScoped(s.store, accountID).ListProfiles(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": list})
}

// handleCreateProfile POST /api/providers（仅 account scope）
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req createProfileRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	// profile_id 为服务端主键：客户端可不传，由服务端生成 ULID。
	if req.ProfileID == "" {
		req.ProfileID = newULID()
	}
	p, err := NewAccountScoped(s.store, accountID).CreateProfile(
		r.Context(), req.ProfileID, req.ProviderID, req.DisplayName, req.BaseURL, req.ModelID, req.APIKeyEnv)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// handleDeleteProfile DELETE /api/providers/{id}
// account scope 属主可删；system scope 不可删（返回 403 由 DeleteProfile 保护）。
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := NewAccountScoped(s.store, accountID).DeleteProfile(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleActivateProfile POST /api/providers/{id}/activate
func (s *Server) handleActivateProfile(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := NewAccountScoped(s.store, accountID).ActivateProfile(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

// handleProfileHealth GET /api/providers/{id}/health
// 复用 providers.HealthCheck（3 秒超时）；不记 token、不调 Chat。
func (s *Server) handleProfileHealth(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	p, err := NewAccountScoped(s.store, accountID).GetProfile(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if p == nil {
		writeError(w, harness.ErrNotFound("provider_profile", r.PathValue("id")))
		return
	}
	result := providers.HealthCheck(r.Context(), providers.ProviderConfig{
		ProviderID:  p.ProviderID,
		BaseURL:     p.BaseURL,
		ModelID:     p.ModelID,
		APIKeyEnv:   p.APIKeyEnv,
		DisplayName: p.DisplayName,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"profile_id":  p.ProfileID,
		"ok":          result.OK,
		"latency_ms":  result.LatencyMs,
		"error":       result.Error,
	})
}

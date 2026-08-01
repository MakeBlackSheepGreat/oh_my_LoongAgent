package workbench

import (
	"encoding/json"
	"net/http"

	"slim-agent/internal/harness"
)

// createProjectRequest 创建项目请求体。
type createProjectRequest struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// handleListProjects GET /api/projects
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	projects, err := NewAccountScoped(s.store, accountID).ListProjects(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

// handleCreateProject POST /api/projects
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req createProjectRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	if req.ProjectID == "" || req.Name == "" {
		writeError(w, harness.ErrValidation("project_id and name are required", nil))
		return
	}
	p, err := NewAccountScoped(s.store, accountID).CreateProject(r.Context(), req.ProjectID, req.Name, req.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// handleGetProject GET /api/projects/{id}
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	p, err := NewAccountScoped(s.store, accountID).GetProject(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if p == nil {
		writeError(w, harness.ErrNotFound("project", r.PathValue("id")))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleUpdateProject PATCH /api/projects/{id}
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req createProjectRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	scoped := NewAccountScoped(s.store, accountID)
	// 先确认存在且属主，再更新
	existing, err := scoped.GetProject(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if existing == nil {
		writeError(w, harness.ErrNotFound("project", r.PathValue("id")))
		return
	}
	if err := scoped.UpdateProject(r.Context(), r.PathValue("id"), req.Name, req.Description); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleDeleteProject DELETE /api/projects/{id}
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := NewAccountScoped(s.store, accountID).DeleteProject(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

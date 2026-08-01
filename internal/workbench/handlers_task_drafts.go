package workbench

import (
	"encoding/json"
	"net/http"

	"slim-agent/internal/harness"
)

// createDraftRequest 创建草案请求体。
type createDraftRequest struct {
	DraftID        string `json:"draft_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	Objective      string `json:"objective"`
	SkillID        string `json:"skill_id,omitempty"`
}

// handleListDrafts GET /api/task-drafts
func (s *Server) handleListDrafts(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	list, err := NewAccountScoped(s.store, accountID).ListDrafts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_drafts": list})
}

// handleCreateDraft POST /api/task-drafts
func (s *Server) handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req createDraftRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	d, err := NewAccountScoped(s.store, accountID).CreateDraft(r.Context(), req.DraftID, req.ConversationID, req.Objective, req.SkillID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

// handleGetDraft GET /api/task-drafts/{id}
func (s *Server) handleGetDraft(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	d, err := NewAccountScoped(s.store, accountID).GetDraft(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if d == nil {
		writeError(w, harness.ErrNotFound("task_draft", r.PathValue("id")))
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// handleUpdateDraft PATCH /api/task-drafts/{id}
func (s *Server) handleUpdateDraft(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req struct {
		Objective string `json:"objective"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	if err := NewAccountScoped(s.store, accountID).UpdateDraftObjective(r.Context(), r.PathValue("id"), req.Objective); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleDeleteDraft DELETE /api/task-drafts/{id}
func (s *Server) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := NewAccountScoped(s.store, accountID).DeleteDraft(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleApproveDraft POST /api/task-drafts/{id}/approve
// D-011：draft→approved，创建 TaskContract + HarnessStore.CreateRun，返回 run_id。
func (s *Server) handleApproveDraft(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	state, err := s.drafts.ApproveDraft(r.Context(), accountID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"run_id": state.RunID, "status": "approved"})
}

// handleRejectDraft POST /api/task-drafts/{id}/reject
func (s *Server) handleRejectDraft(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := s.drafts.RejectDraft(r.Context(), accountID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

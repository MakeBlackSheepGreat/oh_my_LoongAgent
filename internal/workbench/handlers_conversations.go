package workbench

import (
	"encoding/json"
	"net/http"
	"strconv"

	"slim-agent/internal/harness"
)

// createConversationRequest 创建会话请求体。
type createConversationRequest struct {
	ConversationID string `json:"conversation_id"`
	ProjectID      string `json:"project_id,omitempty"`
	Title          string `json:"title"`
}

// appendMessageRequest 追加消息请求体。
type appendMessageRequest struct {
	MessageID string `json:"message_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
}

// handleListConversations GET /api/conversations
func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	list, err := NewAccountScoped(s.store, accountID).ListConversations(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": list})
}

// handleCreateConversation POST /api/conversations
func (s *Server) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req createConversationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	c, err := NewAccountScoped(s.store, accountID).CreateConversation(r.Context(), req.ConversationID, req.ProjectID, req.Title)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// handleGetConversation GET /api/conversations/{id}
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	c, err := NewAccountScoped(s.store, accountID).GetConversation(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if c == nil {
		writeError(w, harness.ErrNotFound("conversation", r.PathValue("id")))
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// handleUpdateConversation PATCH /api/conversations/{id}
func (s *Server) handleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	if err := NewAccountScoped(s.store, accountID).UpdateConversationTitle(r.Context(), r.PathValue("id"), req.Title); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleDeleteConversation DELETE /api/conversations/{id}
func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := NewAccountScoped(s.store, accountID).DeleteConversation(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleListMessages GET /api/conversations/{id}/messages?limit=&offset=
func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, err := NewAccountScoped(s.store, accountID).ListMessages(r.Context(), r.PathValue("id"), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": list})
}

// handleAppendMessage POST /api/conversations/{id}/messages
func (s *Server) handleAppendMessage(w http.ResponseWriter, r *http.Request) {
	accountID, err := AccountFromContext(r.Context())
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req appendMessageRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, harness.ErrValidation("invalid request body", nil))
		return
	}
	m, err := NewAccountScoped(s.store, accountID).AppendMessage(r.Context(), req.MessageID, r.PathValue("id"), req.Role, req.Content)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

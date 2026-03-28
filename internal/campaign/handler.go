package campaign

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/pkg/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	campaign, err := h.svc.CreateCampaign(r.Context(), userID, body.Name, body.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, campaign)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	campaigns, err := h.svc.ListCampaigns(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list campaigns")
		return
	}

	if campaigns == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, campaigns)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	campaign, err := h.svc.GetCampaign(r.Context(), campaignID)
	if err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}

	writeJSON(w, http.StatusOK, campaign)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	campaign, err := h.svc.UpdateCampaign(r.Context(), campaignID, role, body.Name, body.Description)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, campaign)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	if err := h.svc.DeleteCampaign(r.Context(), campaignID, role); err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete campaign")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	members, err := h.svc.ListMembers(r.Context(), campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	if members == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		// Aceita user_id (UUID), email ou username
		Identifier string `json:"identifier"`
		Role       string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Identifier == "" {
		writeError(w, http.StatusBadRequest, "identifier is required (user_id, email or username)")
		return
	}

	member, err := h.svc.AddMember(r.Context(), campaignID, role, body.Identifier, body.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrAlreadyMember):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	targetUserID := mustParseUUID(chi.URLParam(r, "userID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.UpdateMemberRole(r.Context(), campaignID, role, targetUserID, body.Role); err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrCannotModifyCreator):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrLastGM):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNotMember):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	targetUserID := mustParseUUID(chi.URLParam(r, "userID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	if err := h.svc.RemoveMember(r.Context(), campaignID, role, targetUserID); err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrCannotModifyCreator):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrLastGM):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNotMember):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to remove member")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateInvite gera um código/link de convite para a campanha.
func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	role := middleware.CampaignRoleFromContext(r.Context())
	userID, _ := middleware.UserIDFromContext(r.Context())

	var body struct {
		ExpiresInHours int `json:"expires_in_hours"` // 0 = sem expiração
	}
	// body é opcional
	json.NewDecoder(r.Body).Decode(&body)

	invite, err := h.svc.CreateInvite(r.Context(), campaignID, role, userID, body.ExpiresInHours)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create invite")
		return
	}

	writeJSON(w, http.StatusCreated, invite)
}

// JoinByCode entra numa campanha usando um código de convite.
func (h *Handler) JoinByCode(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	member, err := h.svc.JoinByCode(r.Context(), body.Code, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInviteExpired):
			writeError(w, http.StatusGone, err.Error())
		case errors.Is(err, ErrAlreadyMember):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, member)
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

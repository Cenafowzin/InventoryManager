package character

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
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	requesterRole := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		OwnerUserID      *string  `json:"owner_user_id"`
		MaxCarryWeightKg *float64 `json:"max_carry_weight_kg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ownerUserID := requesterID
	if body.OwnerUserID != nil {
		parsed, err := uuid.Parse(*body.OwnerUserID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid owner_user_id")
			return
		}
		ownerUserID = parsed
	}

	ch, err := h.svc.CreateCharacter(r.Context(), campaignID, requesterID, requesterRole, ownerUserID, body.Name, body.Description, body.MaxCarryWeightKg)
	if errors.Is(err, ErrOwnerNotMember) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	requesterRole := middleware.CampaignRoleFromContext(r.Context())

	chars, err := h.svc.ListCharacters(r.Context(), campaignID, requesterID, requesterRole)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chars == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, chars)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	requesterRole := middleware.CampaignRoleFromContext(r.Context())

	ch, err := h.svc.GetCharacter(r.Context(), charID, requesterID, requesterRole)
	if errors.Is(err, ErrCharacterNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	requesterRole := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		MaxCarryWeightKg *float64 `json:"max_carry_weight_kg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ch, err := h.svc.UpdateCharacter(r.Context(), charID, requesterID, requesterRole, body.Name, body.Description, body.MaxCarryWeightKg)
	if errors.Is(err, ErrCharacterNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	requesterRole := middleware.CampaignRoleFromContext(r.Context())

	err := h.svc.DeleteCharacter(r.Context(), charID, requesterID, requesterRole)
	if errors.Is(err, ErrCharacterNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

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

package category

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
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat, err := h.svc.CreateCategory(r.Context(), campaignID, role, body.Name, body.Color)
	if errors.Is(err, ErrDuplicateName) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil && err.Error() == "forbidden" {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cat)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	cats, err := h.svc.ListCategories(r.Context(), campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cats == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	catID := mustParseUUID(chi.URLParam(r, "catID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat, err := h.svc.UpdateCategory(r.Context(), catID, role, body.Name, body.Color)
	if errors.Is(err, ErrCategoryNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrDuplicateName) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil && err.Error() == "forbidden" {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	catID := mustParseUUID(chi.URLParam(r, "catID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	err := h.svc.DeleteCategory(r.Context(), catID, role)
	if errors.Is(err, ErrCategoryNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil && err.Error() == "forbidden" {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

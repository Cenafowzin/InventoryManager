package shop

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

func (h *Handler) CreateShopItem(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(chi.URLParam(r, "campaignID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid campaign_id")
		return
	}

	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Emoji       string      `json:"emoji"`
		WeightKg    float64     `json:"weight_kg"`
		BaseValue   float64     `json:"base_value"`
		ValueCoinID *uuid.UUID  `json:"value_coin_id"`
		IsAvailable *bool       `json:"is_available"`
		CategoryIDs []uuid.UUID `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isAvailable := true
	if body.IsAvailable != nil {
		isAvailable = *body.IsAvailable
	}

	item, err := h.svc.CreateShopItem(r.Context(), campaignID, role, CreateShopItemInput{
		Name:        body.Name,
		Description: body.Description,
		Emoji:       body.Emoji,
		WeightKg:    body.WeightKg,
		BaseValue:   body.BaseValue,
		ValueCoinID: body.ValueCoinID,
		IsAvailable: isAvailable,
		CategoryIDs: body.CategoryIDs,
	})
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) ListShopItems(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(chi.URLParam(r, "campaignID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid campaign_id")
		return
	}

	role := middleware.CampaignRoleFromContext(r.Context())
	query := r.URL.Query()

	var filters ListShopItemsFilters
	if catID, err := uuid.Parse(query.Get("category_id")); err == nil {
		filters.CategoryID = &catID
	}
	includeUnavailable := query.Get("include_unavailable") == "true"

	items, err := h.svc.ListShopItems(r.Context(), campaignID, role, filters, includeUnavailable)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetShopItem(w http.ResponseWriter, r *http.Request) {
	shopItemID, err := uuid.Parse(chi.URLParam(r, "shopItemID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shop_item_id")
		return
	}

	role := middleware.CampaignRoleFromContext(r.Context())

	item, err := h.svc.GetShopItemByID(r.Context(), shopItemID, role)
	if errors.Is(err, ErrShopItemNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateShopItem(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(chi.URLParam(r, "campaignID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid campaign_id")
		return
	}
	shopItemID, err := uuid.Parse(chi.URLParam(r, "shopItemID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shop_item_id")
		return
	}

	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Emoji       string      `json:"emoji"`
		WeightKg    float64     `json:"weight_kg"`
		BaseValue   float64     `json:"base_value"`
		ValueCoinID *uuid.UUID  `json:"value_coin_id"`
		IsAvailable *bool       `json:"is_available"`
		CategoryIDs []uuid.UUID `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.UpdateShopItem(r.Context(), shopItemID, campaignID, role, UpdateShopItemInput{
		Name:        body.Name,
		Description: body.Description,
		Emoji:       body.Emoji,
		WeightKg:    body.WeightKg,
		BaseValue:   body.BaseValue,
		ValueCoinID: body.ValueCoinID,
		IsAvailable: body.IsAvailable,
		CategoryIDs: body.CategoryIDs,
	})
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrShopItemNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteShopItem(w http.ResponseWriter, r *http.Request) {
	shopItemID, err := uuid.Parse(chi.URLParam(r, "shopItemID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid shop_item_id")
		return
	}

	role := middleware.CampaignRoleFromContext(r.Context())

	err = h.svc.DeleteShopItem(r.Context(), shopItemID, role)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrShopItemNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

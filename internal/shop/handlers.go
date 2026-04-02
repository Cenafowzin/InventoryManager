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

// ── Shops ─────────────────────────────────────────────────────────────────────

func (h *Handler) ListShops(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	shops, err := h.svc.ListShops(r.Context(), campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if shops == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, shops)
}

func (h *Handler) CreateShop(w http.ResponseWriter, r *http.Request) {
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
	if body.Color == "" {
		body.Color = "#6366f1"
	}

	shop, err := h.svc.CreateShop(r.Context(), campaignID, role, body.Name, body.Color)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, shop)
}

func (h *Handler) UpdateShop(w http.ResponseWriter, r *http.Request) {
	shopID := mustParseUUID(chi.URLParam(r, "shopID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		IsActive *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	shop, err := h.svc.UpdateShop(r.Context(), shopID, role, body.Name, body.Color, isActive)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrShopNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, shop)
}

func (h *Handler) DeleteShop(w http.ResponseWriter, r *http.Request) {
	shopID := mustParseUUID(chi.URLParam(r, "shopID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	err := h.svc.DeleteShop(r.Context(), shopID, role)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrShopNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Shop items ────────────────────────────────────────────────────────────────

func (h *Handler) CreateShopItem(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name          string      `json:"name"`
		Description   string      `json:"description"`
		Emoji         string      `json:"emoji"`
		WeightKg      float64     `json:"weight_kg"`
		BaseValue     float64     `json:"base_value"`
		ValueCoinID   *uuid.UUID  `json:"value_coin_id"`
		ShopID        *uuid.UUID  `json:"shop_id"`
		StockQuantity *int        `json:"stock_quantity"`
		IsAvailable   *bool       `json:"is_available"`
		CategoryIDs   []uuid.UUID `json:"category_ids"`
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
		Name: body.Name, Description: body.Description, Emoji: body.Emoji,
		WeightKg: body.WeightKg, BaseValue: body.BaseValue, ValueCoinID: body.ValueCoinID,
		ShopID: body.ShopID, StockQuantity: body.StockQuantity,
		IsAvailable: isAvailable, CategoryIDs: body.CategoryIDs,
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
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	role := middleware.CampaignRoleFromContext(r.Context())
	query := r.URL.Query()

	var filters ListShopItemsFilters
	if catID, err := uuid.Parse(query.Get("category_id")); err == nil {
		filters.CategoryID = &catID
	}
	if shopID, err := uuid.Parse(query.Get("shop_id")); err == nil {
		filters.ShopID = &shopID
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
	shopItemID := mustParseUUID(chi.URLParam(r, "shopItemID"))
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
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	shopItemID := mustParseUUID(chi.URLParam(r, "shopItemID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name          string      `json:"name"`
		Description   string      `json:"description"`
		Emoji         string      `json:"emoji"`
		WeightKg      float64     `json:"weight_kg"`
		BaseValue     float64     `json:"base_value"`
		ValueCoinID   *uuid.UUID  `json:"value_coin_id"`
		ShopID        *uuid.UUID  `json:"shop_id"`
		StockQuantity *int        `json:"stock_quantity"`
		IsAvailable   *bool       `json:"is_available"`
		CategoryIDs   []uuid.UUID `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isAvailable := true
	if body.IsAvailable != nil {
		isAvailable = *body.IsAvailable
	}

	item, err := h.svc.UpdateShopItem(r.Context(), shopItemID, campaignID, role, UpdateShopItemInput{
		Name: body.Name, Description: body.Description, Emoji: body.Emoji,
		WeightKg: body.WeightKg, BaseValue: body.BaseValue, ValueCoinID: body.ValueCoinID,
		ShopID: body.ShopID, StockQuantity: body.StockQuantity,
		IsAvailable: isAvailable, CategoryIDs: body.CategoryIDs,
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
	shopItemID := mustParseUUID(chi.URLParam(r, "shopItemID"))
	role := middleware.CampaignRoleFromContext(r.Context())

	err := h.svc.DeleteShopItem(r.Context(), shopItemID, role)
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

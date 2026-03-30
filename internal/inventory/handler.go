package inventory

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/pkg/middleware"
)

type Handler struct {
	storageSvc *StorageService
	itemSvc    *ItemService
	coinSvc    *CoinService
	summarySvc *SummaryService
}

func NewHandler(storageSvc *StorageService, itemSvc *ItemService, coinSvc *CoinService, summarySvc *SummaryService) *Handler {
	return &Handler{
		storageSvc: storageSvc,
		itemSvc:    itemSvc,
		coinSvc:    coinSvc,
		summarySvc: summarySvc,
	}
}

// ── Storage Spaces ─────────────────────────────────────────────────────────────

func (h *Handler) CreateStorage(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name             string     `json:"name"`
		Description      string     `json:"description"`
		CountsTowardLoad *bool      `json:"counts_toward_load"`
		CapacityKg       *float64   `json:"capacity_kg"`
		ItemID           *uuid.UUID `json:"item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	counts := true
	if body.CountsTowardLoad != nil {
		counts = *body.CountsTowardLoad
	}

	ss, err := h.storageSvc.CreateStorageSpace(r.Context(), charID, requesterID, role, body.Name, body.Description, counts, body.CapacityKg, body.ItemID)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrDuplicateStorageName) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ss)
}

func (h *Handler) ListStorages(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	spaces, err := h.storageSvc.ListStorageSpaces(r.Context(), charID, requesterID, role)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if spaces == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, spaces)
}

func (h *Handler) UpdateStorage(w http.ResponseWriter, r *http.Request) {
	storageID := mustParseUUID(chi.URLParam(r, "storageID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name             string   `json:"name"`
		Description      string   `json:"description"`
		CountsTowardLoad *bool    `json:"counts_toward_load"`
		CapacityKg       *float64 `json:"capacity_kg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	counts := true
	if body.CountsTowardLoad != nil {
		counts = *body.CountsTowardLoad
	}

	ss, err := h.storageSvc.UpdateStorageSpace(r.Context(), storageID, requesterID, role, body.Name, body.Description, counts, body.CapacityKg)
	if errors.Is(err, ErrStorageNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrDuplicateStorageName) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ss)
}

func (h *Handler) DeleteStorage(w http.ResponseWriter, r *http.Request) {
	storageID := mustParseUUID(chi.URLParam(r, "storageID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	err := h.storageSvc.DeleteStorageSpace(r.Context(), storageID, requesterID, role)
	if errors.Is(err, ErrStorageNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrCannotDeleteDefault) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Items ──────────────────────────────────────────────────────────────────────

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name           string      `json:"name"`
		Description    string      `json:"description"`
		Emoji          string      `json:"emoji"`
		WeightKg       float64     `json:"weight_kg"`
		WeightUnit     string      `json:"weight_unit"`
		Value          float64     `json:"value"`
		ValueCoinID    *uuid.UUID  `json:"value_coin_id"`
		StorageSpaceID *uuid.UUID  `json:"storage_space_id"`
		CategoryIDs    []uuid.UUID `json:"category_ids"`
		Quantity       int         `json:"quantity"`
		ShopItemID     *uuid.UUID  `json:"shop_item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.itemSvc.CreateItem(r.Context(), charID, requesterID, role, ItemInput{
		Name: body.Name, Description: body.Description, Emoji: body.Emoji,
		WeightKg: body.WeightKg, WeightUnit: body.WeightUnit,
		Value: body.Value, ValueCoinID: body.ValueCoinID,
		StorageSpaceID: body.StorageSpaceID, CategoryIDs: body.CategoryIDs,
		Quantity: body.Quantity, ShopItemID: body.ShopItemID,
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

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var filters ItemFilters
	if v := r.URL.Query().Get("category_id"); v != "" {
		id := mustParseUUID(v)
		filters.CategoryID = &id
	}
	if v := r.URL.Query().Get("storage_id"); v != "" {
		id := mustParseUUID(v)
		filters.StorageSpaceID = &id
	}

	items, err := h.itemSvc.ListItems(r.Context(), charID, requesterID, role, filters)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
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

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	itemID := mustParseUUID(chi.URLParam(r, "itemID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	item, err := h.itemSvc.GetItem(r.Context(), itemID, requesterID, role)
	if errors.Is(err, ErrItemNotFound) {
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
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := mustParseUUID(chi.URLParam(r, "itemID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Name           string      `json:"name"`
		Description    string      `json:"description"`
		Emoji          string      `json:"emoji"`
		WeightKg       float64     `json:"weight_kg"`
		WeightUnit     string      `json:"weight_unit"`
		Value          float64     `json:"value"`
		ValueCoinID    *uuid.UUID  `json:"value_coin_id"`
		StorageSpaceID *uuid.UUID  `json:"storage_space_id"`
		CategoryIDs    []uuid.UUID `json:"category_ids"`
		Quantity       int         `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.itemSvc.UpdateItem(r.Context(), itemID, requesterID, role, ItemInput{
		Name: body.Name, Description: body.Description, Emoji: body.Emoji,
		WeightKg: body.WeightKg, WeightUnit: body.WeightUnit,
		Value: body.Value, ValueCoinID: body.ValueCoinID,
		StorageSpaceID: body.StorageSpaceID, CategoryIDs: body.CategoryIDs,
		Quantity: body.Quantity,
	})
	if errors.Is(err, ErrItemNotFound) {
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
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := mustParseUUID(chi.URLParam(r, "itemID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	err := h.itemSvc.DeleteItem(r.Context(), itemID, requesterID, role)
	if errors.Is(err, ErrItemNotFound) {
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

// ── Coins ──────────────────────────────────────────────────────────────────────

func (h *Handler) GetCoinPurse(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	purse, err := h.coinSvc.GetCoinPurse(r.Context(), charID, requesterID, role)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if purse == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, purse)
}

func (h *Handler) SetCoinBalance(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	coinID := mustParseUUID(chi.URLParam(r, "coinID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	purse, err := h.coinSvc.SetCoinBalance(r.Context(), charID, coinID, requesterID, role, body.Amount)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, purse)
}

func (h *Handler) ConvertCoins(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		FromCoinID uuid.UUID `json:"from_coin_id"`
		ToCoinID   uuid.UUID `json:"to_coin_id"`
		Amount     float64   `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.coinSvc.ConvertCoins(r.Context(), charID, body.FromCoinID, body.ToCoinID, requesterID, role, body.Amount)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrInsufficientFunds) {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if errors.Is(err, ErrNoConversion) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Summary ────────────────────────────────────────────────────────────────────

func (h *Handler) GetLoad(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	summary, err := h.summarySvc.GetLoad(r.Context(), charID, requesterID, role)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) GetInventorySummary(w http.ResponseWriter, r *http.Request) {
	charID := mustParseUUID(chi.URLParam(r, "charID"))
	requesterID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	summary, err := h.summarySvc.GetInventorySummary(r.Context(), charID, requesterID, role)
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// ── helpers ────────────────────────────────────────────────────────────────────

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

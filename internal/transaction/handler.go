package transaction

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
	userID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		CharacterID uuid.UUID   `json:"character_id"`
		Type        string      `json:"type"`
		TotalCoinID *uuid.UUID  `json:"total_coin_id"`
		Items       []itemInput `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items := make([]ItemInput, len(body.Items))
	for i, it := range body.Items {
		items[i] = ItemInput{
			ShopItemID:      it.ShopItemID,
			InventoryItemID: it.InventoryItemID,
			Quantity:        it.Quantity,
		}
	}

	tx, err := h.svc.CreateDraft(r.Context(), campaignID, userID, role, CreateInput{
		CharacterID: body.CharacterID,
		Type:        body.Type,
		TotalCoinID: body.TotalCoinID,
		Items:       items,
	})
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrItemUnavailable) || errors.Is(err, ErrNoItems) {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	userID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())
	q := r.URL.Query()

	var f ListFilters
	if v := q.Get("character_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.CharacterID = &id
		}
	}
	if v := q.Get("status"); v != "" {
		f.Status = &v
	}
	if v := q.Get("type"); v != "" {
		f.Type = &v
	}

	txs, err := h.svc.List(r.Context(), campaignID, userID, role, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, txs)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	txID := mustParseUUID(chi.URLParam(r, "txID"))
	userID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	tx, err := h.svc.Get(r.Context(), txID, userID, role)
	if errors.Is(err, ErrTransactionNotFound) {
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
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) Adjust(w http.ResponseWriter, r *http.Request) {
	txID := mustParseUUID(chi.URLParam(r, "txID"))
	userID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	var body struct {
		AdjustedTotal *float64 `json:"adjusted_total"`
		Notes         *string  `json:"notes"`
		Items         []struct {
			ID                uuid.UUID `json:"id"`
			AdjustedUnitValue float64   `json:"adjusted_unit_value"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	itemAdj := make([]ItemAdjustInput, len(body.Items))
	for i, it := range body.Items {
		itemAdj[i] = ItemAdjustInput{ItemID: it.ID, AdjustedUnitValue: it.AdjustedUnitValue}
	}

	tx, err := h.svc.Adjust(r.Context(), txID, userID, role, AdjustInput{
		AdjustedTotal: body.AdjustedTotal,
		Notes:         body.Notes,
		Items:         itemAdj,
	})
	if errors.Is(err, ErrConflictingAdjust) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, ErrNotDraft) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTransactionNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	txID := mustParseUUID(chi.URLParam(r, "txID"))
	userID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	tx, err := h.svc.Confirm(r.Context(), txID, userID, role)
	if errors.Is(err, ErrInsufficientFunds) || errors.Is(err, ErrInventoryItemMissing) {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if errors.Is(err, ErrNotDraft) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTransactionNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	txID := mustParseUUID(chi.URLParam(r, "txID"))
	userID, _ := middleware.UserIDFromContext(r.Context())
	role := middleware.CampaignRoleFromContext(r.Context())

	tx, err := h.svc.Cancel(r.Context(), txID, userID, role)
	if errors.Is(err, ErrNotDraft) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, ErrForbidden) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if errors.Is(err, ErrTransactionNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

// ── helpers ───────────────────────────────────────────────────────────────────

type itemInput struct {
	ShopItemID      *uuid.UUID `json:"shop_item_id"`
	InventoryItemID *uuid.UUID `json:"inventory_item_id"`
	Quantity        int        `json:"quantity"`
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

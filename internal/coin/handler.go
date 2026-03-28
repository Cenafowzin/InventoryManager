package coin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ── CoinTypes ─────────────────────────────────────────────────────────────────

func (h *Handler) CreateCoinType(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	var body struct {
		Name         string `json:"name"`
		Abbreviation string `json:"abbreviation"`
		Emoji        string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	coin, err := h.svc.CreateCoinType(r.Context(), campaignID, body.Name, body.Abbreviation, body.Emoji)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, coin)
}

func (h *Handler) ListCoinTypes(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	coins, err := h.svc.ListCoinTypes(r.Context(), campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if coins == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, coins)
}

func (h *Handler) GetCoinType(w http.ResponseWriter, r *http.Request) {
	coinID := mustParseUUID(chi.URLParam(r, "coinID"))

	coin, err := h.svc.GetCoinType(r.Context(), coinID)
	if errors.Is(err, ErrCoinNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, coin)
}

func (h *Handler) UpdateCoinType(w http.ResponseWriter, r *http.Request) {
	coinID := mustParseUUID(chi.URLParam(r, "coinID"))

	var body struct {
		Name         string `json:"name"`
		Abbreviation string `json:"abbreviation"`
		Emoji        string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	coin, err := h.svc.UpdateCoinType(r.Context(), coinID, body.Name, body.Abbreviation, body.Emoji)
	if errors.Is(err, ErrCoinNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, coin)
}

func (h *Handler) DeleteCoinType(w http.ResponseWriter, r *http.Request) {
	coinID := mustParseUUID(chi.URLParam(r, "coinID"))

	err := h.svc.DeleteCoinType(r.Context(), coinID)
	if errors.Is(err, ErrCoinNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrCoinInUse) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetDefaultCoin(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))
	coinID := mustParseUUID(chi.URLParam(r, "coinID"))

	err := h.svc.SetDefaultCoin(r.Context(), campaignID, coinID)
	if errors.Is(err, ErrCoinNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetDefaultCoin(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	coin, err := h.svc.GetDefaultCoin(r.Context(), campaignID)
	if errors.Is(err, ErrNoDefaultCoin) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, coin)
}

// ── CoinConversions ───────────────────────────────────────────────────────────

func (h *Handler) CreateConversion(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	var body struct {
		FromCoinID uuid.UUID `json:"from_coin_id"`
		ToCoinID   uuid.UUID `json:"to_coin_id"`
		Rate       float64   `json:"rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	convs, err := h.svc.CreateConversion(r.Context(), campaignID, body.FromCoinID, body.ToCoinID, body.Rate)
	if errors.Is(err, ErrSameCoin) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, convs)
}

func (h *Handler) ListConversions(w http.ResponseWriter, r *http.Request) {
	campaignID := mustParseUUID(chi.URLParam(r, "campaignID"))

	convs, err := h.svc.ListConversions(r.Context(), campaignID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if convs == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, convs)
}

func (h *Handler) GetConversion(w http.ResponseWriter, r *http.Request) {
	convID := mustParseUUID(chi.URLParam(r, "conversionID"))

	conv, err := h.svc.GetConversion(r.Context(), convID)
	if errors.Is(err, ErrConversionNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (h *Handler) DeleteConversion(w http.ResponseWriter, r *http.Request) {
	convID := mustParseUUID(chi.URLParam(r, "conversionID"))

	err := h.svc.DeleteConversion(r.Context(), convID)
	if errors.Is(err, ErrConversionNotFound) {
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

// mustParseUUID parses a UUID string; returns uuid.Nil on failure (caller should validate).
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


package discord

import (
	"encoding/json"
	"net/http"

	"github.com/rubendubeux/inventory-manager/pkg/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /auth/discord/code — usuário logado gera o código para usar no bot (/link <codigo>)
func (h *Handler) GenerateCode(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}

	code, err := h.svc.GenerateLinkCode(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao gerar código")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"code": code})
}

// GET /auth/discord/status — verifica se o usuário logado tem Discord vinculado
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}
	linked, err := h.svc.IsLinked(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao verificar")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"linked": linked})
}

// DELETE /auth/discord/link — remove a vinculação
func (h *Handler) Unlink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "não autorizado")
		return
	}
	if err := h.svc.Unlink(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao desvincular")
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

package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type campaignRoleKey string

const roleKey campaignRoleKey = "campaign_role"

// RequireCampaignRole retorna um middleware que verifica se o usuário autenticado
// é membro da campanha e possui ao menos a role mínima exigida.
// Injeta a role no context para uso nos handlers.
func RequireCampaignRole(db *pgxpool.Pool, minRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				writeForbidden(w)
				return
			}

			campaignIDStr := chi.URLParam(r, "campaignID")
			campaignID, err := uuid.Parse(campaignIDStr)
			if err != nil {
				writeNotFound(w)
				return
			}

			var role string
			err = db.QueryRow(r.Context(), `
				SELECT role FROM campaign_members
				WHERE campaign_id = $1 AND user_id = $2
			`, campaignID, userID).Scan(&role)
			if err != nil {
				// Não é membro
				writeNotFound(w)
				return
			}

			if !hasRole(role, minRole) {
				writeForbidden(w)
				return
			}

			ctx := context.WithValue(r.Context(), roleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CampaignRoleFromContext retorna a role do usuário na campanha atual.
func CampaignRoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(roleKey).(string)
	return role
}

// hasRole verifica se a role atual satisfaz o mínimo exigido.
func hasRole(actual, minimum string) bool {
	rank := map[string]int{"player": 1, "gm": 2}
	return rank[actual] >= rank[minimum]
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
}

func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "campaign not found or not a member"})
}

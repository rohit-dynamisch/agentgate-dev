package govapi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// AdminAuthMiddleware validates that the request presents valid administrator credentials.
func AdminAuthMiddleware(adminToken string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Code:    "UNAUTHORIZED",
					Message: "valid admin credentials required",
				},
			})
			return
		}
		next(w, r)
	}
}

func extractToken(r *http.Request) string {
	if header := r.Header.Get("X-AgentGate-Admin-Key"); header != "" {
		return strings.TrimSpace(header)
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

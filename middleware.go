package main

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// requireAuth is middleware that validates the "Authorization: Bearer <token>"
// header and, on success, stores the authenticated user's ID in the request
// context for downstream handlers to use.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "Authorization header must be: Bearer <token>")
			return
		}

		userID, err := parseAndVerifyToken(parts[1], s.jwtSecret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// userIDFromContext extracts the authenticated user's ID set by requireAuth.
func userIDFromContext(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(userIDContextKey).(string)
	return id, ok
}

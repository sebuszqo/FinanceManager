package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// JWTAuthMiddleware validates JWT tokens
func JWTAuthMiddleware(jwtManager JWTManagerInterface, unprotectedPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the request path is unprotected
			for _, path := range unprotectedPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "Authorization header is required")
				return
			}

			// Extract the token from the header
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				writeJSONError(w, http.StatusUnauthorized, "Invalid token format")
				return
			}

			// Verify the JWT token
			userID, err := jwtManager.VerifyJWT(tokenString)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Add userID to the request context and pass to the next handler
			ctx := context.WithValue(r.Context(), "userID", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeJSONError writes an error response in JSON format
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Status:  "error",
		Message: message,
	})
}

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (s *authService) JWTAccessTokenMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "Authorization header is required")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				writeJSONError(w, http.StatusUnauthorized, "Invalid token format")
				return
			}

			userID, err := s.jwtManager.ValidateAccessToken(tokenString)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			_, err = s.repo.GetUserByID(userID)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					writeJSONError(w, http.StatusUnauthorized, ErrUserNotFound.Error())
					return
				} else {
					fmt.Println("TU ZWRACAM?")
					writeJSONError(w, http.StatusInternalServerError, ErrInternalError.Error())
					return
				}
			}

			ctx := context.WithValue(r.Context(), "userID", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *authService) JWTRefreshTokenMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			userID, err := s.jwtManager.ExtractUserIDFromRefreshToken(tokenString)
			if err != nil {
				if errors.Is(err, ErrExpiredJWTToken) {
					writeJSONError(w, http.StatusUnauthorized, ErrExpiredJWTToken.Error())
					return
				}
				writeJSONError(w, http.StatusInternalServerError, ErrInternalError.Error())
				return
			}

			existingUser, err := s.repo.GetUserByID(userID)
			if err != nil {
				if errors.Is(err, ErrUserNotFound) {
					writeJSONError(w, http.StatusUnauthorized, ErrUserNotFound.Error())
					return
				} else {
					writeJSONError(w, http.StatusInternalServerError, ErrInternalError.Error())
					return
				}
			}
			err = s.jwtManager.ValidateRefreshToken(tokenString, existingUser.HashToken)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, ErrInvalidJWTRefreshToken.Error())
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
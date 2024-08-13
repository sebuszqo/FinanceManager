package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	authService AuthService
}

func NewHandler(authService AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"status":  "error",
		"message": message,
		"code":    status,
	})
}

func (s *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := s.authService.Register(req.Email, req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) || errors.Is(err, ErrLoginAlreadyExists) {
			respondError(w, http.StatusConflict, err.Error())
			return
		} else if errors.Is(err, ErrInvalidEmail) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not register user")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"user_id": user.ID,
		},
	})
}

func (s *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmailOrLogin string `json:"email_or_login"`
		Password     string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, sessionTokenOrJWT, refreshToken, err := s.authService.Login(req.EmailOrLogin, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		if errors.Is(err, ErrInvalidTwoFactorMethod) {
			respondError(w, http.StatusInternalServerError, "Invalid two-factor method")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if refreshToken == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"data": map[string]string{
				"message":         "Two-factor authentication required",
				"2fa_auth_method": user.TwoFactorMethod,
				"session_token":   sessionTokenOrJWT,
			},
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"access_token":  sessionTokenOrJWT,
			"refresh_token": refreshToken,
		},
	})
}

func (s *Handler) HandleRegisterTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID := r.Context().Value("userID").(string)

	otpURI, err := s.authService.RegisterTwoFactor(userID, req.Method)
	if err != nil {
		if errors.Is(err, ErrInvalidTwoFactorMethod) || errors.Is(err, ErrUser2FAAlreadyEnabled) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not register two-factor authentication")
		return
	}

	if req.Method == google2FAAuthMethod {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "Two-factor authentication enabled",
			"data": map[string]string{
				"otp_uri": otpURI,
			},
		})
	} else {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "Two-factor authentication enabled",
		})
	}
}

func (s *Handler) HandleVerifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionToken string `json:"session_token"`
		Code         string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, jwtToken, refreshToken, err := s.authService.VerifyTwoFactor(req.SessionToken, req.Code)
	if err != nil {
		if errors.Is(err, ErrInvalidSessionToken) || errors.Is(err, ErrInvalid2FACode) {
			respondError(w, http.StatusUnauthorized, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not verify two-factor authentication")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"user_id":       user.ID,
			"access_token":  jwtToken,
			"refresh_token": refreshToken,
		},
	})
}

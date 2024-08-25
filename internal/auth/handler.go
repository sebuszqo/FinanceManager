package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/user"
	"net/http"
)

type Handler struct {
	authService Service
}

func NewHandler(authService Service) *Handler {
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

func checkRequestMethod(requestMethod, method string, w http.ResponseWriter) {
	if requestMethod != method {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
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
		if errors.Is(err, ErrTooManyEmailCodeRequests) {
			respondError(w, http.StatusTooManyRequests, ErrTooManyEmailCodeRequests.Error())
			return
		}
		if errors.Is(err, ErrUserNotVerified) {
			//err := s.service.SendVerificationCode(user)
			//if err != nil {
			//	respondError(w, http.StatusInternalServerError, "Failed to send verification email")
			//	return
			//}
			respondJSON(w, http.StatusForbidden, map[string]interface{}{
				"status":  "verification_required",
				"message": "Account not verified. A verification code has been sent to your email.",
			})
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
			"message": "Two-factor authentication initiated. Please verify to enable.",
			"data": map[string]string{
				"otp_uri": otpURI,
			},
		})
	} else {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "Two-factor authentication initiated. Please verify to enable.",
		})
	}
}

func (s *Handler) HandleVerifyTwoFactorCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
		Code   string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	userID := r.Context().Value("userID").(string)
	err := s.authService.VerifyTwoFactorCode(userID, req.Method, req.Code)
	if err != nil {
		fmt.Println(err.Error())
		if errors.Is(err, ErrInvalid2FACode) {
			respondError(w, http.StatusUnauthorized, "Invalid 2fa code")
			return
		} else if errors.Is(err, InvalidCodeType) {
			respondError(w, http.StatusUnauthorized, InvalidCodeType.Error())
			return
		} else if errors.Is(err, ErrVerificationCodeExpired) {
			respondError(w, http.StatusConflict, "two-factor code is expired")
			return
		} else if errors.Is(err, ErrUser2FAAlreadyEnabled) {
			respondError(w, http.StatusConflict, "Two-factor authentication is already enabled")
			return
		} else if errors.Is(err, ErrInvalidTwoFactorMethod) {
			respondError(w, http.StatusBadRequest, "Invalid two-factor method")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "success",
	})
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
		if errors.Is(err, ErrInvalidSessionToken) || errors.Is(err, ErrInvalid2FACode) || errors.Is(err, InvalidCodeType) {
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

func (s *Handler) HandleRequestEmail2FACode(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	err := s.authService.RequestEmail2FACode(userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			respondError(w, http.StatusNotFound, "User not found")
		case errors.Is(err, ErrUser2FANotEnabled):
			respondError(w, http.StatusBadRequest, "Two-factor authentication is not enabled")
		case errors.Is(err, ErrInvalidTwoFactorMethod):
			respondError(w, http.StatusBadRequest, "Invalid two-factor method")
		default:
			respondError(w, http.StatusInternalServerError, "Could not send two-factor code")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Two-factor code sent to email",
	})
}

func (s *Handler) HandleDisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	checkRequestMethod(r.Method, http.MethodPost, w)
	var req struct {
		Method string `json:"method"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authorized")
		return
	}

	err := s.authService.DisableTwoFactorAuth(userID, req.Method, req.Code)
	if err != nil {
		if errors.Is(err, ErrInvalidTwoFactorMethod) {
			respondError(w, http.StatusBadRequest, "Invalid two-factor method")
			return
		} else if errors.Is(err, user.ErrNoTwoFactorCodeGenerated) {
			respondError(w, http.StatusBadRequest, "No two-factor code generated")
			return
		} else if errors.Is(err, ErrInvalid2FACode) {
			respondError(w, http.StatusUnauthorized, "Invalid 2FA code")
			return
		} else if errors.Is(err, ErrVerificationCodeExpired) {
			respondError(w, http.StatusConflict, "Two-factor code is expired")
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not disable two-factor authentication")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Two-factor authentication disabled successfully",
	})
}

func (s *Handler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUserNotFound.Error())
	}

	accessToken, refreshToken, err := s.authService.RefreshAccessToken(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalError.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"session_token": accessToken,
			"refresh_token": refreshToken,
		},
	})
}

func (s *Handler) RequestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email string `json:"email"`
	}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = s.authService.RequestPasswordReset(request.Email)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			respondError(w, http.StatusNotFound, "User not found")
		case errors.Is(err, ErrTooManyEmailCodeRequests):
			respondError(w, http.StatusTooManyRequests, "Too many requests, please try again later")
		default:
			respondError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
	})
}

func (s *Handler) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = s.authService.ResetPassword(request.Email, request.Code, request.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			respondError(w, http.StatusNotFound, "User not found")
		case errors.Is(err, ErrInvalid2FACode):
			respondError(w, http.StatusUnauthorized, "Invalid verification code")
		case errors.Is(err, ErrVerificationCodeExpired):
			respondError(w, http.StatusUnauthorized, "Verification code expired")
		case errors.Is(err, ErrInvalidTwoFactorMethod), errors.Is(err, InvalidCodeType):
			respondError(w, http.StatusBadRequest, "Invalid request")
		case errors.Is(err, user.ErrNoTwoFactorCodeGenerated):
			respondError(w, http.StatusBadRequest, "Two-factor authentication code not generated. Please initiate password reset process.")
		default:
			respondError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Password has been reset successfully",
	})
}

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sebuszqo/FinanceManager/internal/user"
	"net/http"
	"time"
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

func checkRequestMethod(requestMethod, method string, w http.ResponseWriter) bool {
	if requestMethod != method {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
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
	if req.Password == "" || req.EmailOrLogin == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	existingUser, sessionTokenOrJWT, refreshToken, err := s.authService.Login(req.EmailOrLogin, req.Password)
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
				"2fa_auth_method": existingUser.TwoFactorMethod,
				"session_token":   sessionTokenOrJWT,
			},
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
		Path:     "/api/refresh/token",
		// Optional: Domain: "yourdomain.com",
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"access_token": sessionTokenOrJWT,
		},
	})
}

func (s *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			respondJSON(w, http.StatusOK, "Logout successful")
			return
		}
		respondError(w, http.StatusBadRequest, "Error during logout request.")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/refresh/token",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
	})

	respondJSON(w, http.StatusOK, "Logout successful")
	//http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Handler) HandleRegisterTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Method == "" {
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

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Method == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	userID := r.Context().Value("userID").(string)
	err = s.authService.VerifyTwoFactorCode(userID, req.Method, req.Code)
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

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.SessionToken == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	existingUser, jwtToken, refreshToken, err := s.authService.VerifyTwoFactor(req.SessionToken, req.Code)
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
			"user_id":       existingUser.ID,
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
	if !checkRequestMethod(r.Method, http.MethodDelete, w) {
		return
	}
	var req struct {
		Method string `json:"method"`
		Code   string `json:"code"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Method == "" || req.Code == "" {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authorized")
		return
	}

	err = s.authService.DisableTwoFactorAuth(userID, req.Method, req.Code)
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

	accessToken, newRefreshToken, err := s.authService.RefreshAccessToken(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalError.Error())
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteNoneMode,
		Path:     "/api/refresh/token",
		// Opcjonalnie: Domain: "yourdomain.com",
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]string{
			"access_token": accessToken,
		},
	})
}

func (s *Handler) RequestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Email == "" {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email/Login and Password are required")
		return
	}
	err = s.authService.RequestPasswordReset(req.Email)
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
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Code == "" || req.NewPassword == "" || req.Email == "" {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = s.authService.ResetPassword(req.Email, req.Code, req.NewPassword)
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

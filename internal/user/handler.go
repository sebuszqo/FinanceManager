package user

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type Handler struct {
	userService Service
}

func NewHandler(authService Service) *Handler {
	return &Handler{
		userService: authService,
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("JSON encoding error: %v", err)
		return
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"status":  "error",
		"message": message,
		"code":    status,
	})
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.userService.Register(req.Email, req.Login, req.Password)
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

func (h *Handler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.userService.VerifyRegistrationCode(req.Email, req.Code)
	if err != nil {
		if errors.Is(err, ErrInvalidVerificationCode) {
			respondError(w, http.StatusUnauthorized, "Invalid verification code")
			return
		} else if errors.Is(err, ErrVerificationCodeExpired) {
			respondError(w, http.StatusGone, "Verification code expired")
			return
		} else if errors.Is(err, ErrUserAlreadyVerified) {
			respondError(w, http.StatusConflict, "User already verified")
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not verify email")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "success",
	})
}

func (h *Handler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.userService.ChangePasswordWithOldPassword(userID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		} else if errors.Is(err, ErrInvalidOldPassword) {
			respondError(w, http.StatusUnauthorized, "Invalid old password")
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not change password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Password changed successfully",
	})
}

func (h *Handler) HandleRequestEmailChange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewEmail string `json:"new_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.userService.RequestEmailChange(userID, req.NewEmail)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			respondError(w, http.StatusConflict, "Email is already in use")
			return
		} else if errors.Is(err, ErrInvalidEmail) {
			respondError(w, http.StatusBadRequest, "Invalid email address")
			return
		} else if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}

		respondError(w, http.StatusInternalServerError, "Could not request email change")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Verification code sent to new email",
	})
}

func (h *Handler) HandleConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	err := h.userService.ConfirmEmailChange(userID, req.Code)
	if err != nil {
		if errors.Is(err, ErrInvalidVerificationCode) {
			respondError(w, http.StatusUnauthorized, "Invalid verification code")
			return
		} else if errors.Is(err, ErrVerificationCodeExpired) {
			respondError(w, http.StatusConflict, "Verification code expired")
			return
		}

		respondError(w, http.StatusInternalServerError, "Could not confirm email change")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Email changed successfully",
	})
}

func (h *Handler) HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Could not fetch user data")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"user_id":     user.ID,
			"email":       user.Email,
			"login":       user.Login,
			"2fa_enabled": user.TwoFactorEnabled,
			"2fa_method":  user.TwoFactorMethod,
			"created_at":  user.CreatedAt,
			"updated_at":  user.UpdatedAt,
		},
	})
}

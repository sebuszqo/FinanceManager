package investments

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	portfolios "github.com/sebuszqo/FinanceManager/internal/investment/portfolio"
	"net/http"
	"time"
)

type InvestmentHandler struct {
	portfolioService portfolios.Service
	// assetService assets.Service
	// transactionService transactions.Service
	respondJSON  func(w http.ResponseWriter, status int, payload interface{})
	respondError func(w http.ResponseWriter, status int, message string)
}

func NewInvestmentHandler(
	portfolioService portfolios.Service,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string),
) *InvestmentHandler {
	return &InvestmentHandler{
		portfolioService: portfolioService,
		respondJSON:      respondJSON,
		respondError:     respondError,
	}
}

type CreatePortfolioRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PortfolioResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h *InvestmentHandler) getUserIDReq(w http.ResponseWriter, r *http.Request) string {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return ""
	}
	return userID
}

func (h *InvestmentHandler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	var req CreatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "Portfolio name is required")
		return
	}

	portfolio, err := h.portfolioService.CreatePortfolio(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, portfolios.ErrPortfolioNameTaken) {
			h.respondError(w, http.StatusConflict, "Portfolio name already exists")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to create portfolio")
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": "Portfolio successfully created.",
		"data": PortfolioResponse{
			ID:          portfolio.ID.String(),
			Name:        portfolio.Name,
			Description: portfolio.Description,
			CreatedAt:   portfolio.CreatedAt,
			UpdatedAt:   portfolio.UpdatedAt,
		},
	})

}

func (h *InvestmentHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}
	// don't need to check for "" or something similar as it was checked by investment_middleware
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)
	portfolio, err := h.portfolioService.GetPortfolio(r.Context(), portfolioID, userID)
	if err != nil {
		if errors.Is(err, portfolios.ErrPortfolioNotFound) {
			h.respondError(w, http.StatusNotFound, "Portfolio not found")
			return
		}
		if errors.Is(err, portfolios.ErrUnauthorizedAccess) {
			h.respondError(w, http.StatusUnauthorized, "Unauthorized access to portfolio")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve portfolio")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Portfolio retrieved successfully.",
		"data": PortfolioResponse{
			ID:          portfolio.ID.String(),
			Name:        portfolio.Name,
			Description: portfolio.Description,
			CreatedAt:   portfolio.CreatedAt,
			UpdatedAt:   portfolio.UpdatedAt,
		},
	})

}

func (h *InvestmentHandler) GetAllPortfolios(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	portfoliosList, err := h.portfolioService.GetAllPortfolios(r.Context(), userID)

	if err != nil {
		fmt.Println("err", err.Error())
		if errors.Is(err, portfolios.ErrPortfolioNotFound) {
			h.respondError(w, http.StatusNotFound, "Portfolio not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve portfolios list")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "List of portfolios retrieved successfully.",
		"data":    portfoliosList,
	})

}

func (h *InvestmentHandler) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// don't need to check for "" or something similar as it was checked by investment_middleware
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if (req.Name == nil && req.Description == nil) || (*req.Name == "" && *req.Description == "") {
		h.respondError(w, http.StatusBadRequest, "At least one field (name or description) must be provided for update")
		return
	}

	if *req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "Portfolio Name cannot be empty")
		return
	}
	err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, userID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, portfolios.ErrPortfolioNotFound) {
			h.respondError(w, http.StatusNotFound, "Portfolio not found")
			return
		}
		if errors.Is(err, portfolios.ErrPortfolioNameTaken) {
			h.respondError(w, http.StatusNotFound, "Portfolio with this name already exists")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to update portfolio")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Portfolio successfully updated.",
	})
}

func (h *InvestmentHandler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	// don't need to check for "" or something similar as it was checked by investment_middleware
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)

	err := h.portfolioService.DeletePortfolio(r.Context(), portfolioID, userID)
	if err != nil {
		if errors.Is(err, portfolios.ErrPortfolioNotFound) {
			h.respondError(w, http.StatusNotFound, "Portfolio not found")
			return
		}
		if errors.Is(err, portfolios.ErrUnauthorizedAccess) {
			h.respondError(w, http.StatusUnauthorized, "Unauthorized access to portfolio")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to delete portfolio")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Portfolio deleted successfully.",
	})

}

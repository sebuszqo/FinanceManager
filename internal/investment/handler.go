package investments

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	assets "github.com/sebuszqo/FinanceManager/internal/investment/asset"
	portfolios "github.com/sebuszqo/FinanceManager/internal/investment/portfolio"
	"net/http"
	"time"
)

type InvestmentHandler struct {
	portfolioService portfolios.Service
	assetService     assets.Service
	//transactionService transactions.Service
	respondJSON  func(w http.ResponseWriter, status int, payload interface{})
	respondError func(w http.ResponseWriter, status int, message string)
}

func NewInvestmentHandler(
	portfolioService portfolios.Service,
	assetsService assets.Service,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string),
) *InvestmentHandler {
	return &InvestmentHandler{
		portfolioService: portfolioService,
		assetService:     assetsService,
		respondJSON:      respondJSON,
		respondError:     respondError,
	}
}

type createPortfolioRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type portfolioResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type createAssetRequest struct {
	Name        string `json:"name"`
	Ticker      string `json:"ticker"`
	AssetTypeID int    `json:"asset_type_id"`
	// Add other optional fields (for bonds, stocks, ETFs, etc.)
	CouponRate    *float64 `json:"coupon_rate,omitempty"`
	MaturityDate  *string  `json:"maturity_date,omitempty"`
	FaceValue     *float64 `json:"face_value,omitempty"`
	DividendYield *float64 `json:"dividend_yield,omitempty"`
	Accumulation  *bool    `json:"accumulation,omitempty"`
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

	var req createPortfolioRequest
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
		"data": portfolioResponse{
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
		"data": portfolioResponse{
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
	//if req.Description == nil {
	//	fmt.Println("DESCIRPTION is not provided:")
	//}
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

func (h *InvestmentHandler) GetAssetTypes(w http.ResponseWriter, _ *http.Request) {
	assetTypes := h.assetService.GetAssetTypes()
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   assetTypes,
	})
}

func (h *InvestmentHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	// don't need to check for "" or something similar as it was checked by investment_middleware
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)
	owned, err := h.portfolioService.CheckPortfolioOwnership(r.Context(), portfolioID, userID)
	if err != nil {
		http.Error(w, "Failed to check portfolio ownership", http.StatusInternalServerError)
		return
	}
	if !owned {
		http.Error(w, "Unauthorized access to portfolio", http.StatusUnauthorized)
		return
	}
	assetRequest := &createAssetRequest{}
	if err := json.NewDecoder(r.Body).Decode(assetRequest); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
	}

	if assetRequest.Name == "" || assetRequest.Ticker == "" || assetRequest.AssetTypeID == 0 {
		h.respondError(w, http.StatusBadRequest, "Name, Ticker, and AssetTypeID are required fields")
		return
	}

	// Validate AssetTypeID using cached asset types
	isValid := h.assetService.IsValidAssetType(assetRequest.AssetTypeID)
	if !isValid {
		h.respondError(w, http.StatusBadRequest, "Invalid asset type")
		return
	}

	// Check if the asset already exists in the portfolio
	exists, err := h.assetService.DoesAssetExist(r.Context(), portfolioID, assetRequest.Name)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to check if asset exists")
		return
	}
	if exists {
		h.respondError(w, http.StatusConflict, "Asset already exists in this portfolio")
		return
	}

	asset := assets.Asset{
		ID:          uuid.New(),
		PortfolioID: portfolioID,
		Name:        assetRequest.Name,
		Ticker:      assetRequest.Ticker,
	}

	switch assetRequest.AssetTypeID {
	case 1: // Stock
		if assetRequest.DividendYield == nil {
			h.respondError(w, http.StatusBadRequest, "DividendYield is required for stocks")
			return
		}
		asset.DividendYield = *assetRequest.DividendYield
		asset.CouponRate = 0       // not applicable for stocks
		asset.MaturityDate = nil   // not applicable for stocks
		asset.FaceValue = 0        // not applicable for stocks
		asset.Accumulation = false // not applicable for stocks

	case 2: // Bond
		if assetRequest.CouponRate == nil || assetRequest.MaturityDate == nil || assetRequest.FaceValue == nil {
			h.respondError(w, http.StatusBadRequest, "CouponRate, MaturityDate, and FaceValue are required for bonds")
			return
		}

		maturityDate, err := time.Parse("2006-01-02", *assetRequest.MaturityDate) // Example format: YYYY-MM-DD
		if err != nil {
			h.respondError(w, http.StatusBadRequest, "Invalid MaturityDate format, expected YYYY-MM-DD")
			return
		}
		asset.CouponRate = *assetRequest.CouponRate
		asset.MaturityDate = &maturityDate
		asset.FaceValue = *assetRequest.FaceValue
		asset.DividendYield = 0    // not applicable for bonds
		asset.Accumulation = false // not applicable for bonds

	case 3: // ETF
		if assetRequest.Accumulation == nil {
			h.respondError(w, http.StatusBadRequest, "Accumulation status is required for ETFs")
			return
		}
		asset.Accumulation = *assetRequest.Accumulation
		asset.CouponRate = 0     // not applicable for ETFs
		asset.MaturityDate = nil // not applicable for ETFs
		asset.FaceValue = 0      // not applicable for ETFs
		asset.DividendYield = 0  // not applicable for ETFs

	case 4: // Cryptocurrency
		// No specific fields required, so leave everything else as 0 or nil
		asset.CouponRate = 0
		asset.MaturityDate = nil
		asset.FaceValue = 0
		asset.DividendYield = 0
		asset.Accumulation = false

	case 5: // Savings Accounts
		// No specific fields required, similar to Cryptocurrency
		asset.CouponRate = 0
		asset.MaturityDate = nil
		asset.FaceValue = 0
		asset.DividendYield = 0
		asset.Accumulation = false

	case 6: // Cash
		// No specific fields required, similar to Cryptocurrency and Savings Accounts
		asset.CouponRate = 0
		asset.MaturityDate = nil
		asset.FaceValue = 0
		asset.DividendYield = 0
		asset.Accumulation = false

	default:
		h.respondError(w, http.StatusBadRequest, "Unsupported asset type")
		return
	}

	asset.AssetTypeID = assetRequest.AssetTypeID
	if err := h.assetService.CreateAsset(r.Context(), &asset); err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to create asset")
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data":   asset,
	})
}

func (h *InvestmentHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	// Get portfolioID and assetID from context
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)
	assetID := r.Context().Value("assetID").(uuid.UUID)

	// Single query to check if the asset belongs to the portfolio and if the user owns the portfolio
	exists, err := h.assetService.DoesAssetBelongToUser(r.Context(), assetID, portfolioID, userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to check asset and portfolio ownership")
		return
	}
	if !exists {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized access or asset not found in portfolio")
		return
	}

	// Proceed to delete the asset
	if err := h.assetService.DeleteAsset(r.Context(), assetID); err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to delete asset")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Asset deleted successfully.",
	})
}

func (h *InvestmentHandler) GetAllAssets(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)
	assetList, err := h.assetService.GetAllAssets(r.Context(), userID, portfolioID)

	if err != nil {
		fmt.Println("err", err.Error())
		if errors.Is(err, assets.ErrAssetsNotFound) {
			h.respondError(w, http.StatusNotFound, "No assets in this portfolio")
			return
		}
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve assets list")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "List of assets retrieved successfully.",
		"data":    assetList,
	})

}

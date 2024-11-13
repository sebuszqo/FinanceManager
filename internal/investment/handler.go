package investments

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	assets "github.com/sebuszqo/FinanceManager/internal/investment/asset"
	"github.com/sebuszqo/FinanceManager/internal/investment/models"
	portfolios "github.com/sebuszqo/FinanceManager/internal/investment/portfolio"
	transactions "github.com/sebuszqo/FinanceManager/internal/investment/transaction"
	"net/http"
	"time"
)

// I know that this handler should be divided into smaller ones in each file like it's done in "instrument" --> I will refactor it in the future

type InvestmentHandler struct {
	portfolioService   portfolios.Service
	assetService       assets.Service
	transactionService transactions.Service
	respondJSON        func(w http.ResponseWriter, status int, payload interface{})
	respondError       func(w http.ResponseWriter, status int, message string, errors ...[]string)
}

func NewInvestmentHandler(
	portfolioService portfolios.Service,
	assetsService assets.Service,
	transactionsService transactions.Service,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string, errors ...[]string),
) *InvestmentHandler {
	return &InvestmentHandler{
		portfolioService:   portfolioService,
		transactionService: transactionsService,
		assetService:       assetsService,
		respondJSON:        respondJSON,
		respondError:       respondError,
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
	Currency    string `json:"currency"`
	// Add other optional fields (for bonds, stocks, ETFs, etc.)
	CouponRate    *float64 `json:"coupon_rate,omitempty"`
	MaturityDate  *string  `json:"maturity_date,omitempty"`
	FaceValue     *float64 `json:"face_value,omitempty"`
	DividendYield *float64 `json:"dividend_yield,omitempty"`
	Accumulation  *bool    `json:"accumulation,omitempty"`
	Exchange      *string  `json:"exchange,omitempty"`
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
	userID := h.getUserIDReq(w, r)
	if userID == "" {
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
		h.respondError(w, http.StatusInternalServerError, "Failed to check portfolio ownership")
		return
	}
	if !owned {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized access to portfolio")
		return
	}
	assetRequest := &createAssetRequest{}
	if err := json.NewDecoder(r.Body).Decode(assetRequest); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if assetRequest.Name == "" || assetRequest.Ticker == "" || assetRequest.AssetTypeID == 0 || assetRequest.Currency == "" {
		h.respondError(w, http.StatusBadRequest, "Name, Ticker, Currency and AssetTypeID are required fields")
		return
	}

	// Validate AssetTypeID using cached asset types
	isValid := h.assetService.IsValidAssetType(assetRequest.AssetTypeID)
	if !isValid {
		h.respondError(w, http.StatusBadRequest, "Invalid asset type")
		return
	}

	// Check if the asset already exists in the portfolio
	exists, err := h.assetService.DoesAssetExist(r.Context(), portfolioID, assetRequest.Name, assetRequest.Ticker)
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
		Currency:    assetRequest.Currency,
	}

	switch assetRequest.AssetTypeID {
	case 1: // Stock
		if assetRequest.DividendYield == nil {
			h.respondError(w, http.StatusBadRequest, "DividendYield is required for stocks")
			return
		}
		if assetRequest.Exchange == nil {
			h.respondError(w, http.StatusBadRequest, "Exchange is required for stocks")
			return
		}
		asset.DividendYield = *assetRequest.DividendYield
		asset.Exchange = *assetRequest.Exchange
		asset.CouponRate = 0       // not applicable for stocks
		asset.MaturityDate = nil   // not applicable for stocks
		asset.FaceValue = 0        // not applicable for stocks
		asset.Accumulation = false // not applicable for stocks
		asset.InterestAccrued = 0

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
		asset.InterestAccrued = 0

	case 3: // ETF
		if assetRequest.Accumulation == nil {
			h.respondError(w, http.StatusBadRequest, "Accumulation status is required for ETFs")
			return
		}
		if assetRequest.Exchange == nil {
			h.respondError(w, http.StatusBadRequest, "Exchange is required for ETFs")
			return
		}
		asset.Accumulation = *assetRequest.Accumulation
		asset.Exchange = *assetRequest.Exchange
		asset.CouponRate = 0     // not applicable for ETFs
		asset.MaturityDate = nil // not applicable for ETFs
		asset.FaceValue = 0      // not applicable for ETFs
		asset.DividendYield = 0  // not applicable for ETFs
		asset.InterestAccrued = 0

	case 4: // Cryptocurrency
		// No specific fields required, so leave everything else as 0 or nil
		asset.CouponRate = 0
		asset.MaturityDate = nil
		asset.FaceValue = 0
		asset.DividendYield = 0
		asset.Accumulation = false
		asset.InterestAccrued = 0
	case 5: // Savings Accounts
		// No specific fields required, similar to Cryptocurrency
		asset.CouponRate = 0
		asset.MaturityDate = nil
		asset.FaceValue = 0
		asset.DividendYield = 0
		asset.Accumulation = false
		asset.InterestAccrued = 0
	case 6: // Cash
		// No specific fields required, similar to Cryptocurrency and Savings Accounts
		asset.CouponRate = 0
		asset.MaturityDate = nil
		asset.FaceValue = 0
		asset.DividendYield = 0
		asset.Accumulation = false
		asset.InterestAccrued = 0
	default:
		h.respondError(w, http.StatusBadRequest, "Unsupported asset type")
		return
	}

	asset.AssetTypeID = assetRequest.AssetTypeID
	asset.CreatedAt = time.Now()
	asset.UpdatedAt = time.Now()
	if err := h.assetService.CreateAsset(r.Context(), &asset); err != nil {
		if errors.Is(err, assets.ErrNotValidTicker) {
			h.respondError(w, http.StatusBadRequest, "Ticker, currency or exchange of the asset is not valid")
			return
		}
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
	owned, err := h.assetService.CheckAssetOwnership(r.Context(), assetID, portfolioID, userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to check asset and portfolio ownership")
		return
	}
	if !owned {
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

	owned, err := h.portfolioService.CheckPortfolioOwnership(r.Context(), portfolioID, userID)
	if err != nil {
		http.Error(w, "Failed to check portfolio ownership", http.StatusInternalServerError)
		return
	}
	if !owned {
		http.Error(w, "Unauthorized access to portfolio", http.StatusUnauthorized)
		return
	}
	assetList, err := h.assetService.GetAllAssets(r.Context(), portfolioID)

	if err != nil {
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

// Transaction handler
type createTransactionRequest struct {
	TransactionTypeID int      `json:"transaction_type_id"`
	Quantity          float64  `json:"quantity"`
	Price             float64  `json:"price"`
	TransactionDate   string   `json:"transaction_date"` // Could be ISO8601 format
	DividendAmount    *float64 `json:"dividend_amount,omitempty"`
	CouponAmount      *float64 `json:"coupon_amount,omitempty"`
}

func (h *InvestmentHandler) GetTransactionTypes(w http.ResponseWriter, r *http.Request) {
	// Retrieve the transaction types from the service
	transactionTypes := h.transactionService.GetTransactionTypes()
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   transactionTypes,
	})
}

func (h *InvestmentHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)
	assetID := r.Context().Value("assetID").(uuid.UUID)

	owned, err := h.assetService.CheckAssetOwnership(r.Context(), assetID, portfolioID, userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to check asset and portfolio ownership")
		return
	}
	if !owned {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized access or asset not found in portfolio")
		return
	}

	asset, err := h.assetService.GetAssetByID(r.Context(), assetID)
	if err != nil {
		if errors.Is(err, assets.ErrAssetNotFound) {
			h.respondError(w, http.StatusNotFound, "Asset doesn't exist")
			return
		}
		fmt.Println(err.Error())
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve asset")
		return
	}

	var req createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	transactionDateStr := req.TransactionDate
	if len(transactionDateStr) == 10 {
		transactionDateStr = transactionDateStr + "T00:00:00Z"
	}

	// Convert transaction date from string to time.Time
	transactionDate, err := time.Parse(time.RFC3339, transactionDateStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid transaction date format")
		return
	}

	assetTypeName := h.assetService.GetAssetTypeName(asset.AssetTypeID)
	err = h.validateTransactionForAssetType(assetTypeName, req)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	transaction := &models.Transaction{
		ID:                uuid.New(),
		AssetID:           assetID,
		TransactionTypeID: req.TransactionTypeID,
		Quantity:          req.Quantity,
		Price:             req.Price,
		TransactionDate:   transactionDate,
		DividendAmount:    req.DividendAmount,
		CouponAmount:      req.CouponAmount,
		CreatedAt:         time.Now(),
	}

	err = h.transactionService.CreateTransaction(r.Context(), assetID, userID, transaction)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to create transaction")
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data":   transaction,
	})
}

func (h *InvestmentHandler) validateTransactionForAssetType(assetTypeName string, req createTransactionRequest) error {
	switch assetTypeName {
	case "Stock":
		return h.validateStockTransaction(req)
	case "Bond":
		return h.validateBondTransaction(req)
	case "ETF":
		return h.validateETFTransaction(req)
	case "Cryptocurrency":
		return h.validateCryptocurrencyTransaction(req)
	case "Savings Accounts":
		return h.validateSavingsAccountTransaction(req)
	case "Cash":
		return h.validateCashTransaction(req)
	default:
		return fmt.Errorf("unsupported asset type: %s", assetTypeName)
	}
}

func (h *InvestmentHandler) validateStockTransaction(req createTransactionRequest) error {
	switch req.TransactionTypeID {
	case 1: // Buy
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and price must be greater than 0 for stock Buy transactions")
		}
	case 2: // Sell
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and price must be greater than 0 for stock Sell transactions")
		}
	case 3: // Dividend
		if req.DividendAmount == nil || *req.DividendAmount <= 0 {
			return fmt.Errorf("dividendAmount must be greater than 0 for stock Dividend transactions")
		}
	default:
		return fmt.Errorf("unsupported transaction type for stock")
	}
	return nil
}

func (h *InvestmentHandler) validateBondTransaction(req createTransactionRequest) error {
	switch req.TransactionTypeID {
	case 1: // Buy
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and Price must be greater than 0 for bond Buy transactions")
		}
	case 2: // Sell
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and Price must be greater than 0 for bond Sell transactions")
		}
	case 4: // Coupon Payment
		if req.CouponAmount == nil || *req.CouponAmount <= 0 {
			return fmt.Errorf("couponAmount must be greater than 0 for bond Coupon Payment transactions")
		}
	default:
		return fmt.Errorf("unsupported transaction type for bond")
	}
	return nil
}

func (h *InvestmentHandler) validateETFTransaction(req createTransactionRequest) error {
	switch req.TransactionTypeID {
	case 1: // Buy
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and Price must be greater than 0 for ETF Buy transactions")
		}
	case 2: // Sell
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and Price must be greater than 0 for ETF Sell transactions")
		}
	default:
		return fmt.Errorf("unsupported transaction type for ETF")
	}
	return nil
}

func (h *InvestmentHandler) validateCryptocurrencyTransaction(req createTransactionRequest) error {
	switch req.TransactionTypeID {
	case 1: // Buy
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and Price must be greater than 0 for cryptocurrency Buy transactions")
		}
	case 2: // Sell
		if req.Quantity <= 0 || req.Price <= 0 {
			return fmt.Errorf("quantity and Price must be greater than 0 for cryptocurrency Sell transactions")
		}
	default:
		return fmt.Errorf("unsupported transaction type for cryptocurrency")
	}
	return nil
}

func (h *InvestmentHandler) validateSavingsAccountTransaction(req createTransactionRequest) error {
	switch req.TransactionTypeID {
	case 1: // Deposit
		if req.Quantity <= 0 {
			return fmt.Errorf("quantity must be greater than 0 for savings account Deposit transactions")
		}
	case 2: // Withdrawal
		if req.Quantity <= 0 {
			return fmt.Errorf("quantity must be greater than 0 for savings account Withdrawal transactions")
		}
	default:
		return fmt.Errorf("unsupported transaction type for savings account")
	}
	return nil
}

func (h *InvestmentHandler) validateCashTransaction(req createTransactionRequest) error {
	switch req.TransactionTypeID {
	case 1: // Deposit
		if req.Quantity <= 0 {
			return fmt.Errorf("quantity must be greater than 0 for cash Deposit transactions")
		}
	case 2: // Withdrawal
		if req.Quantity <= 0 {
			return fmt.Errorf("quantity must be greater than 0 for cash Withdrawal transactions")
		}
	default:
		return fmt.Errorf("unsupported transaction type for cash")
	}
	return nil
}

func (h *InvestmentHandler) GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserIDReq(w, r)
	if userID == "" {
		return
	}

	// Retrieve the portfolioID and assetID from the request context
	portfolioID := r.Context().Value("portfolioID").(uuid.UUID)
	assetID := r.Context().Value("assetID").(uuid.UUID)

	// Single query to check if the asset belongs to the portfolio and if the user owns the portfolio
	owned, err := h.assetService.CheckAssetOwnership(r.Context(), assetID, portfolioID, userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to check asset and portfolio ownership")
		return
	}
	if !owned {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized access or asset not found in portfolio")
		return
	}

	// Fetch allTransactions from the asset service
	allTransactions, err := h.transactionService.GetAllTransactions(r.Context(), assetID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve allTransactions")
		return
	}

	// Return the allTransactions as a JSON response
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   allTransactions,
	})
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/sebuszqo/FinanceManager/db"
	"github.com/sebuszqo/FinanceManager/internal/finance/application"
	"github.com/sebuszqo/FinanceManager/internal/finance/infrastructure"
	"github.com/sebuszqo/FinanceManager/internal/finance/interfaces"
	investments "github.com/sebuszqo/FinanceManager/internal/investment"
	assets "github.com/sebuszqo/FinanceManager/internal/investment/asset"
	"github.com/sebuszqo/FinanceManager/internal/investment/instrument"
	"github.com/sebuszqo/FinanceManager/internal/investment/marketdata"
	portfolios "github.com/sebuszqo/FinanceManager/internal/investment/portfolio"
	transactions "github.com/sebuszqo/FinanceManager/internal/investment/transaction"

	"github.com/sebuszqo/FinanceManager/internal/auth"
	emailService "github.com/sebuszqo/FinanceManager/internal/email"
	"github.com/sebuszqo/FinanceManager/internal/user"
	"log"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Message string `json:"message"`
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s in %v", r.URL.Path, time.Since(start))
	})
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(payload)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, message string, errors ...[]string) {
	payload := map[string]interface{}{
		"status":  "error",
		"message": message,
		"code":    status,
	}

	if len(errors) > 0 && len(errors[0]) > 0 {
		payload["errors"] = errors[0]
	}

	respondJSON(w, status, payload)
}

type Server struct {
	router                      *http.ServeMux
	authHandler                 *auth.Handler
	userHandler                 *user.Handler
	authService                 auth.Service
	userService                 user.Service
	instrumentHandler           instrument.Handler
	investmentsHandler          *investments.InvestmentHandler
	personalTransactionsHandler *interfaces.PersonalTransactionHandler
	financeCategoriesHandler    *interfaces.CategoryHandler
	financePaymentHandler       *interfaces.PaymentHandler
}

func NewServer(authHandler *auth.Handler, authService auth.Service, userHandler *user.Handler, investmentHandler *investments.InvestmentHandler, instrumentHandler instrument.Handler, personalTransactionsHandler *interfaces.PersonalTransactionHandler, financeCategoriesHandler *interfaces.CategoryHandler, financePaymentHandler *interfaces.PaymentHandler) *Server {
	return &Server{
		authHandler:                 authHandler,
		userHandler:                 userHandler,
		investmentsHandler:          investmentHandler,
		authService:                 authService,
		instrumentHandler:           instrumentHandler,
		personalTransactionsHandler: personalTransactionsHandler,
		financeCategoriesHandler:    financeCategoriesHandler,
		financePaymentHandler:       financePaymentHandler,
		router:                      http.NewServeMux(),
	}
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	err := json.NewEncoder(w).Encode(Response{Message: "Path not found"})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func checkConfiguration() error {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, continuing with system environment variables")
	}

	if os.Getenv("JWT_SECRET") == "" {
		return errors.New("no JWT_SECRET Provided")
	}
	return nil
}

func (s *Server) handleReady(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) RegisterRoutes() {
	// Public routes
	publicRoutes := http.NewServeMux()
	publicRoutes.Handle("POST /api/user/register", http.HandlerFunc(s.userHandler.HandleRegister))
	publicRoutes.Handle("POST /api/user/email/verify", http.HandlerFunc(s.userHandler.HandleVerifyEmail))
	publicRoutes.Handle("POST /api/auth/login", http.HandlerFunc(s.authHandler.HandleLogin))
	publicRoutes.Handle("POST /api/auth/logout", http.HandlerFunc(s.authHandler.HandleLogout))
	publicRoutes.Handle("POST /api/auth/2fa/verify", http.HandlerFunc(s.authHandler.HandleVerifyTwoFactor))
	publicRoutes.Handle("POST /api/auth/password-reset/request", http.HandlerFunc(s.authHandler.RequestPasswordResetHandler))
	publicRoutes.Handle("POST /api/auth/password-reset/confirm", http.HandlerFunc(s.authHandler.ResetPasswordHandler))
	publicRoutes.Handle("GET /api/ready", http.HandlerFunc(s.handleReady))

	// Protected routes (using JWT Access Token Middleware)
	protectedRoutes := http.NewServeMux()
	// email changes request and confirm endpoint
	protectedRoutes.Handle("POST /api/protected/user/email/change-request", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleRequestEmailChange)))
	protectedRoutes.Handle("POST /api/protected/user/email/change-confirm", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleConfirmEmailChange)))

	// get user data endpoint
	protectedRoutes.Handle("GET /api/protected/user/profile", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleGetUserProfile)))

	protectedRoutes.Handle("POST /api/protected/auth/2fa/register",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleRegisterTwoFactor)))

	protectedRoutes.Handle("POST /api/protected/auth/2fa/verify-registration",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleVerifyTwoFactorCode)))

	protectedRoutes.Handle("POST /api/protected/auth/2fa/request-email-code",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleRequestEmail2FACode)))

	protectedRoutes.Handle("DELETE /api/protected/auth/2fa/disable",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleDisableTwoFactor)))

	protectedRoutes.Handle("POST /api/protected/user/change-password",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleChangePassword)))

	// PORTFOLIOS API
	protectedRoutes.Handle("POST /api/protected/investments/portfolios",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.investmentsHandler.CreatePortfolio)))

	protectedRoutes.Handle("GET /api/protected/investments/portfolios/{portfolioID}",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.GetPortfolio), "portfolioID")))

	protectedRoutes.Handle("PUT /api/protected/investments/portfolios/{portfolioID}",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.UpdatePortfolio), "portfolioID")))

	protectedRoutes.Handle("DELETE /api/protected/investments/portfolios/{portfolioID}",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.DeletePortfolio), "portfolioID")))

	protectedRoutes.Handle("GET /api/protected/investments/portfolios",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.investmentsHandler.GetAllPortfolios)))

	// ASSET API
	protectedRoutes.Handle("GET /api/protected/investments/asset_types",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.investmentsHandler.GetAssetTypes)))

	protectedRoutes.Handle("DELETE /api/protected/investments/portfolios/{portfolioID}/assets/{assetID}",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.DeleteAsset), "portfolioID", "assetID")))

	protectedRoutes.Handle("POST /api/protected/investments/portfolios/{portfolioID}/assets",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.CreateAsset), "portfolioID")))

	protectedRoutes.Handle("GET /api/protected/investments/portfolios/{portfolioID}/assets",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.GetAllAssets), "portfolioID")))

	//"PUT /api/protected/portfolios/{portfolioID}/assets/{assetID}"

	// TRANSACTION API
	protectedRoutes.Handle("GET /api/protected/investments/transaction_types",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.investmentsHandler.GetTransactionTypes)))

	protectedRoutes.Handle("POST /api/protected/investments/portfolios/{portfolioID}/assets/{assetID}/transactions",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.CreateTransaction), "portfolioID", "assetID")))

	protectedRoutes.Handle("GET /api/protected/investments/portfolios/{portfolioID}/assets/{assetID}/transactions",
		s.authService.JWTAccessTokenMiddleware()(s.investmentsHandler.ValidateInvestmentPathParamsMiddleware(http.HandlerFunc(s.investmentsHandler.GetAllTransactions), "portfolioID", "assetID")))

	//GET /api/protected/portfolios/{portfolioID}/assets/{assetID}/transactions/{transactionID}
	//PUT	/api/protected/portfolios/{portfolioID}/assets/{assetID}/transactions/{transactionID}
	//DELETE	/api/protected/portfolios/{portfolioID}/assets/{assetID}/transactions/{transactionID}
	// INSTRUMENTS
	protectedRoutes.Handle("GET /api/protected/investments/instruments/search",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.instrumentHandler.SearchInstruments)))

	// FINANCE API
	protectedRoutes.Handle("POST /api/protected/finance/transactions",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.personalTransactionsHandler.CreateTransaction)))

	protectedRoutes.Handle("POST /api/protected/finance/transactions/bulk",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.personalTransactionsHandler.CreateTransactionsBulk)))

	protectedRoutes.Handle("GET /api/protected/finance/transactions/summary",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.personalTransactionsHandler.GetTransactionSummary)))

	protectedRoutes.Handle("GET /api/protected/finance/transactions/summary/categories",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.personalTransactionsHandler.GetTransactionSummaryByCategory)))

	protectedRoutes.Handle("GET /api/protected/finance/transactions",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.personalTransactionsHandler.GetUserTransactions)))

	protectedRoutes.Handle("GET /api/protected/finance/categories/predefined",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.financeCategoriesHandler.GetPredefinedCategories)))

	//protectedRoutes.Handle("GET /api/protected/finance/categories/user",
	//	s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.financeCategoriesHandler.GetUserCategories)))

	protectedRoutes.Handle("GET /api/protected/finance/payment/methods",
		s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.financePaymentHandler.GetPaymentMethods)))

	//protectedRoutes.Handle("GET /api/protected/finance/payment/sources",
	//	s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.financePaymentHandler.GetUserPaymentSources)))

	// Refresh token routes
	refreshTokenRoutes := http.NewServeMux()
	refreshTokenRoutes.Handle("PUT /api/auth/refresh/token", s.authService.JWTRefreshTokenMiddleware()(http.HandlerFunc(s.authHandler.RefreshAccessToken)))

	// Main router
	mainRouter := http.NewServeMux()

	// Combine public, protected, and refresh routes with distinct paths
	mainRouter.Handle("/api/", publicRoutes)
	mainRouter.Handle("/api/protected/", protectedRoutes)
	mainRouter.Handle("/api/auth/refresh/", refreshTokenRoutes)
	mainRouter.Handle("/", http.HandlerFunc(notFoundHandler))

	s.router = mainRouter
}

func main() {
	if err := checkConfiguration(); err != nil {
		log.Fatalf("Missing configuration, update to start server")
	}

	dbService, err := database.NewDBService()
	if err != nil {
		log.Fatalf("Could not initialize database: %v", err)
	}
	defer func() {
		if err := dbService.Close(); err != nil {
			log.Printf("Error while closing database: %v", err)
		}
	}()

	err = godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, continuing with system environment variables")
	}

	apiKey := os.Getenv("MARKET_DATA_API_KEY")
	marketDataService := marketdata.NewFMPClient(apiKey)

	authRepo := auth.NewUserRepository(dbService.DB)
	userRepo := user.NewUserRepository(dbService.DB)

	sessionManager := auth.NewSessionManager()
	jwtManager := auth.NewJWTManager()
	newEmailService := emailService.NewEmailService()
	authenticator := auth.Authenticator{}

	userService := user.NewUserService(userRepo, newEmailService)
	userHandler := user.NewHandler(userService)
	authService := auth.NewAuthService(authRepo, userService, sessionManager, jwtManager, newEmailService, authenticator)
	authHandler := auth.NewHandler(authService)

	instrumentRepo := instrument.NewInstrumentRepository(dbService.DB)
	instrumentService := instrument.NewInstrumentService(instrumentRepo, marketDataService)
	instrumentHandler := instrument.NewInstrumentHandler(
		instrumentService,
		respondJSON,
		respondError,
	)

	transactionRepo := transactions.NewTransactionRepository(dbService.DB)
	transactionService := transactions.NewTransactionService(transactionRepo)

	assetRepo := assets.NewAssetRepository(dbService.DB)
	assetService := assets.NewAssetService(assetRepo, transactionService, marketDataService, instrumentService)

	transactionService.SetAssetService(assetService)

	portfolioRepo := portfolios.NewPortfolioRepository(dbService.DB)
	portfolioService := portfolios.NewPortfolioService(portfolioRepo)

	investmentsHandler := investments.NewInvestmentHandler(portfolioService, assetService, transactionService, respondJSON, respondError)

	categoryRepository := infrastructure.NewCategoryRepository(dbService.DB)
	personalTransactionRepository := infrastructure.NewPersonalTransactionRepository(dbService.DB)

	categoryService := application.NewCategoryService(categoryRepository)

	financeCategoriesHandler := interfaces.NewCategoryHandler(categoryService, respondJSON, respondError)

	financePaymentRepository := infrastructure.NewPaymentRepository(dbService.DB)
	financePaymentService := application.NewPaymentService(financePaymentRepository)
	financePaymentHandler := interfaces.NewPaymentHandler(financePaymentService, respondJSON, respondError)

	personalTransactionService := application.NewPersonalTransactionService(personalTransactionRepository, categoryService, financePaymentService)
	personalTransactionHandler := interfaces.NewPersonalTransactionHandler(personalTransactionService, respondJSON, respondError)
	server := NewServer(authHandler, authService, userHandler, investmentsHandler, instrumentHandler, personalTransactionHandler, financeCategoriesHandler, financePaymentHandler)

	server.RegisterRoutes()

	needsUpdate, err := instrumentService.NeedsUpdate(context.Background())
	if err != nil {
		log.Fatalf("Error checking if update is needed: %v", err)
	}

	if needsUpdate {
		log.Println("Data is outdated or missing. Starting initial data import...")
		err = instrumentService.ImportInstruments(context.Background())
		if err != nil {
			log.Fatalf("Error updating instruments: %v", err)
		}
		log.Println("Initial data import completed.")
	} else {
		log.Println("Data is valid, skipping initial data import")
	}
	err = StartScheduler(instrumentService)
	if err != nil {
		log.Fatalf("Scheduler didn't start, stoping the app ...")
	}
	err = StartUpdateAssetScheduler(assetService)
	if err != nil {
		log.Fatalf("Scheduler didn't start, stoping the app ...")
	}
	loggingMiddleware := loggingMiddleware(http.HandlerFunc(server.router.ServeHTTP))
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      loggingMiddleware,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Server starting on port 8080...")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func StartUpdateAssetScheduler(assetService assets.Service) error {
	c := cron.New()
	// Schedule the job to run every 24 hour --> 0 5 0 * * *
	_, err := c.AddFunc("@every 5m", func() {
		err := assetService.UpdateAssetPricing(context.Background())
		if err != nil {
			log.Printf("Error updating asset pricing: %v", err)
		} else {
			log.Println("Assets prices updated successfully.")
		}
	})
	if err != nil {
		return err
	}
	c.Start()
	return nil
}

func StartScheduler(instrumentService instrument.Service) error {
	c := cron.New()
	// Schedule the job to run every 6 hours --> 0 0 */6 * * *
	_, err := c.AddFunc("@every 6h", func() {
		err := instrumentService.ImportInstruments(context.Background())
		if err != nil {
			log.Printf("Error updating instruments: %v", err)
		} else {
			log.Println("Instruments updated successfully.")
		}
	})
	if err != nil {
		return err
	}
	c.Start()
	return nil
}

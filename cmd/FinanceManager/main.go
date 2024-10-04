package main

import (
	"encoding/json"
	"errors"
	"github.com/joho/godotenv"

	"github.com/sebuszqo/FinanceManager/internal/auth"
	database "github.com/sebuszqo/FinanceManager/internal/db"
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

type Server struct {
	router      *http.ServeMux
	authHandler *auth.Handler
	userHandler *user.Handler
	authService auth.Service
	userService user.Service
}

func NewServer(authHandler *auth.Handler, authService auth.Service, userHandler *user.Handler) *Server {
	return &Server{
		authHandler: authHandler,
		userHandler: userHandler,
		authService: authService,
		router:      http.NewServeMux(),
	}
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(Response{Message: "Path not found"})
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
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}

func (s *Server) RegisterRoutes() {
	// Public routes
	publicRoutes := http.NewServeMux()
	publicRoutes.Handle("POST /api/register", http.HandlerFunc(s.userHandler.HandleRegister))
	publicRoutes.Handle("POST /api/email/verify", http.HandlerFunc(s.userHandler.HandleVerifyEmail))
	publicRoutes.Handle("POST /api/auth/login", http.HandlerFunc(s.authHandler.HandleLogin))
	publicRoutes.Handle("POST /api/auth/2fa/verify", http.HandlerFunc(s.authHandler.HandleVerifyTwoFactor))
	publicRoutes.Handle("POST /api/password-reset/request", http.HandlerFunc(s.authHandler.RequestPasswordResetHandler))
	publicRoutes.Handle("POST /api/password-reset/confirm", http.HandlerFunc(s.authHandler.ResetPasswordHandler))
	publicRoutes.Handle("GET /api/ready", http.HandlerFunc(s.handleReady))

	// Protected routes (using JWT Access Token Middleware)
	protectedRoutes := http.NewServeMux()
	// email changes request and confirm endpoint
	protectedRoutes.Handle("POST /api/protected/email/change-request", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleRequestEmailChange)))
	protectedRoutes.Handle("POST /api/protected/email/change-confirm", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleConfirmEmailChange)))

	// get user data endpoint
	protectedRoutes.Handle("GET /api/protected/profile", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleGetUserProfile)))

	protectedRoutes.Handle("POST /api/protected/2fa/register", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleRegisterTwoFactor)))
	protectedRoutes.Handle("POST /api/protected/2fa/verify-registration", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleVerifyTwoFactorCode)))
	protectedRoutes.Handle("POST /api/protected/2fa/request-email-code", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleRequestEmail2FACode)))
	protectedRoutes.Handle("POST /api/protected/2fa/disable", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.authHandler.HandleDisableTwoFactor)))
	protectedRoutes.Handle("POST /api/protected/change-password", s.authService.JWTAccessTokenMiddleware()(http.HandlerFunc(s.userHandler.HandleChangePassword)))

	// Refresh token routes
	refreshTokenRoutes := http.NewServeMux()
	refreshTokenRoutes.Handle("POST /api/refresh/token", s.authService.JWTRefreshTokenMiddleware()(http.HandlerFunc(s.authHandler.RefreshAccessToken)))

	// Main router
	mainRouter := http.NewServeMux()

	// Combine public, protected, and refresh routes with distinct paths
	mainRouter.Handle("/api/", publicRoutes)
	mainRouter.Handle("/api/protected/", protectedRoutes)
	mainRouter.Handle("/api/refresh/", refreshTokenRoutes)
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
	defer dbService.Close()

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

	server := NewServer(authHandler, authService, userHandler)

	server.RegisterRoutes()

	loggingMiddleware := loggingMiddleware(http.HandlerFunc(server.router.ServeHTTP))
	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", loggingMiddleware); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

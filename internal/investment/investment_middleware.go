package investments

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"
)

func (h *InvestmentHandler) ValidatePortfolioIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		portfolioID := r.PathValue("portfolioID")
		if portfolioID == "" {
			fmt.Println("[Investment_Middleware] Portfolio ID is empty")
			h.respondError(w, http.StatusBadRequest, "Portfolio ID is required")
			return
		}
		parsedPortfolioID, err := uuid.Parse(portfolioID)
		if err != nil {
			fmt.Println("[Investment_Middleware] Portfolio ID is invalid")
			h.respondError(w, http.StatusNotFound, "Portfolio not found")
			return
		}
		ctx := context.WithValue(r.Context(), "portfolioID", parsedPortfolioID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

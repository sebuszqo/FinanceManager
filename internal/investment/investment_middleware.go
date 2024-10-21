package investments

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func (h *InvestmentHandler) ValidateInvestmentPathParamsMiddleware(next http.Handler, params ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, param := range params {
			paramValue := r.PathValue(param)
			if paramValue == "" {
				fmt.Printf("[Investment_Middleware] %s is empty\n", param)
				h.respondError(w, http.StatusBadRequest, capitalizeFirstLetter(fmt.Sprintf("%s is required", param)))
				return
			}

			parsedUUID, err := uuid.Parse(paramValue)
			if err != nil {
				fmt.Printf("[Investment_Middleware] %s is invalid\n", param)
				switch param {
				case "portfolioID":
					h.respondError(w, http.StatusNotFound, "Portfolio not found")
					return
				case "assetID":
					h.respondError(w, http.StatusNotFound, "Asset not found")
					return
				case "transactionID":
					h.respondError(w, http.StatusNotFound, "Transaction not found")
					return
				default:
					http.Error(w, fmt.Sprintf("Invalid %s format", param), http.StatusBadRequest)
				}

			}
			r = r.WithContext(context.WithValue(r.Context(), param, parsedUUID))

		}
		next.ServeHTTP(w, r)
	})
}

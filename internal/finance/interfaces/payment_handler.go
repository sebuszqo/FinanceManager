package interfaces

import (
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"net/http"
)

type PaymentServiceInterface interface {
	ListPaymentMethods() ([]domain.PaymentMethod, error)
}

type PaymentHandler struct {
	service      PaymentServiceInterface
	respondJSON  func(w http.ResponseWriter, status int, payload interface{})
	respondError func(w http.ResponseWriter, status int, message string, errors ...[]string)
}

func NewPaymentHandler(
	service PaymentServiceInterface,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string, errors ...[]string),
) *PaymentHandler {
	if service == nil || respondJSON == nil || respondError == nil {
		panic("Service and response functions must not be nil")
	}
	return &PaymentHandler{
		service:      service,
		respondJSON:  respondJSON,
		respondError: respondError,
	}
}

func (h *PaymentHandler) GetPaymentMethods(w http.ResponseWriter, _ *http.Request) {
	methods, err := h.service.ListPaymentMethods()
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve payment methods")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Methods retrieved successfully.",
		"methods": methods,
	})
}

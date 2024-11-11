package interfaces

import (
	"github.com/sebuszqo/FinanceManager/internal/finance/domain"
	"net/http"
)

type CategoryServiceInterface interface {
	GetAllPredefinedCategories(categoryType string) ([]domain.PredefinedCategory, error)
}

type CategoryHandler struct {
	service      CategoryServiceInterface
	respondJSON  func(w http.ResponseWriter, status int, payload interface{})
	respondError func(w http.ResponseWriter, status int, message string)
}

func NewCategoryHandler(
	service CategoryServiceInterface,
	respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string),
) *CategoryHandler {
	if service == nil || respondJSON == nil || respondError == nil {
		panic("Service and response functions must not be nil")
	}
	return &CategoryHandler{
		service:      service,
		respondJSON:  respondJSON,
		respondError: respondError,
	}
}

func (h *CategoryHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categoryType := r.URL.Query().Get("type")
	if categoryType != "" && categoryType != "income" && categoryType != "expense" {
		h.respondError(w, http.StatusBadRequest, "Invalid category type")
		return
	}

	categories, err := h.service.GetAllPredefinedCategories(categoryType)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve categories")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "success",
		"message":    "Categories retrieved successfully.",
		"categories": categories,
	})
}

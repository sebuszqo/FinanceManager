package instrument

import (
	"net/http"
	"strconv"
)

type Handler interface {
	SearchInstruments(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	instrumentService Service
	respondJSON       func(w http.ResponseWriter, status int, payload interface{})
	respondError      func(w http.ResponseWriter, status int, message string)
}

func NewInstrumentHandler(instrumentService Service, respondJSON func(w http.ResponseWriter, status int, payload interface{}),
	respondError func(w http.ResponseWriter, status int, message string)) Handler {
	return &handler{
		instrumentService: instrumentService,
		respondJSON:       respondJSON,
		respondError:      respondError,
	}
}

func (h *handler) SearchInstruments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	assetTypeIDStr := r.URL.Query().Get("typeID")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	if query == "" || assetTypeIDStr == "" {
		http.Error(w, "Query parameters 'q' and 'typeID' are required", http.StatusBadRequest)
		return
	}

	assetTypeID, err := strconv.Atoi(assetTypeIDStr)
	if err != nil {
		http.Error(w, "Query parameter 'typeID' must be an integer", http.StatusBadRequest)
		return
	}

	if assetTypeID != 1 && assetTypeID != 2 {
		http.Error(w, "Query parameter 'typeID' can be 1 (stock) or 2 (bonds)", http.StatusBadRequest)
		return
	}

	instruments, err := h.instrumentService.SearchInstruments(r.Context(), query, assetTypeID, limit)
	if err != nil {
		http.Error(w, "Error searching instruments", http.StatusInternalServerError)
		return
	}
	if len(*instruments) == 0 {
		h.respondError(w, http.StatusNotFound, "Instrument not found")
	}

	w.Header().Set("Content-Type", "application/json")
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "List of instruments retrieved successfully.",
		"data":    instruments,
	})
}

package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"sally/server/internal/extract"
	"sally/server/internal/provider"
)

func NewExtractHandler(extractor provider.Extractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req extract.ExtractSpecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if !validExtractSpecRequest(req) {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		resp, err := extractor.Extract(r.Context(), req)
		if err != nil {
			writeProviderError(w, req.RequestID, extractor.Meta(), err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func validExtractSpecRequest(req extract.ExtractSpecRequest) bool {
	if req.RequestID == "" || req.SentAt == "" {
		return false
	}
	if req.Client.App == "" || req.Client.Version == "" {
		return false
	}
	if req.Page.Title == "" || req.Page.URL == "" || req.Page.VisibleText == "" {
		return false
	}
	if req.Page.StructuredData == nil || req.Page.PDFLinks == nil {
		return false
	}
	return true
}

func writeProviderError(w http.ResponseWriter, requestID string, meta extract.ResponseMeta, err error) {
	message := "Extraction provider failed."
	if err != nil {
		message = err.Error()
	}

	resp := extract.ExtractSpecResponse{
		RequestID: requestID,
		Status:    "error",
		Error: &extract.ErrorPayload{
			Code:    "PROVIDER_ERROR",
			Message: message,
		},
		Meta: meta,
	}

	status := http.StatusBadGateway
	if errors.Is(err, provider.ErrTimeout) {
		status = http.StatusGatewayTimeout
		resp.Error.Code = "MODEL_TIMEOUT"
		resp.Error.Message = "Extraction did not complete in time."
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

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

		log.Printf("[extract] %s: received page=%q", req.RequestID, req.Page.URL)
		start := time.Now()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, _ := w.(http.Flusher)
		sendEvent := func(event string, data []byte) {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
			if flusher != nil {
				flusher.Flush()
			}
		}

		var resp extract.ExtractSpecResponse
		var err error

		if se, ok := extractor.(provider.StreamingExtractor); ok {
			resp, err = se.ExtractStreaming(r.Context(), req, func(chunkCount int) {
				data, _ := json.Marshal(map[string]int{"tokens": chunkCount})
				sendEvent("progress", data)
			})
		} else {
			resp, err = extractor.Extract(r.Context(), req)
		}

		elapsed := time.Since(start)
		if err != nil {
			log.Printf("[extract] %s: failed after %dms: %v", req.RequestID, elapsed.Milliseconds(), err)
			data, _ := json.Marshal(buildErrorResponse(req.RequestID, extractor.Meta(), err))
			sendEvent("error", data)
			return
		}

		log.Printf("[extract] %s: ok in %dms", req.RequestID, elapsed.Milliseconds())
		data, _ := json.Marshal(resp)
		sendEvent("done", data)
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

func buildErrorResponse(requestID string, meta extract.ResponseMeta, err error) extract.ExtractSpecResponse {
	code := "PROVIDER_ERROR"
	message := "Extraction provider failed."
	if err != nil {
		message = err.Error()
	}
	if errors.Is(err, provider.ErrTimeout) {
		code = "MODEL_TIMEOUT"
		message = "Extraction did not complete in time."
	}
	return extract.ExtractSpecResponse{
		RequestID: requestID,
		Status:    "error",
		Error: &extract.ErrorPayload{
			Code:    code,
			Message: message,
		},
		Meta: meta,
	}
}

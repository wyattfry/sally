package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"sally/server/internal/extract"
	"sally/server/internal/provider"

	queries "sally/server/internal/db/generated"
)

type scheduleQuerier interface {
	GetSchedule(ctx context.Context, id string) (queries.Schedule, error)
	ListSchedulesByProject(ctx context.Context, projectID string) ([]queries.Schedule, error)
	ListScheduleItems(ctx context.Context, scheduleID string) ([]queries.ScheduleItem, error)
}

type extractionLogger interface {
	InsertExtractionLog(ctx context.Context, p queries.InsertExtractionLogParams) error
}

func NewExtractHandler(extractor provider.Extractor, q scheduleQuerier, logger extractionLogger) http.HandlerFunc {
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

		var computedNextCode string
		var selectedZones []string
		if q != nil && req.ScheduleID != "" {
			if schedule, err := q.GetSchedule(r.Context(), req.ScheduleID); err == nil {
				if items, err := q.ListScheduleItems(r.Context(), req.ScheduleID); err == nil {
					computedNextCode = nextCode(items, schedule.Name)
					selectedZones = zonesFromItems(items)
				}
				if allSchedules, err := q.ListSchedulesByProject(r.Context(), schedule.ProjectID); err == nil {
					summaries := make([]extract.ScheduleSummary, 0, len(allSchedules))
					for _, s := range allSchedules {
						items, _ := q.ListScheduleItems(r.Context(), s.ID)
						summaries = append(summaries, extract.ScheduleSummary{
							Name:       s.Name,
							IsSelected: s.ID == req.ScheduleID,
							Zones:      zonesFromItems(items),
						})
					}
					req.ProjectContext.Schedules = summaries
				}
				req.ProjectContext.KnownZones = selectedZones
			}
		}

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
			if logger != nil {
				meta := extractor.Meta()
				_ = logger.InsertExtractionLog(context.Background(), queries.InsertExtractionLogParams{
					RequestID:     req.RequestID,
					ScheduleID:    req.ScheduleID,
					Provider:      meta.Provider,
					Model:         meta.Model,
					PromptVersion: meta.PromptVersion,
					DurationMS:    int(elapsed.Milliseconds()),
					Success:       false,
					ErrorMessage:  err.Error(),
					PageURL:       req.Page.URL,
				})
			}

			data, _ := json.Marshal(buildErrorResponse(req.RequestID, extractor.Meta(), err))
			sendEvent("error", data)
			return
		}

		log.Printf("[extract] %s: ok in %dms prompt_tok=%d completion_tok=%d",
			req.RequestID, elapsed.Milliseconds(), resp.Meta.PromptTokens, resp.Meta.CompletionTokens)
		if logger != nil {
			meta := resp.Meta
			var missingCount int
			if resp.Analysis != nil {
				missingCount = len(resp.Analysis.MissingFields)
			}
			_ = logger.InsertExtractionLog(context.Background(), queries.InsertExtractionLogParams{
				RequestID:         req.RequestID,
				ScheduleID:        req.ScheduleID,
				Provider:          meta.Provider,
				Model:             meta.Model,
				PromptVersion:     meta.PromptVersion,
				DurationMS:        int(elapsed.Milliseconds()),
				Success:           true,
				PageURL:           req.Page.URL,
				PromptTokens:      meta.PromptTokens,
				CompletionTokens:  meta.CompletionTokens,
				MissingFieldCount: missingCount,
			})
		}
		resp.NextCode = computedNextCode
		resp.KnownZones = selectedZones
		data, _ := json.Marshal(resp)
		sendEvent("done", data)
	}
}

func zonesFromItems(items []queries.ScheduleItem) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		if it.Zone != "" && !seen[it.Zone] {
			seen[it.Zone] = true
			out = append(out, it.Zone)
		}
	}
	return out
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

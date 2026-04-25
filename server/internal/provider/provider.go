package provider

import (
	"context"
	"errors"
	"strings"

	"sally/server/internal/extract"
)

var (
	ErrTimeout = errors.New("provider timeout")
	ErrFailure = errors.New("provider failure")
)

type Extractor interface {
	Extract(ctx context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error)
	Meta() extract.ResponseMeta
}

// ProgressFunc receives the cumulative token chunk count as generation progresses.
type ProgressFunc func(chunkCount int)

// StreamingExtractor is an optional interface for providers that support token streaming.
// The HTTP handler detects this interface and uses SSE when available.
type StreamingExtractor interface {
	ExtractStreaming(ctx context.Context, req extract.ExtractSpecRequest, onProgress ProgressFunc) (extract.ExtractSpecResponse, error)
}

type StubExtractor struct{}

func NewStubExtractor() StubExtractor {
	return StubExtractor{}
}

func (s StubExtractor) Extract(_ context.Context, req extract.ExtractSpecRequest) (extract.ExtractSpecResponse, error) {
	return extract.ExtractSpecResponse{
		RequestID: req.RequestID,
		Status:    "ok",
		Proposal: &extract.Proposal{
			Title:               req.Page.Title,
			Manufacturer:        "Stub Manufacturer",
			ModelNumber:         "STUB-MODEL",
			Category:            firstOrDefault(req.ProjectContext.KnownCategories, "Accessory"),
			Description:         truncateDescription(req.Page.VisibleText),
			Finish:              "",
			FinishModelNumber:   "",
			AvailableFinishes:   []string{},
			FinishModelMappings: []extract.FinishModelMapping{},
			RequiredAddOns:      []string{},
			OptionalCompanions:  []string{},
			Zone:                firstOrDefault(req.ProjectContext.KnownZones, ""),
			SourceURL:           req.Page.URL,
			SourceTitle:         req.Page.Title,
			SourceImageURL:      req.Page.MainImageURL,
			SourcePDFLinks:      req.Page.PDFLinks,
		},
		Meta: s.Meta(),
	}, nil
}

func (StubExtractor) Meta() extract.ResponseMeta {
	return extract.ResponseMeta{
		Provider:      "stub",
		Model:         "stub-extractor",
		PromptVersion: "extract-spec-v1",
		DurationMS:    0,
	}
}

func firstOrDefault(values []string, fallback string) string {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}

func truncateDescription(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 240 {
		return trimmed
	}
	return trimmed[:240]
}

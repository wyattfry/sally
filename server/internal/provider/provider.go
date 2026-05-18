package provider

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"sally/server/internal/extract"
)

var (
	ErrTimeout    = errors.New("provider timeout")
	ErrFailure    = errors.New("provider failure")
	ErrOverloaded = errors.New("provider overloaded")
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
			Room:                firstOrDefault(req.ProjectContext.KnownRooms, ""),
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

func coalesceStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func coalesceFinishMappings(s []extract.FinishModelMapping) []extract.FinishModelMapping {
	if s == nil {
		return []extract.FinishModelMapping{}
	}
	return s
}

var xmlTagRe = regexp.MustCompile(`<[^>]+>`)

// sanitizeRoom strips XML/markup artifacts that models occasionally inject into
// plain-text fields (e.g. "</room><parameter ...>").
func sanitizeRoom(s string) string {
	s = xmlTagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

var placeholderScheduleRe = regexp.MustCompile(`^(?i)(new schedule|new note)( \d+)?$`)

// filterPromptableSchedules drops auto-named empty placeholder schedules
// ("New Schedule", "New Note 2", etc.) from the LLM's candidate list — they
// carry no semantic signal and bias the model toward force-fitting items onto
// them rather than proposing a real new name.
func filterPromptableSchedules(schedules []extract.ScheduleSummary) []extract.ScheduleSummary {
	out := make([]extract.ScheduleSummary, 0, len(schedules))
	for _, s := range schedules {
		if len(s.Rooms) == 0 && placeholderScheduleRe.MatchString(strings.TrimSpace(s.Name)) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// SnapSuggestedScheduleName canonicalizes the LLM's suggestedScheduleName to
// an existing schedule name when they refer to the same thing — exact match
// (case/whitespace-insensitive), or one normalized name contains the other as
// a whole-word substring after stripping common suffixes ("Schedule",
// "Schedules", trailing "s"). Returns (canonical, true) on match, ("", false)
// otherwise. Skips placeholder existing names so "New Schedule" doesn't snap
// onto random suggestions.
func SnapSuggestedScheduleName(suggested string, existing []string) (string, bool) {
	want := normalizeScheduleName(suggested)
	if want == "" {
		return "", false
	}
	// Exact normalized match wins.
	for _, name := range existing {
		if normalizeScheduleName(name) == want {
			return name, true
		}
	}
	// Whole-word substring either direction. Only consider non-placeholder
	// existing names so empty "New Schedule" placeholders don't capture.
	wantTokens := strings.Fields(want)
	for _, name := range existing {
		if placeholderScheduleRe.MatchString(strings.TrimSpace(name)) {
			continue
		}
		got := normalizeScheduleName(name)
		if got == "" {
			continue
		}
		gotTokens := strings.Fields(got)
		if containsAll(wantTokens, gotTokens) || containsAll(gotTokens, wantTokens) {
			return name, true
		}
	}
	return "", false
}

var scheduleSuffixRe = regexp.MustCompile(`(?i)\s+(schedule|schedules)\s*$`)

func normalizeScheduleName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = scheduleSuffixRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// containsAll reports whether every token in needles appears in haystack.
func containsAll(haystack, needles []string) bool {
	if len(needles) == 0 {
		return false
	}
	set := make(map[string]bool, len(haystack))
	for _, t := range haystack {
		set[t] = true
	}
	for _, n := range needles {
		if !set[n] {
			return false
		}
	}
	return true
}

const maxLogBytes = 64 * 1024 // 64 KB cap per prompt/response field

func capLog(s string) string {
	if len(s) > maxLogBytes {
		return s[:maxLogBytes]
	}
	return s
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

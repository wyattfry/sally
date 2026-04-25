package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"sally/server/internal/extract"
)

func handleExtractSpec(w http.ResponseWriter, r *http.Request) {
	var req extract.ExtractSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if !validExtractSpecRequest(req) {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	resp := extract.ExtractSpecResponse{
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
		Meta: extract.ResponseMeta{
			Provider:      "stub",
			Model:         "stub-extractor",
			PromptVersion: "extract-spec-v1",
			DurationMS:    0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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

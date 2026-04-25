package extract

import "encoding/json"

type ExtractSpecRequest struct {
	RequestID      string             `json:"requestId"`
	SentAt         string             `json:"sentAt"`
	Client         ClientInfo         `json:"client"`
	Page           PagePayload        `json:"page"`
	ProjectContext ProjectContext     `json:"projectContext"`
	Options        ExtractSpecOptions `json:"options"`
}

type ClientInfo struct {
	App           string `json:"app"`
	Version       string `json:"version"`
	ChromeVersion string `json:"chromeVersion"`
}

type PagePayload struct {
	Title          string            `json:"title"`
	URL            string            `json:"url"`
	VisibleText    string            `json:"visibleText"`
	MainImageURL   string            `json:"mainImageUrl,omitempty"`
	StructuredData []json.RawMessage `json:"structuredData"`
	PDFLinks       []string          `json:"pdfLinks"`
}

type ProjectContext struct {
	ProjectName     string   `json:"projectName"`
	KnownZones      []string `json:"knownZones"`
	KnownCategories []string `json:"knownCategories"`
}

type ExtractSpecOptions struct {
	IncludeDebug       bool `json:"includeDebug"`
	ReturnAlternatives bool `json:"returnAlternatives"`
}

type ExtractSpecResponse struct {
	RequestID string        `json:"requestId"`
	Status    string        `json:"status"`
	Proposal  *Proposal     `json:"proposal,omitempty"`
	Analysis  *Analysis     `json:"analysis,omitempty"`
	Error     *ErrorPayload `json:"error,omitempty"`
	Meta      ResponseMeta  `json:"meta"`
}

type Proposal struct {
	Title               string               `json:"title"`
	Manufacturer        string               `json:"manufacturer"`
	ModelNumber         string               `json:"modelNumber"`
	Category            string               `json:"category"`
	Description         string               `json:"description"`
	Finish              string               `json:"finish"`
	FinishModelNumber   string               `json:"finishModelNumber"`
	AvailableFinishes   []string             `json:"availableFinishes"`
	FinishModelMappings []FinishModelMapping `json:"finishModelMappings"`
	RequiredAddOns      []string             `json:"requiredAddOns"`
	OptionalCompanions  []string             `json:"optionalCompanions"`
	Zone                string               `json:"zone"`
	SourceURL           string               `json:"sourceUrl"`
	SourceTitle         string               `json:"sourceTitle"`
	SourceImageURL      string               `json:"sourceImageUrl,omitempty"`
	SourcePDFLinks      []string             `json:"sourcePdfLinks"`
}

type FinishModelMapping struct {
	Finish      string `json:"finish"`
	ModelNumber string `json:"modelNumber"`
}

type Analysis struct {
	MissingFields []string   `json:"missingFields"`
	Warnings      []string   `json:"warnings"`
	Confidence    Confidence `json:"confidence"`
}

type Confidence struct {
	Overall        float64 `json:"overall"`
	Title          float64 `json:"title"`
	Manufacturer   float64 `json:"manufacturer"`
	ModelNumber    float64 `json:"modelNumber"`
	Category       float64 `json:"category"`
	Description    float64 `json:"description"`
	Finish         float64 `json:"finish"`
	RequiredAddOns float64 `json:"requiredAddOns"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseMeta struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	PromptVersion string `json:"promptVersion"`
	DurationMS    int    `json:"durationMs"`
}

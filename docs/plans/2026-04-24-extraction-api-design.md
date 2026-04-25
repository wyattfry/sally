# Sally Extraction API Design

## Goal

Replace the current mock extractor with a real backend call while preserving the existing extension UX:

- user clicks `SPEC`
- Sally reads the page
- Sally returns one editable proposal
- user remains responsible for review and approval

The first production-facing backend will be a small Go service in this repo, reachable from the extension, with a single synchronous extraction endpoint.

## Recommendation

Use a "thin behavior, rich metadata" API:

- one synchronous request
- one proposal response
- strict JSON contract
- extra metadata for evaluation, tracing, and provider swaps

This keeps the implementation small while preserving room for prompt/version/provider analysis later.

## Architecture

Initial request path:

```text
Chrome extension
  -> POST /v1/extract-spec
  -> Go backend
  -> hosted multimodal model
  -> strict JSON response
  -> Sally review panel
```

The extension should not call model providers directly. The backend owns:

- provider API keys
- prompt templates
- schema enforcement
- retries/timeouts
- logging
- later provider switching and evals

## Endpoint

- `POST /v1/extract-spec`

## Request Contract

```json
{
  "requestId": "uuid",
  "sentAt": "2026-04-24T18:30:00Z",
  "client": {
    "app": "sally-extension",
    "version": "0.1.0",
    "chromeVersion": "136.0.0.0"
  },
  "page": {
    "title": "Example Co. WF-200 Wall Faucet",
    "url": "https://example.com/products/wf-200",
    "visibleText": "....",
    "mainImageUrl": "https://example.com/faucet.jpg",
    "structuredData": [],
    "pdfLinks": [
      "https://example.com/spec-sheet.pdf"
    ]
  },
  "projectContext": {
    "projectName": "My New Project",
    "knownZones": ["Primary Bath", "Powder Room"],
    "knownCategories": [
      "Plumbing Fixture",
      "Lighting",
      "Appliance",
      "Hardware",
      "Finish",
      "Furniture",
      "Accessory"
    ]
  },
  "options": {
    "includeDebug": true,
    "returnAlternatives": false
  }
}
```

### Request Notes

- `page` mirrors the current extension-side `CapturedPage` shape.
- `projectContext` is lightweight grounding, not a mode system.
- `knownZones` and `knownCategories` help the model align proposals with the current project language.
- `requestId` allows end-to-end tracing across extension logs and backend logs.
- `options.includeDebug` allows the backend to include richer analysis metadata without changing the endpoint.

## Response Contract

```json
{
  "requestId": "uuid",
  "status": "ok",
  "proposal": {
    "title": "Wall Faucet",
    "manufacturer": "Example Co.",
    "modelNumber": "WF-200",
    "category": "Plumbing Fixture",
    "description": "Wall-mounted faucet with rough-in requirements and installation constraints noted from the page.",
    "finish": "Polished Chrome",
    "finishModelNumber": "",
    "availableFinishes": ["Polished Chrome", "Brushed Nickel"],
    "finishModelMappings": [
      {
        "finish": "Polished Chrome",
        "modelNumber": "WF-200-PC"
      },
      {
        "finish": "Brushed Nickel",
        "modelNumber": "WF-200-BN"
      }
    ],
    "requiredAddOns": ["Rough valve body"],
    "optionalCompanions": ["Drain assembly"],
    "zone": "",
    "sourceUrl": "https://example.com/products/wf-200",
    "sourceTitle": "Example Co. WF-200 Wall Faucet",
    "sourceImageUrl": "https://example.com/faucet.jpg",
    "sourcePdfLinks": [
      "https://example.com/spec-sheet.pdf"
    ]
  },
  "analysis": {
    "missingFields": [],
    "warnings": [
      "Finish-to-model mapping inferred from page copy, verify before approval."
    ],
    "confidence": {
      "overall": 0.84,
      "title": 0.96,
      "manufacturer": 0.94,
      "modelNumber": 0.92,
      "category": 0.78,
      "description": 0.73,
      "finish": 0.81,
      "requiredAddOns": 0.67
    }
  },
  "meta": {
    "provider": "openai",
    "model": "gpt-5-mini",
    "promptVersion": "extract-spec-v1",
    "durationMs": 1820
  }
}
```

### Response Notes

- `proposal` intentionally resembles the current editable `ScheduleItem`.
- The backend should not assign local persistence fields like `id` or `capturedAt`.
- `availableFinishes` and `finishModelMappings` are included now because they are part of the intended Sally extraction behavior, even if the UI does not use them immediately.
- `analysis` is optional for the UI but useful for evaluation and future product affordances.
- `meta` is part of the product's internal audit trail and should be preserved even if hidden from the user.

## Error Contract

```json
{
  "requestId": "uuid",
  "status": "error",
  "error": {
    "code": "MODEL_TIMEOUT",
    "message": "Extraction did not complete in time."
  },
  "meta": {
    "provider": "openai",
    "model": "gpt-5-mini",
    "promptVersion": "extract-spec-v1",
    "durationMs": 15000
  }
}
```

## Initial Field Rules

### Required request fields

- `requestId`
- `sentAt`
- `client`
- `page.title`
- `page.url`
- `page.visibleText`
- `page.structuredData`
- `page.pdfLinks`
- `projectContext`
- `options`

### Optional request fields

- `page.mainImageUrl`

### Required response fields on success

- `requestId`
- `status`
- `proposal`
- `meta`

### Optional response fields on success

- `analysis`

### Required response fields on error

- `requestId`
- `status`
- `error`
- `meta`

## Timeout And Failure Behavior

Recommended initial behavior:

- backend timeout: 15 seconds
- extension timeout: 18 seconds
- one request equals one proposal attempt

For local development only, the extension may fall back to the current mock extractor if the backend is unavailable. Tester-facing builds should not silently fall back to mock extraction.

## Logging And Evaluation

The backend should log enough metadata to support future evals:

- `requestId`
- request timestamp
- provider
- model
- prompt version
- duration
- success/error code

Later, when persistent storage exists, Sally should also preserve:

- raw captured page payload
- backend proposal
- user-edited accepted result

That comparison set will be the basis for prompt tuning and provider evaluation.

## Go Boundary

The initial Go service should remain narrow:

- one HTTP server
- one extraction handler
- one provider adapter
- environment-based config

Suggested first internal packages:

```text
server/
  cmd/sally-server/
  internal/config/
  internal/httpapi/
  internal/extract/
  internal/provider/
```

## Testing Strategy

Start with:

- JSON contract tests for request/response encoding
- handler tests for success, timeout, and provider failure
- provider adapter tests with mocked upstream responses
- extension integration tests against a stub backend response

Do not start with persistence. The first milestone is only:

- extension sends captured page to backend
- backend returns one real model proposal
- user reviews and approves it in the existing panel

## Recommendation Summary

Build the first real Sally backend as a Go service with a single `POST /v1/extract-spec` endpoint, rich tracing metadata, and a strict proposal schema that maps closely to the current extension UI.

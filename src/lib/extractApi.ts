import type {
  CapturedPage,
  ColumnDefinition,
  ExtractSpecRequest,
  ExtractSpecResponse,
  FinishModelMapping,
  ScheduleColumn,
  ScheduleItem
} from "./types";

const DEFAULT_BACKEND_BASE_URL = "http://10.0.0.104:8080";
export const EXTRACT_TIMEOUT_MS = 180_000;
const EXTRACT_PATH = "/api/v1/extract-spec";

type ExtractionErrorKind = "transport" | "backend" | "invalid";

type SallyRuntimeConfig = {
  backendBaseUrl?: string;
  allowMockFallback?: boolean;
  developmentMode?: boolean;
};

export class ExtractionError extends Error {
  kind: ExtractionErrorKind;

  constructor(kind: ExtractionErrorKind, message: string) {
    super(message);
    this.name = "ExtractionError";
    this.kind = kind;
  }
}

type ExtractScheduleItemArgs = {
  capturedPage: CapturedPage;
  knownCategories: string[];
  knownRooms?: string[];
  knownScheduleNames?: string[];
  columns?: ScheduleColumn[];
  scheduleId?: string;
  now?: Date;
  onProgress?: (tokenCount: number) => void;
};

type ExtractScheduleItemResult = {
  item: ScheduleItem;
  suggestedScheduleName?: string;
  knownRooms: string[];
  // Per-spec-session UI hints. Not persisted on the item; used by the panel
  // to render a finish combobox that swaps the model number when the user
  // picks a different available finish. Discarded on save.
  availableFinishes?: string[];
  finishModelMappings?: FinishModelMapping[];
};

export async function extractScheduleItem({
  capturedPage,
  knownCategories,
  knownRooms = [],
  knownScheduleNames = [],
  columns = [],
  scheduleId,
  now = new Date(),
  onProgress
}: ExtractScheduleItemArgs): Promise<ExtractScheduleItemResult> {
  const request = buildExtractSpecRequest({
    capturedPage,
    knownCategories,
    knownRooms,
    knownScheduleNames,
    columns,
    scheduleId,
    now
  });

  const apiUrl = getExtractApiUrl();
  console.debug("Sending extraction request to", apiUrl, "with payload", request);

  if (typeof chrome !== "undefined" && chrome.runtime && chrome.runtime.connect) {
    return new Promise((resolve, reject) => {
      const port = chrome.runtime.connect({ name: "extract-spec" });
      let resolved = false;
      let timeoutId: number;

      const cleanup = () => {
        window.clearTimeout(timeoutId);
        if (!resolved) {
          port.disconnect();
        }
      };

      port.onMessage.addListener((msg) => {
        if (msg.type === "SSE_EVENT") {
          const { event, data } = msg;
          if (event === "progress") {
            const parsed = JSON.parse(data) as { tokens: number };
            onProgress?.(parsed.tokens);
          } else if (event === "done") {
            const payload = JSON.parse(data) as ExtractSpecResponse;
            if (!payload.proposal) {
              cleanup();
              reject(new ExtractionError("invalid", "Extraction response was missing a proposal."));
              return;
            }
            resolved = true;
            cleanup();
            resolve(toExtractResult(payload, now));
          } else if (event === "error") {
            const payload = JSON.parse(data) as ExtractSpecResponse;
            cleanup();
            reject(new ExtractionError("backend", payload?.error?.message || "Extraction request failed."));
          }
        } else if (msg.type === "ERROR") {
          cleanup();
          reject(new ExtractionError(msg.kind, msg.message));
        } else if (msg.type === "DONE") {
          if (!resolved) {
            cleanup();
            reject(new ExtractionError("invalid", "Stream ended without a completion event."));
          }
        }
      });

      port.onDisconnect.addListener(() => {
        if (!resolved) {
          cleanup();
          reject(new ExtractionError("transport", "Connection to background script closed unexpectedly."));
        }
      });

      port.postMessage({
        type: "START_EXTRACTION",
        apiUrl,
        request
      });

      timeoutId = window.setTimeout(() => {
        cleanup();
        reject(new ExtractionError("transport", "Extraction request timed out."));
      }, EXTRACT_TIMEOUT_MS);
    });
  }

  const controller = new AbortController();
  const timeoutId = window.setTimeout(() => controller.abort(), EXTRACT_TIMEOUT_MS);

  try {
    const response = await fetch(apiUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
      signal: controller.signal
    });

    if (!response.ok) {
      const text = await response.text();
      throw new ExtractionError("backend", text.trim() || "Extraction request failed.");
    }

    if (!response.body) {
      throw new ExtractionError("invalid", "Empty response from extraction server.");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      const { events, remaining } = parseSSEBuffer(buffer);
      buffer = remaining;

      for (const { event, data } of events) {
        if (event === "progress") {
          const parsed = JSON.parse(data) as { tokens: number };
          onProgress?.(parsed.tokens);
        } else if (event === "done") {
          const payload = JSON.parse(data) as ExtractSpecResponse;
          if (!payload.proposal) {
            throw new ExtractionError("invalid", "Extraction response was missing a proposal.");
          }
          return toExtractResult(payload, now);
        } else if (event === "error") {
          const payload = JSON.parse(data) as ExtractSpecResponse;
          throw new ExtractionError("backend", payload?.error?.message || "Extraction request failed.");
        }
      }
    }

    throw new ExtractionError("invalid", "Stream ended without a completion event.");
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new ExtractionError("transport", "Extraction request timed out.");
    }
    if (error instanceof ExtractionError) {
      throw error;
    }
    if (error instanceof TypeError) {
      throw new ExtractionError("transport", "Extraction backend is unreachable.");
    }
    throw error;
  } finally {
    window.clearTimeout(timeoutId);
  }
}

function parseSSEBuffer(buffer: string): {
  events: Array<{ event: string; data: string }>;
  remaining: string;
} {
  const events: Array<{ event: string; data: string }> = [];
  const parts = buffer.split("\n\n");
  const remaining = parts.pop() ?? "";

  for (const part of parts) {
    if (!part.trim()) continue;
    let event = "message";
    let data = "";
    for (const line of part.split("\n")) {
      if (line.startsWith("event: ")) event = line.slice(7).trim();
      else if (line.startsWith("data: ")) data = line.slice(6);
    }
    if (data) events.push({ event, data });
  }

  return { events, remaining };
}

function getExtractApiUrl(): string {
  return `${getBackendBaseUrl()}${EXTRACT_PATH}`;
}

export function shouldAllowMockFallback(): boolean {
  const config = getRuntimeConfig();
  return Boolean(config.developmentMode && config.allowMockFallback);
}

export function shouldFallbackToMock(error: unknown): boolean {
  return error instanceof ExtractionError && error.kind === "transport";
}

function getBackendBaseUrl(): string {
  const config = getRuntimeConfig();
  return (config.backendBaseUrl || DEFAULT_BACKEND_BASE_URL).replace(/\/+$/, "");
}

function getRuntimeConfig(): Required<SallyRuntimeConfig> {
  const config = (globalThis as { __SALLY_CONFIG__?: SallyRuntimeConfig }).__SALLY_CONFIG__;
  return {
    backendBaseUrl:
      config?.backendBaseUrl ||
      import.meta.env.VITE_SALLY_BACKEND_BASE_URL ||
      DEFAULT_BACKEND_BASE_URL,
    allowMockFallback:
      config?.allowMockFallback ?? import.meta.env.VITE_SALLY_ALLOW_MOCK_FALLBACK === "true",
    developmentMode: config?.developmentMode ?? import.meta.env.DEV
  };
}


function buildExtractSpecRequest({
  capturedPage,
  knownCategories,
  knownRooms,
  knownScheduleNames,
  columns,
  scheduleId,
  now
}: Required<Omit<ExtractScheduleItemArgs, "onProgress" | "scheduleId">> & { scheduleId?: string; now: Date }): ExtractSpecRequest {
  const customColumns: ColumnDefinition[] = columns
    .filter((c) => c.key !== "room" && c.key !== "code")
    .map((c) => ({ key: c.key, label: c.label }));

  return {
    requestId: createId(),
    sentAt: now.toISOString(),
    client: {
      app: "sally-extension",
      version: "0.1.0",
      chromeVersion: globalThis.navigator?.userAgent.match(/Chrome\/([0-9.]+)/)?.[1] || ""
    },
    page: capturedPage,
    projectContext: {
      projectName: "",
      knownCategories,
      knownRooms,
      knownScheduleNames
    },
    scheduleId,
    customColumns: customColumns.length > 0 ? customColumns : undefined,
    options: {
      includeDebug: true,
      returnAlternatives: false
    }
  };
}

function clean(v: string | undefined | null): string {
  if (!v) return "";
  const t = v.trim();
  return /^<[^>]+>$/.test(t) ? "" : t;
}

function toExtractResult(response: ExtractSpecResponse, now: Date): ExtractScheduleItemResult {
  const proposal = response.proposal;
  if (!proposal) {
    throw new ExtractionError("invalid", "Extraction response was missing a proposal.");
  }

  const data: Record<string, string> = {};
  if (response.nextCode) data.code = response.nextCode;
  const title = clean(proposal.title); if (title) data.title = title;
  const manufacturer = clean(proposal.manufacturer); if (manufacturer) data.manufacturer = manufacturer;
  const baseModel = clean(proposal.modelNumber);
  const finish = clean(proposal.finish);
  const finishModelNumber = clean(proposal.finishModelNumber);
  const mappings = (proposal.finishModelMappings ?? []).filter(m => m.finish && m.modelNumber);
  // Prefer the finish-specific SKU for the Model column when we have one,
  // so users see the orderable variant out of the gate. Fall back to the
  // base model otherwise.
  const matchedMapping = finish ? mappings.find(m => m.finish === finish) : undefined;
  const initialModel = matchedMapping?.modelNumber || finishModelNumber || baseModel;
  if (initialModel) data.model_number = initialModel;
  const description = clean(proposal.description); if (description) data.description = description;
  if (finish) data.finish = finish;
  if (finishModelNumber) data.finish_model_number = finishModelNumber;
  const notes = [...(proposal.requiredAddOns ?? []), ...(proposal.optionalCompanions ?? [])]
    .map(clean).filter(Boolean).join("; ");
  if (notes) data.notes = notes;
  if (proposal.customFields) {
    for (const [key, value] of Object.entries(proposal.customFields)) {
      const v = clean(value); if (v) data[key] = v;
    }
  }

  const room = clean(proposal.room);
  return {
    item: {
      id: `draft-${response.requestId}`,
      capturedAt: now.toISOString(),
      room: room || undefined,
      data,
      sourceUrl: proposal.sourceUrl,
      sourceTitle: proposal.sourceTitle,
      sourceImageUrl: proposal.sourceImageUrl,
      sourceImageUrls: proposal.sourceImageUrls ?? [],
      sourcePdfLinks: proposal.sourcePdfLinks ?? []
    },
    suggestedScheduleName: proposal.suggestedScheduleName || undefined,
    knownRooms: response.knownRooms ?? [],
    availableFinishes: (proposal.availableFinishes ?? []).filter(Boolean),
    finishModelMappings: mappings
  };
}

function createId(): string {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return `req-${Math.random().toString(36).slice(2, 10)}`;
}

import type {
  CapturedPage,
  ExtractSpecRequest,
  ExtractSpecResponse,
  ScheduleItem
} from "./types";

const DEFAULT_BACKEND_BASE_URL = "http://10.0.0.104:8080";
export const EXTRACT_TIMEOUT_MS = 180_000;
const EXTRACT_PATH = "/v1/extract-spec";

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
  projectName: string;
  knownZones: string[];
  knownCategories: string[];
  now?: Date;
  onProgress?: (tokenCount: number) => void;
};

export async function extractScheduleItem({
  capturedPage,
  projectName,
  knownZones,
  knownCategories,
  now = new Date(),
  onProgress
}: ExtractScheduleItemArgs): Promise<ScheduleItem> {
  const controller = new AbortController();
  const timeoutId = window.setTimeout(() => controller.abort(), EXTRACT_TIMEOUT_MS);
  const request = buildExtractSpecRequest({
    capturedPage,
    projectName,
    knownZones,
    knownCategories,
    now
  });

  try {
    const apiUrl = getExtractApiUrl();
    console.debug("Sending extraction request to", apiUrl, "with payload", request);
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
          return toScheduleItem(payload, now);
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
  projectName,
  knownZones,
  knownCategories,
  now
}: Required<Omit<ExtractScheduleItemArgs, "onProgress">> & { now: Date }): ExtractSpecRequest {
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
      projectName,
      knownZones,
      knownCategories
    },
    options: {
      includeDebug: true,
      returnAlternatives: false
    }
  };
}

function toScheduleItem(response: ExtractSpecResponse, now: Date): ScheduleItem {
  const proposal = response.proposal;
  if (!proposal) {
    throw new ExtractionError("invalid", "Extraction response was missing a proposal.");
  }

  return {
    id: `draft-${response.requestId}`,
    capturedAt: now.toISOString(),
    zone: proposal.zone,
    title: proposal.title,
    manufacturer: proposal.manufacturer,
    modelNumber: proposal.modelNumber,
    category: proposal.category,
    description: proposal.description,
    finish: proposal.finish,
    finishModelNumber: proposal.finishModelNumber || undefined,
    requiredAddOns: proposal.requiredAddOns ?? [],
    optionalCompanions: proposal.optionalCompanions ?? [],
    sourceUrl: proposal.sourceUrl,
    sourceTitle: proposal.sourceTitle,
    sourceImageUrl: proposal.sourceImageUrl,
    sourcePdfLinks: proposal.sourcePdfLinks ?? []
  };
}

function createId(): string {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return `req-${Math.random().toString(36).slice(2, 10)}`;
}

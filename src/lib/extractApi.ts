import type {
  CapturedPage,
  ExtractSpecRequest,
  ExtractSpecResponse,
  ScheduleItem
} from "./types";

const DEFAULT_EXTRACT_API_URL = "http://10.0.0.104:8080/v1/extract-spec";
const EXTRACT_TIMEOUT_MS = 18_000;

type ExtractScheduleItemArgs = {
  capturedPage: CapturedPage;
  projectName: string;
  knownZones: string[];
  knownCategories: string[];
  now?: Date;
};

export async function extractScheduleItem({
  capturedPage,
  projectName,
  knownZones,
  knownCategories,
  now = new Date()
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
    const response = await fetch(getExtractApiUrl(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
      signal: controller.signal
    });

    const payload = await parseExtractSpecResponse(response);
    if (!response.ok) {
      throw new Error(payload?.error?.message || "Extraction request failed.");
    }
    if (!payload || payload.status !== "ok" || !payload.proposal) {
      throw new Error("Extraction request failed.");
    }

    return toScheduleItem(payload, now);
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new Error("Extraction request timed out.");
    }
    throw error;
  } finally {
    window.clearTimeout(timeoutId);
  }
}

function getExtractApiUrl(): string {
  return import.meta.env.VITE_SALLY_EXTRACT_API_URL || DEFAULT_EXTRACT_API_URL;
}

async function parseExtractSpecResponse(response: Response): Promise<ExtractSpecResponse | null> {
  const text = await response.text();
  if (!text.trim()) {
    return null;
  }

  try {
    return JSON.parse(text) as ExtractSpecResponse;
  } catch {
    if (response.ok) {
      throw new Error("Extraction request failed.");
    }
    return null;
  }
}

function buildExtractSpecRequest({
  capturedPage,
  projectName,
  knownZones,
  knownCategories,
  now
}: Required<ExtractScheduleItemArgs>): ExtractSpecRequest {
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
    throw new Error("Extraction response was missing a proposal.");
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
    requiredAddOns: proposal.requiredAddOns,
    optionalCompanions: proposal.optionalCompanions,
    sourceUrl: proposal.sourceUrl,
    sourceTitle: proposal.sourceTitle,
    sourceImageUrl: proposal.sourceImageUrl,
    sourcePdfLinks: proposal.sourcePdfLinks
  };
}

function createId(): string {
  if (globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return `req-${Math.random().toString(36).slice(2, 10)}`;
}

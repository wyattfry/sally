import { afterEach, describe, expect, it, vi } from "vitest";
import type { CapturedPage, ExtractSpecResponse, ScheduleItem } from "./types";
import { extractScheduleItem } from "./extractApi";

const FIXED_NOW = new Date("2026-04-24T18:30:00.000Z");

function capturedPage(overrides: Partial<CapturedPage> = {}): CapturedPage {
  return {
    title: "Example Co. WF-200 Wall Faucet",
    url: "https://example.com/products/wf-200",
    visibleText: "Wall-mounted faucet rough-in dimensions and installation notes.",
    mainImageUrl: "https://example.com/faucet.jpg",
    structuredData: [],
    pdfLinks: ["https://example.com/spec-sheet.pdf"],
    ...overrides
  };
}

function successResponse(overrides: Partial<ExtractSpecResponse> = {}): ExtractSpecResponse {
  return {
    requestId: "request-123",
    status: "ok",
    proposal: {
      title: "Wall Faucet",
      manufacturer: "Example Co.",
      modelNumber: "WF-200",
      category: "Plumbing Fixture",
      description: "Wall-mounted faucet with rough-in requirements.",
      finish: "Polished Chrome",
      finishModelNumber: "",
      availableFinishes: ["Polished Chrome"],
      finishModelMappings: [{ finish: "Polished Chrome", modelNumber: "WF-200-PC" }],
      requiredAddOns: ["Rough valve body"],
      optionalCompanions: ["Drain assembly"],
      zone: "Primary Bath",
      sourceUrl: "https://example.com/products/wf-200",
      sourceTitle: "Example Co. WF-200 Wall Faucet",
      sourceImageUrl: "https://example.com/faucet.jpg",
      sourcePdfLinks: ["https://example.com/spec-sheet.pdf"]
    },
    meta: {
      provider: "stub",
      model: "stub-extractor",
      promptVersion: "extract-spec-v1",
      durationMs: 25
    },
    ...overrides
  };
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("extractScheduleItem", () => {
  it("builds the request from captured page plus project context", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(successResponse()), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await extractScheduleItem({
      capturedPage: capturedPage(),
      projectName: "My New Project",
      knownZones: ["Primary Bath", "Powder Room"],
      knownCategories: ["Plumbing Fixture", "Lighting"],
      now: FIXED_NOW
    });

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://localhost:8080/v1/extract-spec");
    expect(init.method).toBe("POST");
    expect(init.signal).toBeInstanceOf(AbortSignal);

    const body = JSON.parse(String(init.body));
    expect(body.page).toMatchObject({
      title: "Example Co. WF-200 Wall Faucet",
      url: "https://example.com/products/wf-200",
      visibleText: "Wall-mounted faucet rough-in dimensions and installation notes.",
      mainImageUrl: "https://example.com/faucet.jpg"
    });
    expect(body.projectContext).toEqual({
      projectName: "My New Project",
      knownZones: ["Primary Bath", "Powder Room"],
      knownCategories: ["Plumbing Fixture", "Lighting"]
    });
    expect(body.options).toEqual({
      includeDebug: true,
      returnAlternatives: false
    });
    expect(body.client.app).toBe("sally-extension");
    expect(body.sentAt).toBe("2026-04-24T18:30:00.000Z");
  });

  it("decodes a successful response into a draft proposal", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(successResponse()), {
          status: 200,
          headers: { "Content-Type": "application/json" }
        })
      )
    );

    const item = await extractScheduleItem({
      capturedPage: capturedPage(),
      projectName: "My New Project",
      knownZones: ["Primary Bath"],
      knownCategories: ["Plumbing Fixture"],
      now: FIXED_NOW
    });

    expect(item).toMatchObject<Partial<ScheduleItem>>({
      capturedAt: "2026-04-24T18:30:00.000Z",
      zone: "Primary Bath",
      title: "Wall Faucet",
      manufacturer: "Example Co.",
      modelNumber: "WF-200",
      category: "Plumbing Fixture",
      description: "Wall-mounted faucet with rough-in requirements.",
      finish: "Polished Chrome",
      sourceUrl: "https://example.com/products/wf-200",
      sourceTitle: "Example Co. WF-200 Wall Faucet",
      sourceImageUrl: "https://example.com/faucet.jpg",
      sourcePdfLinks: ["https://example.com/spec-sheet.pdf"]
    });
    expect(item.id).toBeTruthy();
  });

  it("throws on non-OK backend responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            requestId: "request-123",
            status: "error",
            error: { code: "MODEL_TIMEOUT", message: "Extraction did not complete in time." },
            meta: {
              provider: "stub",
              model: "stub-extractor",
              promptVersion: "extract-spec-v1",
              durationMs: 18000
            }
          }),
          {
            status: 500,
            headers: { "Content-Type": "application/json" }
          }
        )
      )
    );

    await expect(
      extractScheduleItem({
        capturedPage: capturedPage(),
        projectName: "My New Project",
        knownZones: [],
        knownCategories: [],
        now: FIXED_NOW
      })
    ).rejects.toThrow("Extraction did not complete in time.");
  });

  it("honors timeout behavior with AbortController", async () => {
    vi.useFakeTimers();

    const fetchMock = vi.fn((_url: string, init?: RequestInit) => {
      const signal = init?.signal;
      return new Promise<Response>((_resolve, reject) => {
        signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")));
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const promise = extractScheduleItem({
      capturedPage: capturedPage(),
      projectName: "My New Project",
      knownZones: [],
      knownCategories: [],
      now: FIXED_NOW
    });
    const rejection = expect(promise).rejects.toThrow("Extraction request timed out.");

    await vi.advanceTimersByTimeAsync(18_000);

    await rejection;
    vi.useRealTimers();
  });

  it("turns non-JSON backend errors into a stable extraction error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("", {
          status: 502,
          headers: { "Content-Type": "text/plain" }
        })
      )
    );

    await expect(
      extractScheduleItem({
        capturedPage: capturedPage(),
        projectName: "My New Project",
        knownZones: [],
        knownCategories: [],
        now: FIXED_NOW
      })
    ).rejects.toThrow("Extraction request failed.");
  });
});

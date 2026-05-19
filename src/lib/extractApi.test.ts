import { afterEach, describe, expect, it, vi } from "vitest";
import type { CapturedPage, ExtractSpecResponse, ScheduleItem } from "./types";
import { EXTRACT_TIMEOUT_MS, extractScheduleItem } from "./extractApi";

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
  vi.unstubAllGlobals();
});

describe("extractScheduleItem", () => {
  it("builds the request from captured page plus project context", async () => {
    vi.stubGlobal("__SALLY_CONFIG__", {
      backendBaseUrl: "http://localhost:8080",
      allowMockFallback: false,
      developmentMode: false
    });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(sseDone(successResponse()), {
        status: 200,
        headers: { "Content-Type": "text/event-stream" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await extractScheduleItem({
      capturedPage: capturedPage(),
      knownCategories: ["Plumbing Fixture", "Lighting"],
      now: FIXED_NOW
    });

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://localhost:8080/api/v1/extract-spec");
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
      projectName: "",
      knownCategories: ["Plumbing Fixture", "Lighting"],
      knownRooms: [],
      knownScheduleNames: []
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
        new Response(sseDone(successResponse()), {
          status: 200,
          headers: { "Content-Type": "text/event-stream" }
        })
      )
    );

    const { item } = await extractScheduleItem({
      capturedPage: capturedPage(),
      knownCategories: ["Plumbing Fixture"],
      now: FIXED_NOW
    });

    expect(item).toMatchObject<Partial<ScheduleItem>>({
      capturedAt: "2026-04-24T18:30:00.000Z",
      sourceImageUrl: "https://example.com/faucet.jpg",
      sourcePdfLinks: ["https://example.com/spec-sheet.pdf"]
    });
    expect(item.id).toBeTruthy();
  });

  it("synthesizes a mapping for the initial finish when the LLM didn't provide one", async () => {
    // When the LLM enumerates available finishes but only knows the
    // current SKU (no per-finish mappings), the initial finish/model
    // pair must still be in finishModelMappings so the panel can
    // restore the Model on a round-trip back to the original finish.
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          sseDone(successResponse({
            proposal: {
              ...successResponse().proposal!,
              finish: "Matte Black / Weathered Oak Wood",
              modelNumber: "700GRC30BW-LED930",
              finishModelNumber: "700GRC30BW-LED930",
              availableFinishes: [
                "Hand Rubbed Antique Brass / Natural Oak",
                "Matte Black / Weathered Oak Wood",
                "Natural Brass / Weathered Oak"
              ],
              finishModelMappings: []
            }
          })),
          { status: 200, headers: { "Content-Type": "text/event-stream" } }
        )
      )
    );

    const result = await extractScheduleItem({
      capturedPage: capturedPage(),
      knownCategories: [],
      now: FIXED_NOW
    });

    expect(result.item.data.model_number).toBe("700GRC30BW-LED930");
    expect(result.availableFinishes).toHaveLength(3);
    // Synthesized: the initial finish gets a mapping even though the LLM
    // returned finishModelMappings:[].
    expect(result.finishModelMappings).toEqual([
      { finish: "Matte Black / Weathered Oak Wood", modelNumber: "700GRC30BW-LED930" }
    ]);
  });

  it("prefers the finish-specific SKU and surfaces finish mappings for the panel", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(sseDone(successResponse()), {
          status: 200,
          headers: { "Content-Type": "text/event-stream" }
        })
      )
    );

    const result = await extractScheduleItem({
      capturedPage: capturedPage(),
      knownCategories: [],
      now: FIXED_NOW
    });

    // model_number takes the finish-specific SKU when the LLM returned a mapping
    // for the selected finish, even though `modelNumber` itself was the base.
    expect(result.item.data.model_number).toBe("WF-200-PC");
    expect(result.availableFinishes).toEqual(["Polished Chrome"]);
    expect(result.finishModelMappings).toEqual([
      { finish: "Polished Chrome", modelNumber: "WF-200-PC" }
    ]);
  });

  it("strips <UNKNOWN> room sentinel from the proposal", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          sseDone(successResponse({ proposal: { ...successResponse().proposal!, room: "<UNKNOWN>" } })),
          { status: 200, headers: { "Content-Type": "text/event-stream" } }
        )
      )
    );

    const { item } = await extractScheduleItem({
      capturedPage: capturedPage(),
      knownCategories: [],
      now: FIXED_NOW
    });

    expect(item.room).toBeUndefined();
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
      knownCategories: [],
      now: FIXED_NOW
    });
    const rejection = expect(promise).rejects.toThrow("Extraction request timed out.");

    await vi.advanceTimersByTimeAsync(EXTRACT_TIMEOUT_MS);

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
        knownCategories: [],
        now: FIXED_NOW
      })
    ).rejects.toThrow("Extraction request failed.");
  });
});

function sseDone(response: ExtractSpecResponse): string {
  return `event: done\ndata: ${JSON.stringify(response)}\n\n`;
}

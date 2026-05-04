import { describe, expect, it } from "vitest";
import { mockExtractScheduleItem } from "./mockExtraction";
import type { CapturedPage } from "./types";

function capturedPage(overrides: Partial<CapturedPage> = {}): CapturedPage {
  return {
    title: "Example Co. WF-200 Wall Faucet",
    url: "https://example.com/products/wf-200",
    visibleText:
      "Example Co. WF-200 wall-mounted faucet in polished chrome. Requires rough valve body and drain assembly.",
    mainImageUrl: "https://example.com/faucet.jpg",
    structuredData: [
      {
        "@type": "Product",
        name: "Wall Faucet",
        brand: { name: "Example Co." },
        sku: "WF-200"
      }
    ],
    pdfLinks: ["https://example.com/spec-sheet.pdf"],
    ...overrides
  };
}

describe("mockExtractScheduleItem", () => {
  it("creates an editable schedule proposal from captured product data", () => {
    const item = mockExtractScheduleItem(capturedPage(), new Date("2026-04-21T12:00:00.000Z"));

    expect(item.capturedAt).toBe("2026-04-21T12:00:00.000Z");
    expect(item.sourceUrl).toBe("https://example.com/products/wf-200");
    expect(item.sourceTitle).toBe("Example Co. WF-200 Wall Faucet");
    expect(item.sourceImageUrl).toBe("https://example.com/faucet.jpg");
    expect(item.sourcePdfLinks).toEqual(["https://example.com/spec-sheet.pdf"]);

    expect(item.data.title).toBe("Wall Faucet");
    expect(item.data.manufacturer).toBe("Example Co.");
    expect(item.data.model_number).toBe("WF-200");
    expect(item.data.finish).toBe("Polished Chrome");
    expect(item.data.description).toContain("wall-mounted faucet");
  });

  it("uses conservative fallback values when product data is sparse", () => {
    const item = mockExtractScheduleItem(
      capturedPage({
        title: "Unknown Catalog Page",
        visibleText: "A page with limited technical detail.",
        structuredData: [],
        mainImageUrl: undefined,
        pdfLinks: []
      }),
      new Date("2026-04-21T12:00:00.000Z")
    );

    expect(item.data.title).toBe("Unknown Catalog Page");
    expect(item.data.manufacturer ?? "").toBe("");
    expect(item.data.model_number ?? "").toBe("");
    expect(item.data.notes ?? "").toBe("");
  });
});

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
  it("creates a deterministic editable schedule proposal from captured product data", () => {
    const item = mockExtractScheduleItem(capturedPage(), new Date("2026-04-21T12:00:00.000Z"));

    expect(item).toMatchObject({
      id: "mock-https-example-com-products-wf-200",
      capturedAt: "2026-04-21T12:00:00.000Z",
      zone: "",
      title: "Wall Faucet",
      manufacturer: "Example Co.",
      modelNumber: "WF-200",
      category: "Faucet",
      finish: "Polished Chrome",
      finishModelNumber: "WF-200-PC",
      requiredAddOns: ["Rough valve body", "Drain assembly"],
      sourceUrl: "https://example.com/products/wf-200",
      sourceTitle: "Example Co. WF-200 Wall Faucet",
      sourceImageUrl: "https://example.com/faucet.jpg",
      sourcePdfLinks: ["https://example.com/spec-sheet.pdf"]
    });
    expect(item.description).toContain("wall-mounted faucet");
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

    expect(item.title).toBe("Unknown Catalog Page");
    expect(item.manufacturer).toBe("");
    expect(item.modelNumber).toBe("");
    expect(item.category).toBe("Uncategorized");
    expect(item.requiredAddOns).toEqual([]);
  });
});


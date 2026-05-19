import { describe, expect, it } from "vitest";
import { capturePage } from "./capturePage";

describe("capturePage", () => {
  it("captures page title, url, visible text, structured data, pdf links, and main image", () => {
    document.documentElement.innerHTML = `
      <head>
        <title>Wall Faucet | Example Co.</title>
        <meta property="og:image" content="https://example.com/og-faucet.jpg" />
        <script type="application/ld+json">
          {
            "@context": "https://schema.org",
            "@type": "Product",
            "name": "Wall Faucet",
            "brand": {"name": "Example Co."},
            "sku": "WF-200"
          }
        </script>
      </head>
      <body>
        <main>
          <h1>Wall Faucet</h1>
          <p>Solid brass wall-mounted faucet. Requires rough valve body.</p>
          <a href="/downloads/install-guide.pdf">Installation Guide</a>
          <a href="https://example.com/spec-sheet.pdf">Spec Sheet</a>
          <img src="https://example.com/tiny.jpg" width="20" height="20" />
          <img src="https://example.com/faucet.jpg" width="900" height="700" />
        </main>
        <aside style="display: none">Hidden marketing text</aside>
      </body>
    `;

    const captured = capturePage(document, new URL("https://example.com/products/wf-200"));

    expect(captured.title).toBe("Wall Faucet | Example Co.");
    expect(captured.url).toBe("https://example.com/products/wf-200");
    expect(captured.visibleText).toContain("Solid brass wall-mounted faucet");
    expect(captured.visibleText).not.toContain("Hidden marketing text");
    expect(captured.structuredData).toMatchObject([
      {
        "@type": "Product",
        name: "Wall Faucet"
      }
    ]);
    expect(captured.pdfLinks).toEqual([
      "https://example.com/downloads/install-guide.pdf",
      "https://example.com/spec-sheet.pdf"
    ]);
    expect(captured.mainImageUrl).toBe("https://example.com/og-faucet.jpg");
  });

  it("captures finish/color variant names from attribute-only swatch pickers", () => {
    // Mirrors the structure of Home Depot's super-sku picker: the active
    // swatch's name is in <p> text, but the inactive ones live only in
    // alt/aria-label/value attributes.
    document.documentElement.innerHTML = `
      <body>
        <div data-fusion-component="@thd-olt-component-react/super-sku">
          <p>Color/Finish</p><span>:</span>
          <p>Spot Resist Stainless</p>
          <div>
            <button aria-pressed="true" value="Spot Resist Stainless" aria-label="Spot Resist Stainless">
              <img alt="Spot Resist Stainless" src="https://example.com/srs.jpg" />
            </button>
            <button aria-pressed="false" value="Matte Black" aria-label="Matte Black">
              <img alt="Matte Black" src="https://example.com/mb.jpg" />
            </button>
          </div>
        </div>
      </body>
    `;

    const captured = capturePage(document, new URL("https://example.com/p"));

    expect(captured.visibleText).toContain("[Variant options:]");
    expect(captured.visibleText).toContain("Spot Resist Stainless");
    // The critical assertion: the attribute-only swatch is now visible to the LLM.
    expect(captured.visibleText).toContain("Matte Black");
  });

  it("falls back to the largest visible image when no open graph image exists", () => {
    document.documentElement.innerHTML = `
      <head><title>Toilet</title></head>
      <body>
        <img src="/small.jpg" width="80" height="80" />
        <img src="/large.jpg" width="800" height="600" />
      </body>
    `;

    const captured = capturePage(document, new URL("https://example.com/toilet"));

    expect(captured.mainImageUrl).toBe("https://example.com/large.jpg");
  });
});


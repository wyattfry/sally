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

  it("extracts product entries from an embedded Apollo state script", () => {
    const apollo = {
      "BaseProduct:itemId/326882450": {
        __typename: "BaseProduct",
        identifiers: {
          __typename: "Identifiers",
          modelNumber: "87260SRS",
          brandName: "MOEN",
          isSuperSku: true,
          parentId: "328375714"
        }
      },
      // Unrelated entry — should be filtered out.
      "PlccPromotion:abc": { __typename: "PlccPromotion", code: "PLCC10" }
    };
    document.documentElement.innerHTML = `
      <head>
        <title>Doherty Faucet</title>
        <script>
          window.__APOLLO_STATE__ = ${JSON.stringify(apollo)};
        </script>
      </head>
      <body><h1>Doherty</h1></body>
    `;

    const captured = capturePage(document, new URL("https://www.homedepot.com/pep/MOEN-Doherty-87260SRS/326882450"));

    expect(captured.visibleText).toContain("[Embedded product state JSON:]");
    expect(captured.visibleText).toContain("BaseProduct");
    expect(captured.visibleText).toContain("87260SRS");
    expect(captured.visibleText).toContain("isSuperSku");
    expect(captured.visibleText).not.toContain("PlccPromotion");
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

  it("extracts finish names from common-prefix swatch alts (Ferguson pattern)", () => {
    // Real DOM shape from fergusonhome.com: 5 swatch <div>s, each with one
    // <img> whose alt is the full product title plus the finish at the end.
    // My old per-label length cap rejected these as too long; common-prefix
    // collapse should recover the finish suffixes.
    const titlePrefix = "Kohler Castia by Studio McGee 1.2 GPM Widespread Bathroom Faucet with Drain Assembly";
    const finishes = ["Matte Black", "Polished Chrome", "Vibrant Brushed Moderne Brass", "Vibrant Brushed Nickel", "Vibrant Polished Nickel"];
    document.documentElement.innerHTML = `
      <body>
        <div>
          <strong>Finish: </strong><span data-automation="finish-name">Matte Black</span>
          <span>- 81 In Stock</span>
          <div class="list flex flex-row flex-wrap">
            ${finishes.map((f, i) => `
              <div data-automation="finish-swatch" role="checkbox" aria-checked="${i===0}">
                <img alt="${titlePrefix} ${f}" src="https://example.com/${i}.jpg" />
              </div>
            `).join("")}
          </div>
        </div>
      </body>
    `;

    const captured = capturePage(document, new URL("https://www.fergusonhome.com/kohler-k-35908-4/s1939306"));

    expect(captured.visibleText).toContain("[Variant options:]");
    for (const f of finishes) {
      expect(captured.visibleText).toContain(f);
    }
    // The product title should NOT appear in the Variant options section —
    // common-prefix stripping should have removed it.
    const variantSection = captured.visibleText.split("[Variant options:]")[1] ?? "";
    expect(variantSection).not.toContain("Widespread Bathroom Faucet with Drain Assembly");
  });

  it("preserves shared qualifier words like 'Vibrant' when stripping the prefix", () => {
    // Pathological case: all four finishes start with "Vibrant". A naive
    // common-prefix strip would lump "Vibrant " into the prefix and we'd
    // lose the qualifier. The active-finish anchor (via aria-checked=true)
    // tells us where the finish actually begins.
    const titlePrefix = "Kohler Castia by Studio McGee 1.2 GPM Widespread Bathroom Faucet";
    const finishes = [
      "Vibrant Brushed Nickel",
      "Vibrant Polished Nickel",
      "Vibrant Brushed Moderne Brass",
      "Vibrant Polished Brass"
    ];
    document.documentElement.innerHTML = `
      <body>
        <div>
          <strong>Finish: </strong><span data-automation="finish-name">Vibrant Brushed Nickel</span>
          <div>
            ${finishes.map((f, i) => `
              <div data-automation="finish-swatch" aria-checked="${i===0}">
                <img alt="${titlePrefix} ${f}" />
              </div>
            `).join("")}
          </div>
        </div>
      </body>
    `;

    const captured = capturePage(document, new URL("https://example.com/p"));
    const variantSection = captured.visibleText.split("[Variant options:]")[1] ?? "";

    for (const f of finishes) {
      expect(variantSection).toContain(f);
    }
    // Specifically: "Vibrant" must be preserved, not stripped into the prefix.
    expect(variantSection).toMatch(/Vibrant\s+Brushed\s+Nickel/);
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


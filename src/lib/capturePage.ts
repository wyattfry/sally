import type { CapturedPage } from "./types";

const MAX_VISIBLE_TEXT_LENGTH = 12000;

export function capturePage(doc: Document, location: Location | URL): CapturedPage {
  return {
    title: doc.title.trim(),
    url: location.href,
    visibleText: captureVisibleText(doc),
    mainImageUrl: findMainImageUrl(doc, location.href),
    allImageUrls: findAllImageUrls(doc, location.href),
    structuredData: parseStructuredData(doc),
    pdfLinks: findPdfLinks(doc, location.href)
  };
}

function captureVisibleText(doc: Document): string {
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const text = node.textContent?.trim();
      if (!text) {
        return NodeFilter.FILTER_REJECT;
      }

      const element = node.parentElement;
      if (!element || !isVisible(element)) {
        return NodeFilter.FILTER_REJECT;
      }

      return NodeFilter.FILTER_ACCEPT;
    }
  });

  const chunks: string[] = [];
  while (walker.nextNode()) {
    const text = walker.currentNode.textContent?.replace(/\s+/g, " ").trim();
    if (text) {
      chunks.push(text);
    }
  }

  const variantOptions = captureVariantOptions(doc);
  let out = chunks.join("\n");
  if (variantOptions) {
    out += "\n[Variant options:]\n" + variantOptions;
  }
  if (out.length < MAX_VISIBLE_TEXT_LENGTH) {
    const collapsed = captureCollapsedContent(doc);
    if (collapsed) {
      out += "\n[Collapsed sections:]\n" + collapsed;
    }
  }
  return out.slice(0, MAX_VISIBLE_TEXT_LENGTH);
}

// captureVariantOptions extracts finish/color/size names from product
// variant selectors that render as image-only swatches. The labels live in
// alt/aria-label/value attributes — text-node walkers miss them entirely,
// which is why pages like Home Depot and Ferguson only had the
// currently-selected finish in the prompt.
function captureVariantOptions(doc: Document): string {
  const containers = findVariantContainers(doc);
  if (containers.length === 0) return "";

  const seen = new Set<string>();
  const lines: string[] = [];

  for (const container of containers) {
    const label = inferContainerLabel(container);
    const values = new Set<string>();
    for (const el of container.querySelectorAll<HTMLElement>("[alt], [aria-label], [value]")) {
      for (const attr of ["aria-label", "alt", "value"]) {
        const raw = el.getAttribute(attr);
        if (!raw) continue;
        const v = raw.trim();
        if (!v || v.length > 80) continue;
        values.add(v);
      }
    }
    if (values.size === 0) continue;
    const joined = Array.from(values).join(", ");
    const key = (label || "") + "|" + joined;
    if (seen.has(key)) continue;
    seen.add(key);
    lines.push(label ? `${label}: ${joined}` : joined);
    if (lines.length >= 12) break;
  }

  return lines.join("\n").slice(0, 2000);
}

const VARIANT_LABEL_RE = /\b(color\s*\/?\s*finish|finish|color|variant|option|swatch)\b/i;

function findVariantContainers(doc: Document): Element[] {
  const seen = new Set<Element>();
  const out: Element[] = [];

  // (a) Explicit signals: known component tags, ARIA radiogroups.
  const explicit = doc.querySelectorAll(
    '[data-component*="super-sku" i], [data-component*="variant" i], ' +
    '[data-component*="swatch" i], [data-fusion-component*="super-sku" i], ' +
    '[role="radiogroup"], [aria-label*="finish" i], [aria-label*="color" i]'
  );
  for (const el of explicit) {
    if (!seen.has(el)) { seen.add(el); out.push(el); }
  }

  // (b) Heuristic: any element whose own text contains "Color/Finish",
  //     "Finish:", etc. AND that contains <img alt> or <button aria-label>
  //     children — the swatch label sibling pattern.
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_ELEMENT, {
    acceptNode(node) {
      const el = node as Element;
      if (seen.has(el)) return NodeFilter.FILTER_REJECT;
      const own = (el as HTMLElement).innerText?.slice(0, 200) ?? "";
      if (!VARIANT_LABEL_RE.test(own)) return NodeFilter.FILTER_SKIP;
      // Need at least 2 attribute-labeled descendants to count as a swatch row
      const labeled = el.querySelectorAll("[alt], [aria-label]").length;
      if (labeled < 2) return NodeFilter.FILTER_SKIP;
      return NodeFilter.FILTER_ACCEPT;
    }
  });
  while (walker.nextNode()) {
    const el = walker.currentNode as Element;
    // Walk up to a reasonable container parent (sibling group) to avoid
    // grabbing only the label paragraph.
    const container = el.closest('[role="radiogroup"], [data-component], ul, fieldset, section, div') || el;
    if (!seen.has(container)) { seen.add(container); out.push(container); }
    if (out.length >= 8) break;
  }

  return out;
}

function inferContainerLabel(container: Element): string {
  // Look for the label text inside (e.g., "Color/Finish") in the first
  // few text-bearing children. Falls back to aria-label on the container.
  const aria = container.getAttribute("aria-label");
  if (aria && VARIANT_LABEL_RE.test(aria)) return aria.trim();
  const text = (container as HTMLElement).innerText?.slice(0, 200) ?? "";
  const m = text.match(/(Color\s*\/?\s*Finish|Finish|Color|Variant|Option)\s*:?/i);
  return m ? m[1].trim() : "";
}

function captureCollapsedContent(doc: Document): string {
  const chunks: string[] = [];

  // Capture content from aria-controls panels that are currently hidden
  for (const trigger of doc.querySelectorAll("[aria-controls]")) {
    const panelId = trigger.getAttribute("aria-controls");
    if (!panelId) continue;
    const panel = doc.getElementById(panelId);
    if (!panel) continue;
    const style = panel.ownerDocument.defaultView?.getComputedStyle(panel);
    if (!style || style.display !== "none") continue;
    const text = panel.textContent?.replace(/\s+/g, " ").trim();
    if (text) chunks.push(text);
  }

  // Capture content from closed <details> elements
  for (const details of doc.querySelectorAll<HTMLDetailsElement>("details:not([open])")) {
    const summary = details.querySelector("summary");
    const text = Array.from(details.childNodes)
      .filter((n) => n !== summary)
      .map((n) => n.textContent?.replace(/\s+/g, " ").trim())
      .filter(Boolean)
      .join(" ");
    if (text) chunks.push(text);
  }

  return chunks.join("\n");
}

function isVisible(element: Element): boolean {
  const htmlElement = element as HTMLElement;
  if (htmlElement.hidden) {
    return false;
  }

  const style = element.ownerDocument.defaultView?.getComputedStyle(element);
  if (!style) {
    return true;
  }

  return style.display !== "none" && style.visibility !== "hidden" && style.opacity !== "0";
}

function parseStructuredData(doc: Document): unknown[] {
  return Array.from(doc.querySelectorAll('script[type="application/ld+json"]'))
    .flatMap((script) => parseJsonLd(script.textContent ?? ""))
    .filter(Boolean);
}

function parseJsonLd(rawJson: string): unknown[] {
  try {
    const parsed = JSON.parse(rawJson);
    return Array.isArray(parsed) ? parsed : [parsed];
  } catch {
    return [];
  }
}

function findPdfLinks(doc: Document, baseUrl: string): string[] {
  const candidates = Array.from(doc.querySelectorAll<HTMLAnchorElement>("a[href]"));
  const seen = new Set<string>();
  const links: string[] = [];

  for (const anchor of candidates) {
    const href = anchor.getAttribute("href");
    if (!href) {
      continue;
    }

    const text = `${anchor.textContent ?? ""} ${href}`.toLowerCase();
    const looksLikeSpec =
      href.toLowerCase().includes(".pdf") &&
      /\b(pdf|spec|cut.?sheet|install|installation|technical|dimension|guide)\b/.test(text);

    if (!looksLikeSpec) {
      continue;
    }

    const absolute = new URL(href, baseUrl).href;
    if (!seen.has(absolute)) {
      seen.add(absolute);
      links.push(absolute);
    }
  }

  return links.slice(0, 8);
}

function findAllImageUrls(doc: Document, baseUrl: string): string[] {
  const seen = new Set<string>();
  const urls: string[] = [];

  const ogImage = doc.querySelector<HTMLMetaElement>('meta[property="og:image"], meta[name="og:image"]');
  const ogUrl = ogImage?.content?.trim();
  if (ogUrl) {
    try {
      const abs = new URL(ogUrl, baseUrl).href;
      if (!seen.has(abs)) { seen.add(abs); urls.push(abs); }
    } catch { /* ignore invalid URLs */ }
  }

  const images = Array.from(doc.images)
    .filter((img) => isVisible(img) && Boolean(img.currentSrc || img.src))
    .filter((img) => imageArea(img) >= 40000)
    .sort((a, b) => imageArea(b) - imageArea(a));

  for (const img of images) {
    const src = img.getAttribute("src") || img.currentSrc || img.src;
    if (!src) continue;
    try {
      const abs = new URL(src, baseUrl).href;
      if (!seen.has(abs)) { seen.add(abs); urls.push(abs); }
    } catch { /* ignore invalid URLs */ }
  }

  return urls.slice(0, 10);
}

function findMainImageUrl(doc: Document, baseUrl: string): string | undefined {
  const openGraphImage = doc.querySelector<HTMLMetaElement>('meta[property="og:image"], meta[name="og:image"]');
  const openGraphUrl = openGraphImage?.content?.trim();
  if (openGraphUrl) {
    return new URL(openGraphUrl, baseUrl).href;
  }

  const image = Array.from(doc.images)
    .filter((candidate) => isVisible(candidate) && Boolean(candidate.currentSrc || candidate.src))
    .sort((a, b) => imageArea(b) - imageArea(a))[0];

  const src = image?.getAttribute("src") || image?.currentSrc || image?.src;
  return src ? new URL(src, baseUrl).href : undefined;
}

function imageArea(image: HTMLImageElement): number {
  const width = image.naturalWidth || image.width || Number(image.getAttribute("width")) || 0;
  const height = image.naturalHeight || image.height || Number(image.getAttribute("height")) || 0;
  return width * height;
}

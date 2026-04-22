import type { CapturedPage } from "./types";

const MAX_VISIBLE_TEXT_LENGTH = 12000;

export function capturePage(doc: Document, location: Location | URL): CapturedPage {
  return {
    title: doc.title.trim(),
    url: location.href,
    visibleText: captureVisibleText(doc),
    mainImageUrl: findMainImageUrl(doc, location.href),
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

  return chunks.join("\n").slice(0, MAX_VISIBLE_TEXT_LENGTH);
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

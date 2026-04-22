import type { CapturedPage, ScheduleItem } from "./types";

export function mockExtractScheduleItem(captured: CapturedPage, now = new Date()): ScheduleItem {
  const product = findProductData(captured.structuredData);
  const manufacturer = readBrandName(product);
  const modelNumber = readString(product, ["sku", "model", "mpn", "productID"]) || findModelNumber(captured);
  const title = readString(product, ["name"]) || stripSiteSuffix(captured.title);
  const category = inferCategory(`${captured.title} ${captured.visibleText}`);
  const finish = inferFinish(captured.visibleText);

  return {
    id: stableId(captured.url),
    capturedAt: now.toISOString(),
    zone: "",
    title,
    manufacturer,
    modelNumber,
    category,
    description: buildDescription(captured, title),
    finish,
    finishModelNumber: finish && modelNumber ? `${modelNumber}-${finishCode(finish)}` : undefined,
    requiredAddOns: inferRequiredAddOns(captured.visibleText),
    optionalCompanions: inferOptionalCompanions(captured.visibleText),
    sourceUrl: captured.url,
    sourceTitle: captured.title,
    sourceImageUrl: captured.mainImageUrl,
    sourcePdfLinks: captured.pdfLinks
  };
}

function findProductData(structuredData: unknown[]): Record<string, unknown> | undefined {
  return structuredData
    .flatMap((entry) => {
      if (isRecord(entry) && Array.isArray(entry["@graph"])) {
        return entry["@graph"].filter(isRecord);
      }
      return isRecord(entry) ? [entry] : [];
    })
    .find((entry) => {
      const type = entry["@type"];
      return Array.isArray(type) ? type.includes("Product") : type === "Product";
    });
}

function readBrandName(product: Record<string, unknown> | undefined): string {
  const brand = product?.brand;
  if (typeof brand === "string") {
    return brand;
  }
  if (isRecord(brand) && typeof brand.name === "string") {
    return brand.name;
  }
  return "";
}

function readString(product: Record<string, unknown> | undefined, keys: string[]): string {
  if (!product) {
    return "";
  }

  for (const key of keys) {
    const value = product[key];
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  return "";
}

function stripSiteSuffix(title: string): string {
  return title.split(/\s[|-]\s/)[0]?.trim() || title.trim();
}

function findModelNumber(captured: CapturedPage): string {
  const match = `${captured.title} ${captured.visibleText}`.match(/\b[A-Z]{1,5}[- ]?\d{2,5}[A-Z0-9-]*\b/);
  return match?.[0]?.replace(/\s+/, "-") ?? "";
}

function inferCategory(text: string): string {
  const lower = text.toLowerCase();
  if (lower.includes("faucet")) return "Faucet";
  if (lower.includes("toilet")) return "Toilet";
  if (lower.includes("sink") || lower.includes("lavatory")) return "Sink";
  if (lower.includes("shower")) return "Shower";
  if (lower.includes("tub") || lower.includes("bathtub")) return "Tub";
  return "Uncategorized";
}

function inferFinish(text: string): string {
  const lower = text.toLowerCase();
  if (lower.includes("polished chrome")) return "Polished Chrome";
  if (lower.includes("brushed nickel")) return "Brushed Nickel";
  if (lower.includes("matte black")) return "Matte Black";
  if (lower.includes("stainless steel")) return "Stainless Steel";
  return "";
}

function finishCode(finish: string): string {
  return finish
    .split(/\s+/)
    .map((word) => word[0])
    .join("")
    .toUpperCase();
}

function inferRequiredAddOns(text: string): string[] {
  const lower = text.toLowerCase();
  const addOns = [
    ["rough valve body", "Rough valve body"],
    ["pressure-balance valve", "Pressure-balance valve"],
    ["diverter", "Diverter"],
    ["drain assembly", "Drain assembly"],
    ["trap", "Trap"],
    ["toilet seat", "Toilet seat"]
  ] as const;

  return addOns.filter(([needle]) => lower.includes(needle)).map(([, label]) => label);
}

function inferOptionalCompanions(text: string): string[] {
  const lower = text.toLowerCase();
  const companions = [
    ["towel bar", "Towel bar"],
    ["robe hook", "Robe hook"],
    ["soap dispenser", "Soap dispenser"]
  ] as const;

  return companions.filter(([needle]) => lower.includes(needle)).map(([, label]) => label);
}

function buildDescription(captured: CapturedPage, title: string): string {
  const firstUsefulSentence =
    captured.visibleText
      .split(/(?<=[.!?])\s+/)
      .map((sentence) => sentence.trim())
      .find((sentence) => sentence.length > 20) || title;

  return firstUsefulSentence.slice(0, 260);
}

function stableId(url: string): string {
  return `mock-${url.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "")}`;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}


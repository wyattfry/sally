import type { ScheduleItem } from "./types";

const STORAGE_KEY = "sally.scheduleItems";
const ZONES_KEY = "sally.zones";

const DEFAULT_ZONES = [
  "Entry",
  "Kitchen",
  "Powder Room",
  "Primary Bath",
  "Bath 2",
  "Laundry",
  "Exterior"
];

function chromeStorage() {
  if (!globalThis.chrome?.storage?.local) {
    throw new Error("chrome.storage.local is unavailable");
  }
  return globalThis.chrome.storage.local;
}

export async function listScheduleItems(): Promise<ScheduleItem[]> {
  const result = await chromeStorage().get(STORAGE_KEY);
  const items = result[STORAGE_KEY];
  return Array.isArray(items) ? (items as ScheduleItem[]) : [];
}

export async function saveScheduleItem(item: ScheduleItem): Promise<void> {
  const items = await listScheduleItems();
  await chromeStorage().set({ [STORAGE_KEY]: [...items, item] });
}

export async function clearScheduleItems(): Promise<void> {
  await chromeStorage().remove(STORAGE_KEY);
}

export async function listZones(): Promise<string[]> {
  const result = await chromeStorage().get(ZONES_KEY);
  const storedZones = result[ZONES_KEY];
  const zones = Array.isArray(storedZones) ? (storedZones as string[]) : [];
  return uniqueZones([...DEFAULT_ZONES, ...zones]);
}

export async function saveZone(zone: string): Promise<string[]> {
  const trimmedZone = zone.trim();
  if (!trimmedZone) {
    return listZones();
  }

  const zones = uniqueZones([...(await listZones()), trimmedZone]);
  await chromeStorage().set({ [ZONES_KEY]: zones });
  return zones;
}

function uniqueZones(zones: string[]): string[] {
  const seen = new Set<string>();
  return zones.filter((zone) => {
    const normalized = zone.trim().toLowerCase();
    if (!normalized || seen.has(normalized)) {
      return false;
    }
    seen.add(normalized);
    return true;
  });
}

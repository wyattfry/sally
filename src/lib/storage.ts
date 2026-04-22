import type { ScheduleItem } from "./types";

const STORAGE_KEY = "sally.scheduleItems";
const ZONES_KEY = "sally.zones";
const PROJECT_NAME_KEY = "sally.projectName";
const DEFAULT_PROJECT_NAME = "My New Project";

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
  await chromeStorage().set({ [STORAGE_KEY]: [...items, withUniqueId(item, items)] });
}

export async function removeScheduleItem(itemId: string): Promise<ScheduleItem[]> {
  const items = (await listScheduleItems()).filter((item) => item.id !== itemId);
  await chromeStorage().set({ [STORAGE_KEY]: items });
  return items;
}

export async function clearScheduleItems(): Promise<void> {
  await chromeStorage().remove(STORAGE_KEY);
}

export async function getProjectName(): Promise<string> {
  const result = await chromeStorage().get(PROJECT_NAME_KEY);
  const projectName = result[PROJECT_NAME_KEY];
  return typeof projectName === "string" && projectName.trim()
    ? projectName.trim()
    : DEFAULT_PROJECT_NAME;
}

export async function saveProjectName(projectName: string): Promise<string> {
  const trimmedName = projectName.trim() || DEFAULT_PROJECT_NAME;
  await chromeStorage().set({ [PROJECT_NAME_KEY]: trimmedName });
  return trimmedName;
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

function withUniqueId(item: ScheduleItem, existingItems: ScheduleItem[]): ScheduleItem {
  const existingIds = new Set(existingItems.map((existingItem) => existingItem.id));
  if (!existingIds.has(item.id)) {
    return item;
  }

  let suffix = 2;
  let nextId = `${item.id}-${suffix}`;
  while (existingIds.has(nextId)) {
    suffix += 1;
    nextId = `${item.id}-${suffix}`;
  }

  return { ...item, id: nextId };
}

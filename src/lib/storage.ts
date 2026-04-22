import type { ScheduleItem } from "./types";

const STORAGE_KEY = "sally.scheduleItems";

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


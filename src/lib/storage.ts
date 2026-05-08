import type { ActiveContext } from "./types";

const ACTIVE_CONTEXT_KEY = "sally.activeContext";
const BACKEND_URL_KEY = "sally.backendUrl";

export async function getStoredBackendUrl(): Promise<string | null> {
  try {
    const result = await chromeStorage().get(BACKEND_URL_KEY);
    const val = result[BACKEND_URL_KEY];
    return typeof val === "string" && val ? val : null;
  } catch {
    return null;
  }
}

export async function saveBackendUrl(url: string): Promise<void> {
  await chromeStorage().set({ [BACKEND_URL_KEY]: url });
}

function chromeStorage() {
  if (!globalThis.chrome?.storage?.local) {
    throw new Error("chrome.storage.local is unavailable");
  }
  return globalThis.chrome.storage.local;
}

export async function getActiveContext(): Promise<ActiveContext | null> {
  const result = await chromeStorage().get(ACTIVE_CONTEXT_KEY);
  const context = result[ACTIVE_CONTEXT_KEY];
  if (!isActiveContext(context)) {
    return null;
  }
  return context;
}

export async function saveActiveContext(
  context: ActiveContext
): Promise<void> {
  await chromeStorage().set({ [ACTIVE_CONTEXT_KEY]: context });
}

function isActiveContext(value: unknown): value is ActiveContext {
  return (
    typeof value === "object" &&
    value !== null &&
    "projectId" in value &&
    typeof value.projectId === "string" &&
    value.projectId.trim() !== "" &&
    "scheduleId" in value &&
    typeof value.scheduleId === "string" &&
    value.scheduleId.trim() !== ""
  );
}

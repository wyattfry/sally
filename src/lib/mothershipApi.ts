import type { Project, Schedule, ScheduleColumn, ScheduleItem } from "./types";

const DEFAULT_BACKEND_BASE_URL = "http://10.0.0.104:8080";

type SallyRuntimeConfig = {
  backendBaseUrl?: string;
};

export async function checkAuth(): Promise<boolean> {
  try {
    await fetchJSON<unknown>("/api/v1/me");
    return true;
  } catch {
    return false;
  }
}

export function getSignInUrl(): string {
  return `${getBackendBaseUrl()}/auth/google?next=done`;
}

export async function listMothershipProjects(): Promise<Project[]> {
  return fetchJSON<Project[]>("/api/v1/projects");
}

export async function createMothershipProject(name: string): Promise<Project> {
  return fetchJSON<Project>("/api/v1/projects", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, address: "" })
  });
}

export async function listMothershipSchedules(projectId: string): Promise<Schedule[]> {
  return fetchJSON<Schedule[]>(`/api/v1/projects/${encodeURIComponent(projectId)}/schedules`);
}

export async function createMothershipSchedule(projectId: string, name: string): Promise<Schedule> {
  return fetchJSON<Schedule>(`/api/v1/projects/${encodeURIComponent(projectId)}/schedules`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name })
  });
}

export async function listMothershipScheduleColumns(scheduleId: string): Promise<ScheduleColumn[]> {
  return fetchJSON<ScheduleColumn[]>(`/api/v1/schedules/${encodeURIComponent(scheduleId)}/columns`);
}

export async function getMothershipScheduleNextCode(scheduleId: string): Promise<string> {
  const res = await fetchJSON<{ nextCode: string }>(`/api/v1/schedules/${encodeURIComponent(scheduleId)}/next-code`);
  return res.nextCode;
}

export async function saveMothershipScheduleItem(
  scheduleId: string,
  item: ScheduleItem
): Promise<void> {
  await fetchJSON(`/api/v1/schedules/${encodeURIComponent(scheduleId)}/items`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      data: item.data,
      zone: item.zone ?? "",
      sourceUrl: item.sourceUrl,
      sourceTitle: item.sourceTitle,
      sourceImageUrl: item.sourceImageUrl ?? "",
      sourcePdfLinks: item.sourcePdfLinks
    })
  });
}

export function getMothershipScheduleUrl(projectId: string, scheduleId: string): string {
  return `${getBackendBaseUrl()}/projects/${encodeURIComponent(projectId)}/schedules/${encodeURIComponent(scheduleId)}`;
}

async function getSessionToken(): Promise<string | null> {
  if (typeof chrome === "undefined" || !chrome.runtime?.sendMessage) return null;
  return new Promise((resolve) => {
    chrome.runtime.sendMessage(
      { type: "GET_COOKIE", url: getBackendBaseUrl(), name: "sally_session" },
      (result) => {
        if (chrome.runtime.lastError) { resolve(null); return; }
        resolve(result?.value ?? null);
      }
    );
  });
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${getBackendBaseUrl()}${path}`;

  if (typeof chrome !== "undefined" && chrome.runtime && chrome.runtime.sendMessage) {
    const sessionToken = await getSessionToken();
    const mergedInit: RequestInit = {
      ...init,
      headers: {
        ...(sessionToken ? { "X-Session-Token": sessionToken } : {}),
        ...(init?.headers ?? {}),
      },
    };

    const response = await new Promise<any>((resolve, reject) => {
      chrome.runtime.sendMessage(
        { type: "PROXY_FETCH", url, init: mergedInit },
        (result) => {
          if (chrome.runtime.lastError) {
            return reject(new Error(chrome.runtime.lastError.message));
          }
          if (result.error) {
            return reject(new Error(result.error));
          }
          resolve(result);
        }
      );
    });

    if (!response.ok) {
      const message = response.text?.trim() || "Mother Ship request failed.";
      throw new Error(message);
    }
    return JSON.parse(response.text) as T;
  }

  // Fallback for non-extension environment
  const response = await fetch(url, init);
  if (!response.ok) {
    const message = (await response.text()).trim() || "Mother Ship request failed.";
    throw new Error(message);
  }
  return response.json() as Promise<T>;
}

function getBackendBaseUrl(): string {
  const config = (globalThis as { __SALLY_CONFIG__?: SallyRuntimeConfig }).__SALLY_CONFIG__;
  return (
    config?.backendBaseUrl ||
    import.meta.env.VITE_SALLY_BACKEND_BASE_URL ||
    DEFAULT_BACKEND_BASE_URL
  ).replace(/\/+$/, "");
}

import type { Project, Schedule, ScheduleColumn, ScheduleItem } from "./types";

const DEFAULT_BACKEND_BASE_URL = "http://localhost:8080";

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

export async function addMothershipScheduleColumn(projectId: string, scheduleId: string, label: string): Promise<ScheduleColumn[]> {
  await fetchForm(`/projects/${encodeURIComponent(projectId)}/schedules/${encodeURIComponent(scheduleId)}/columns`, new URLSearchParams({ label }));
  return listMothershipScheduleColumns(scheduleId);
}

export async function renameMothershipScheduleColumn(projectId: string, scheduleId: string, columnId: string, label: string): Promise<void> {
  await fetchForm(`/projects/${encodeURIComponent(projectId)}/schedules/${encodeURIComponent(scheduleId)}/columns/${encodeURIComponent(columnId)}/rename`, new URLSearchParams({ label }));
}

export async function deleteMothershipScheduleColumn(projectId: string, scheduleId: string, columnId: string): Promise<void> {
  await fetchForm(`/projects/${encodeURIComponent(projectId)}/schedules/${encodeURIComponent(scheduleId)}/columns/${encodeURIComponent(columnId)}/delete`, new URLSearchParams());
}

export async function reorderMothershipScheduleColumns(projectId: string, scheduleId: string, ids: string[]): Promise<void> {
  const body = new URLSearchParams();
  ids.forEach(id => body.append("ids", id));
  await fetchForm(`/projects/${encodeURIComponent(projectId)}/schedules/${encodeURIComponent(scheduleId)}/columns/reorder`, body);
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
      sourceImageUrls: item.sourceImageUrls ?? [],
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

function extractErrorMessage(text: string): string {
  try {
    const parsed = JSON.parse(text);
    if (typeof parsed.error === "string") return parsed.error;
  } catch { /* use raw text */ }
  return text;
}

async function fetchForm(path: string, body: URLSearchParams): Promise<void> {
  const url = `${getBackendBaseUrl()}${path}`;
  const sessionToken = await getSessionToken();
  const init: RequestInit = {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      ...(sessionToken ? { "X-Session-Token": sessionToken } : {}),
    },
    body: body.toString(),
  };

  if (typeof chrome !== "undefined" && chrome.runtime && chrome.runtime.sendMessage) {
    const response = await new Promise<any>((resolve, reject) => {
      chrome.runtime.sendMessage(
        { type: "PROXY_FETCH", url, init },
        (result) => {
          if (chrome.runtime.lastError) return reject(new Error(chrome.runtime.lastError.message));
          if (result.error) return reject(new Error(result.error));
          resolve(result);
        }
      );
    });
    if (!response.ok) {
      const text = response.text?.trim() || "";
      throw new Error(text || "Request failed.");
    }
    return;
  }

  const response = await fetch(url, init);
  if (!response.ok) {
    const text = (await response.text()).trim();
    throw new Error(text || "Request failed.");
  }
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
      const text = response.text?.trim() || "";
      throw new Error(extractErrorMessage(text) || "Mother Ship request failed.");
    }
    return JSON.parse(response.text) as T;
  }

  // Fallback for non-extension environment
  const response = await fetch(url, init);
  if (!response.ok) {
    const text = (await response.text()).trim();
    throw new Error(extractErrorMessage(text) || "Mother Ship request failed.");
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

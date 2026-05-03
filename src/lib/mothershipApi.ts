import type { MothershipProject, MothershipSchedule, ScheduleItem } from "./types";

const DEFAULT_BACKEND_BASE_URL = "http://10.0.0.104:8080";

type SallyRuntimeConfig = {
  backendBaseUrl?: string;
};

export async function listMothershipProjects(): Promise<MothershipProject[]> {
  return fetchJSON<MothershipProject[]>("/api/v1/projects");
}

export async function listMothershipSchedules(projectId: string): Promise<MothershipSchedule[]> {
  return fetchJSON<MothershipSchedule[]>(`/api/v1/projects/${encodeURIComponent(projectId)}/schedules`);
}

export async function saveMothershipScheduleItem(
  scheduleId: string,
  item: ScheduleItem
): Promise<void> {
  await fetchJSON(`/api/v1/schedules/${encodeURIComponent(scheduleId)}/items`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      code: "",
      title: item.title,
      description: item.description,
      manufacturer: item.manufacturer,
      modelNumber: item.modelNumber,
      finish: item.finish,
      finishModelNumber: item.finishModelNumber ?? "",
      notes: [...item.requiredAddOns, ...item.optionalCompanions].join("; "),
      sourceUrl: item.sourceUrl,
      sourceTitle: item.sourceTitle,
      sourceImageUrl: item.sourceImageUrl ?? "",
      sourcePdfLinks: item.sourcePdfLinks
    })
  });
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${getBackendBaseUrl()}${path}`, init);
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

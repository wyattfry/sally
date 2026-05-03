import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getMothershipScheduleUrl,
  listMothershipProjects,
  listMothershipSchedules,
  saveMothershipScheduleItem
} from "./mothershipApi";
import type { ScheduleItem } from "./types";

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

function scheduleItem(overrides: Partial<ScheduleItem> = {}): ScheduleItem {
  return {
    id: "draft-request-123",
    capturedAt: "2026-04-24T18:30:00.000Z",
    title: "Wall Faucet",
    manufacturer: "Example Co.",
    modelNumber: "WF-200",
    category: "Plumbing Fixture",
    description: "Wall-mounted faucet.",
    finish: "Polished Chrome",
    finishModelNumber: "WF-200-PC",
    requiredAddOns: ["Rough valve body"],
    optionalCompanions: [],
    sourceUrl: "https://example.com/products/wf-200",
    sourceTitle: "Example Co. WF-200 Wall Faucet",
    sourceImageUrl: "https://example.com/faucet.jpg",
    sourcePdfLinks: ["https://example.com/spec-sheet.pdf"],
    ...overrides
  };
}

describe("mothershipApi", () => {
  it("lists projects from the configured backend", async () => {
    vi.stubGlobal("__SALLY_CONFIG__", { backendBaseUrl: "http://localhost:8080" });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: "project-1", name: "House", address: "24 School St." }]), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(listMothershipProjects()).resolves.toEqual([
      { id: "project-1", name: "House", address: "24 School St." }
    ]);
    expect(fetchMock.mock.calls[0][0]).toBe("http://localhost:8080/api/v1/projects");
  });

  it("lists schedules for a project", async () => {
    vi.stubGlobal("__SALLY_CONFIG__", { backendBaseUrl: "http://localhost:8080" });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: "schedule-1", projectId: "project-1", name: "Bath", notes: "", position: 1 }]), {
        status: 200,
        headers: { "Content-Type": "application/json" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(listMothershipSchedules("project-1")).resolves.toEqual([
      { id: "schedule-1", projectId: "project-1", name: "Bath", notes: "", position: 1 }
    ]);
    expect(fetchMock.mock.calls[0][0]).toBe("http://localhost:8080/api/v1/projects/project-1/schedules");
  });

  it("saves a schedule item to the selected schedule", async () => {
    vi.stubGlobal("__SALLY_CONFIG__", { backendBaseUrl: "http://localhost:8080" });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "item-1", title: "Wall Faucet" }), {
        status: 201,
        headers: { "Content-Type": "application/json" }
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await saveMothershipScheduleItem("schedule-1", scheduleItem());

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://localhost:8080/api/v1/schedules/schedule-1/items");
    expect(init.method).toBe("POST");
    expect(JSON.parse(String(init.body))).toMatchObject({
      title: "Wall Faucet",
      manufacturer: "Example Co.",
      modelNumber: "WF-200",
      finish: "Polished Chrome",
      sourcePdfLinks: ["https://example.com/spec-sheet.pdf"]
    });
  });

  it("builds Mother Ship schedule URLs", () => {
    vi.stubGlobal("__SALLY_CONFIG__", { backendBaseUrl: "https://dev.spexxtool.com" });

    expect(getMothershipScheduleUrl("project-1", "schedule-1")).toBe(
      "https://dev.spexxtool.com/projects/project-1/schedules/schedule-1"
    );
  });
});

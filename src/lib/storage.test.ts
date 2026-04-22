import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearScheduleItems,
  listScheduleItems,
  listZones,
  saveScheduleItem,
  saveZone
} from "./storage";
import type { ScheduleItem } from "./types";

const storageState: Record<string, unknown> = {};

function installChromeStorageMock() {
  vi.stubGlobal("chrome", {
    storage: {
      local: {
        get: vi.fn(async (keys: string | string[]) => {
          const result: Record<string, unknown> = {};
          for (const key of Array.isArray(keys) ? keys : [keys]) {
            result[key] = storageState[key];
          }
          return result;
        }),
        set: vi.fn(async (values: Record<string, unknown>) => {
          Object.assign(storageState, values);
        }),
        remove: vi.fn(async (key: string) => {
          delete storageState[key];
        })
      }
    }
  });
}

function scheduleItem(overrides: Partial<ScheduleItem> = {}): ScheduleItem {
  return {
    id: "item-1",
    capturedAt: "2026-04-21T12:00:00.000Z",
    zone: "Primary Bath",
    title: "Wall-mounted faucet",
    manufacturer: "Example Co.",
    modelNumber: "EX-100",
    category: "Faucet",
    description: "Wall-mounted faucet with rough-in constraints.",
    finish: "Polished Chrome",
    finishModelNumber: "EX-100-PC",
    requiredAddOns: ["Rough valve body"],
    optionalCompanions: ["Drain assembly"],
    sourceUrl: "https://example.com/product",
    sourceTitle: "Example Product",
    sourceImageUrl: "https://example.com/product.jpg",
    sourcePdfLinks: ["https://example.com/spec.pdf"],
    ...overrides
  };
}

describe("schedule item storage", () => {
  beforeEach(() => {
    for (const key of Object.keys(storageState)) {
      delete storageState[key];
    }
    installChromeStorageMock();
  });

  it("lists an empty array when no schedule items are stored", async () => {
    await expect(listScheduleItems()).resolves.toEqual([]);
  });

  it("saves schedule items in insertion order", async () => {
    const first = scheduleItem({ id: "item-1" });
    const second = scheduleItem({ id: "item-2", title: "Toilet" });

    await saveScheduleItem(first);
    await saveScheduleItem(second);

    await expect(listScheduleItems()).resolves.toEqual([first, second]);
  });

  it("clears stored schedule items", async () => {
    await saveScheduleItem(scheduleItem());

    await clearScheduleItems();

    await expect(listScheduleItems()).resolves.toEqual([]);
  });

  it("lists default zones when no custom zones are stored", async () => {
    await expect(listZones()).resolves.toEqual([
      "Entry",
      "Kitchen",
      "Powder Room",
      "Primary Bath",
      "Bath 2",
      "Laundry",
      "Exterior"
    ]);
  });

  it("saves new zones without duplicating existing values", async () => {
    await saveZone("Guest Bath");
    await saveZone("Guest Bath");

    await expect(listZones()).resolves.toContain("Guest Bath");
    expect((await listZones()).filter((zone) => zone === "Guest Bath")).toHaveLength(1);
  });
});

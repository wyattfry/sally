import "@testing-library/jest-dom/vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ScheduleItem } from "./lib/types";
import { extractScheduleItem, shouldAllowMockFallback, shouldFallbackToMock } from "./lib/extractApi";
import {
  checkAuth,
  getMothershipScheduleUrl,
  listMothershipProjects,
  createMothershipProject,
  listMothershipSchedules,
  createMothershipSchedule,
  listMothershipScheduleColumns,
  getMothershipScheduleNextCode,
  saveMothershipScheduleItem,
} from "./lib/mothershipApi";
import { mockExtractScheduleItem } from "./lib/mockExtraction";

vi.mock("./lib/extractApi", () => ({
  EXTRACT_TIMEOUT_MS: 180_000,
  extractScheduleItem: vi.fn(),
  shouldAllowMockFallback: vi.fn(),
  shouldFallbackToMock: vi.fn()
}));

vi.mock("./lib/mockExtraction", () => ({
  mockExtractScheduleItem: vi.fn()
}));

vi.mock("./lib/mothershipApi", () => ({
  checkAuth: vi.fn(),
  getSignInUrl: vi.fn(),
  getMothershipScheduleUrl: vi.fn(),
  listMothershipProjects: vi.fn(),
  createMothershipProject: vi.fn(),
  listMothershipSchedules: vi.fn(),
  createMothershipSchedule: vi.fn(),
  listMothershipScheduleColumns: vi.fn(),
  getMothershipScheduleNextCode: vi.fn(),
  saveMothershipScheduleItem: vi.fn(),
}));

import App from "./App";

const storageState: Record<string, unknown> = {};

function extractedResult(itemOverrides: Partial<ScheduleItem> = {}, options: { suggestedScheduleName?: string; knownRooms?: string[] } = {}) {
  return { item: extractedItem(itemOverrides), knownRooms: options.knownRooms ?? [], suggestedScheduleName: options.suggestedScheduleName };
}

function extractedItem(overrides: Partial<ScheduleItem> = {}): ScheduleItem {
  return {
    id: "draft-request-123",
    capturedAt: "2026-04-24T18:30:00.000Z",
    room: "Primary Bath",
    data: {
      title: "Wall Faucet",
      manufacturer: "Example Co.",
      model_number: "WF-200",
      description: "Wall-mounted faucet.",
      finish: "Polished Chrome",
      notes: "Rough valve body"
    },
    sourceUrl: "https://example.com/products/wf-200",
    sourceTitle: "Example Co. WF-200 Wall Faucet",
    sourceImageUrl: "https://example.com/faucet.jpg",
    sourcePdfLinks: ["https://example.com/spec-sheet.pdf"],
    ...overrides
  };
}

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

const TEST_COLUMNS = [
  { id: "col-0", scheduleId: "schedule-1", key: "code", label: "Code", kind: "text", position: 0 },
  { id: "col-1", scheduleId: "schedule-1", key: "title", label: "Title", kind: "text", position: 1 },
  { id: "col-2", scheduleId: "schedule-1", key: "manufacturer", label: "Manufacturer", kind: "text", position: 2 },
  { id: "col-3", scheduleId: "schedule-1", key: "model_number", label: "Model Number", kind: "text", position: 3 },
  { id: "col-4", scheduleId: "schedule-1", key: "finish", label: "Finish", kind: "text", position: 4 },
  { id: "col-5", scheduleId: "schedule-1", key: "notes", label: "Notes", kind: "text", position: 5 }
];

describe("App", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    document.title = "Example Co. WF-200 Wall Faucet";
    document.head.innerHTML = `
      <meta property="og:image" content="https://example.com/faucet.jpg" />
      <script type="application/ld+json">
        {"@type":"Product","name":"Wall Faucet","brand":{"name":"Example Co."},"sku":"WF-200"}
      </script>
    `;
    document.body.innerHTML = `
      <p>Polished chrome wall-mounted faucet. Requires rough valve body.</p>
    `;
    for (const key of Object.keys(storageState)) {
      delete storageState[key];
    }
    installChromeStorageMock();
    vi.mocked(checkAuth).mockResolvedValue(true);
    vi.mocked(extractScheduleItem).mockResolvedValue(extractedResult());
    vi.mocked(mockExtractScheduleItem).mockReturnValue(extractedItem({ id: "mock-draft-123" }));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(false);
    vi.mocked(shouldFallbackToMock).mockReturnValue(false);
    vi.mocked(listMothershipProjects).mockResolvedValue([
      { id: "project-1", name: "Lake House", address: "24 School St.", description: "", updatedAt: "2026-01-01T00:00:00Z" }
    ]);
    vi.mocked(listMothershipSchedules).mockResolvedValue([
      { id: "schedule-1", projectId: "project-1", name: "Bath", kind: "items", notes: "", position: 1 }
    ]);
    vi.mocked(listMothershipScheduleColumns).mockResolvedValue(TEST_COLUMNS);
    vi.mocked(saveMothershipScheduleItem).mockResolvedValue(undefined);
    vi.mocked(createMothershipSchedule).mockResolvedValue(
      { id: "schedule-new", projectId: "project-1", name: "Paint Schedule", kind: "items", notes: "", position: 2 }
    );
    vi.mocked(getMothershipScheduleNextCode).mockResolvedValue("A-4");
    vi.mocked(getMothershipScheduleUrl).mockReturnValue(
      "http://localhost:8080/projects/project-1/schedules/schedule-1"
    );
  });

  it("opens Sally, edits a proposal, saves it, and shows an accepted-item toast", async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(await screen.findByRole("button", { name: "SPEC" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "SPEC" }));

    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();
    expect(screen.getByLabelText("Project")).toHaveValue("project-1");
    expect(screen.getByLabelText("Schedule")).toHaveValue("schedule-1");

    await user.selectOptions(screen.getByLabelText("Room"), "Primary Bath");
    await user.clear(screen.getByLabelText("Title"));
    await user.type(screen.getByLabelText("Title"), "Wall faucet revised");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => expect(screen.queryByLabelText("Sally capture panel")).not.toBeInTheDocument());
    expect(saveMothershipScheduleItem).toHaveBeenCalledWith(
      "schedule-1",
      expect.objectContaining({ data: expect.objectContaining({ title: "Wall faucet revised" }) }),
      expect.anything()
    );
    expect(screen.getByText(/Added to/)).toBeInTheDocument();
  });

  it("lets the user choose a Mother Ship schedule before saving", async () => {
    const user = userEvent.setup();
    vi.mocked(listMothershipProjects).mockResolvedValue([
      { id: "project-1", name: "Lake House", address: "24 School St.", description: "", updatedAt: "2026-01-01T00:00:00Z" },
      { id: "project-2", name: "Townhouse", address: "307 W38th St.", description: "", updatedAt: "2026-01-01T00:00:00Z" }
    ]);
    vi.mocked(listMothershipSchedules).mockImplementation(async (projectId: string) =>
      projectId === "project-2"
        ? [{ id: "schedule-2", projectId: "project-2", name: "Kitchen", kind: "items", notes: "", position: 1 }]
        : [{ id: "schedule-1", projectId: "project-1", name: "Bath", kind: "items", notes: "", position: 1 }]
    );

    render(<App />);
    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await screen.findByDisplayValue("Wall Faucet");

    await user.selectOptions(screen.getByLabelText("Project"), "project-2");

    await waitFor(() => expect(screen.getByLabelText("Schedule")).toHaveValue("schedule-2"));
    await user.click(screen.getByRole("button", { name: "OK" }));

    expect(saveMothershipScheduleItem).toHaveBeenCalledWith(
      "schedule-2",
      expect.objectContaining({ data: expect.objectContaining({ title: "Wall Faucet" }) }),
      expect.anything()
    );
  });

  it("defaults to first (most-recently-updated) project even when stored context points to another", async () => {
    const user = userEvent.setup();
    storageState["sally.activeContext"] = { projectId: "project-2", scheduleId: "schedule-2" };
    vi.mocked(listMothershipProjects).mockResolvedValue([
      { id: "project-1", name: "Newest Project", address: "", description: "", updatedAt: "2026-04-01T00:00:00Z" },
      { id: "project-2", name: "Older Project", address: "", description: "", updatedAt: "2026-01-01T00:00:00Z" }
    ]);
    vi.mocked(listMothershipSchedules).mockImplementation(async (projectId: string) =>
      projectId === "project-1"
        ? [{ id: "schedule-1", projectId: "project-1", name: "Bath", kind: "items", notes: "", position: 1 }]
        : [{ id: "schedule-2", projectId: "project-2", name: "Kitchen", kind: "items", notes: "", position: 1 }]
    );

    render(<App />);
    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await screen.findByDisplayValue("Wall Faucet");

    expect(screen.getByLabelText("Project")).toHaveValue("project-1");
    expect(screen.getByLabelText("Schedule")).toHaveValue("schedule-1");
  });

  it("does not show undo while creating a new proposal", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Undo" })).not.toBeInTheDocument();
  });

  it("supports selecting an existing room and adding a new room", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await screen.findByDisplayValue("Wall Faucet");

    await user.selectOptions(screen.getByLabelText("Room"), "Primary Bath");
    expect(screen.getByLabelText("Room")).toHaveValue("Primary Bath");

    await user.selectOptions(screen.getByLabelText("Room"), "__add_new__");
    await user.type(screen.getByLabelText("Name"), "Guest Bath");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(screen.getByLabelText("Room")).toHaveValue("Guest Bath");
  });

  it("minimizes on Escape and restores the draft without discarding edits", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    const title = await screen.findByLabelText("Title");
    await user.clear(title);
    await user.type(title, "Draft title to keep");

    await user.keyboard("{Escape}");

    expect(screen.queryByLabelText("Sally capture panel")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Restore Sally draft" }));

    expect(await screen.findByDisplayValue("Draft title to keep")).toBeInTheDocument();
  });

  it("does not visually change SPEC when the current page has already been spec'd", async () => {
    render(<App />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "SPEC" })).not.toHaveClass("spec-button--specd")
    );
  });

  it("uses mock extraction in dev when fallback is explicitly enabled", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Extraction backend is unreachable."));
    vi.mocked(mockExtractScheduleItem).mockReturnValue(
      extractedItem({ id: "mock-draft-123", data: { title: "Mock fallback faucet" } })
    );
    vi.mocked(shouldAllowMockFallback).mockReturnValue(true);
    vi.mocked(shouldFallbackToMock).mockReturnValue(true);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByDisplayValue("Mock fallback faucet")).toBeInTheDocument();
  });

  it("does not silently fall back in production-facing mode", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Backend unavailable"));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(false);
    vi.mocked(shouldFallbackToMock).mockReturnValue(true);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByText("Backend unavailable")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("Wall Faucet")).not.toBeInTheDocument();
    expect(mockExtractScheduleItem).not.toHaveBeenCalled();
  });

  it("does not use mock fallback for backend extraction errors even when fallback is enabled", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Model rejected the page."));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(true);
    vi.mocked(shouldFallbackToMock).mockReturnValue(false);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByText("Model rejected the page.")).toBeInTheDocument();
    expect(mockExtractScheduleItem).not.toHaveBeenCalled();
  });

  it("shows extraction error in the panel", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Backend unavailable"));
    vi.mocked(shouldFallbackToMock).mockReturnValue(false);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByText("Backend unavailable")).toBeInTheDocument();
    expect(screen.getByLabelText("Sally capture panel")).toBeInTheDocument();
  });

  it("canceling auto-triggered new-schedule modal restores the correct code for the original schedule", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockResolvedValue(
      extractedResult({}, { suggestedScheduleName: "Paint Schedule" })
    );

    render(<App />);
    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    // Auto-triggered modal appears with suggested name and hint
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Paint Schedule");
    expect(screen.getByText(/doesn't seem to belong/)).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    // Cancel restores the original schedule and re-fetches its next code
    expect(screen.getByLabelText("Schedule")).toHaveValue("schedule-1");
    await waitFor(() => expect(screen.getByLabelText("Code")).toHaveValue("A-4"));
    expect(getMothershipScheduleNextCode).toHaveBeenCalledWith("schedule-1");
  });

  it("uses the matched schedule's next code even when the extraction response already contained a code for a different schedule", async () => {
    const user = userEvent.setup();
    vi.mocked(listMothershipSchedules).mockResolvedValue([
      { id: "schedule-1", projectId: "project-1", name: "Appliance Schedule", kind: "items", notes: "", position: 1 },
      { id: "schedule-paint", projectId: "project-1", name: "Paint Schedule", kind: "items", notes: "", position: 2 },
    ]);
    // Extraction server returns nextCode "A-6" (for Appliance Schedule) but suggests Paint Schedule
    vi.mocked(extractScheduleItem).mockResolvedValue({
      item: extractedItem({ data: { code: "A-6", title: "Wall Paint" } }),
      knownRooms: [],
      suggestedScheduleName: "Paint Schedule",
    });
    vi.mocked(getMothershipScheduleNextCode).mockImplementation(async (scheduleId) =>
      scheduleId === "schedule-paint" ? "P-4" : "A-6"
    );

    render(<App />);
    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    // Panel switches to Paint Schedule
    await waitFor(() => expect(screen.getByLabelText("Schedule")).toHaveValue("schedule-paint"));
    // Code is for Paint Schedule ("P-4"), not the stale "A-6" from the extraction response
    expect(screen.getByLabelText("Code")).toHaveValue("P-4");
    expect(getMothershipScheduleNextCode).toHaveBeenCalledWith("schedule-paint");
  });

  it("creating a new schedule from the auto-triggered modal keeps the new schedule selected", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockResolvedValue(
      extractedResult({}, { suggestedScheduleName: "Paint Schedule" })
    );
    vi.mocked(getMothershipScheduleNextCode).mockImplementation(async (scheduleId) =>
      scheduleId === "schedule-new" ? "P-1" : "A-4"
    );

    render(<App />);
    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    await screen.findByRole("dialog");
    expect(screen.getByLabelText("Name")).toHaveValue("Paint Schedule");

    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    // New schedule stays selected, not reverted to original
    await waitFor(() => expect(screen.getByLabelText("Schedule")).toHaveValue("schedule-new"));
    expect(getMothershipScheduleNextCode).toHaveBeenCalledWith("schedule-new");
  });

  it("prompts to create a project before extracting when the user has no projects", async () => {
    const user = userEvent.setup();
    vi.mocked(listMothershipProjects).mockResolvedValue([]);
    vi.mocked(listMothershipSchedules).mockResolvedValue([]);
    vi.mocked(createMothershipProject).mockImplementation(async (name: string) => ({
      id: "project-new",
      name,
      address: "",
      description: "",
      updatedAt: "2026-01-01T00:00:00Z",
    }));

    render(<App />);
    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    // Should prompt for a project, not extract.
    expect(await screen.findByLabelText("Project name")).toBeInTheDocument();
    expect(extractScheduleItem).not.toHaveBeenCalled();

    // Once a project exists, extraction should proceed.
    vi.mocked(listMothershipProjects).mockResolvedValue([
      { id: "project-new", name: "Main St.", address: "", description: "", updatedAt: "2026-01-01T00:00:00Z" }
    ]);
    vi.mocked(listMothershipSchedules).mockResolvedValue([
      { id: "schedule-1", projectId: "project-new", name: "Bath", kind: "items", notes: "", position: 1 }
    ]);

    await user.type(screen.getByLabelText("Project name"), "Main St.");
    await user.click(screen.getByRole("button", { name: "Create and continue" }));

    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();
    expect(createMothershipProject).toHaveBeenCalledWith("Main St.");
    expect(extractScheduleItem).toHaveBeenCalled();
  });
});

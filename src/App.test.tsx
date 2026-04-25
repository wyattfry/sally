import "@testing-library/jest-dom/vitest";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ScheduleItem } from "./lib/types";
import { extractScheduleItem, shouldAllowMockFallback, shouldFallbackToMock } from "./lib/extractApi";
import { mockExtractScheduleItem } from "./lib/mockExtraction";

vi.mock("./lib/extractApi", () => ({
  extractScheduleItem: vi.fn(),
  shouldAllowMockFallback: vi.fn(),
  shouldFallbackToMock: vi.fn()
}));

vi.mock("./lib/mockExtraction", () => ({
  mockExtractScheduleItem: vi.fn()
}));

import App from "./App";

const storageState: Record<string, unknown> = {};

function extractedItem(overrides: Partial<ScheduleItem> = {}): ScheduleItem {
  return {
    id: "draft-request-123",
    capturedAt: "2026-04-24T18:30:00.000Z",
    zone: "",
    title: "Wall Faucet",
    manufacturer: "Example Co.",
    modelNumber: "WF-200",
    category: "Faucet",
    description: "Wall-mounted faucet.",
    finish: "Polished Chrome",
    requiredAddOns: ["Rough valve body"],
    optionalCompanions: [],
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
    vi.mocked(extractScheduleItem).mockResolvedValue(extractedItem());
    vi.mocked(mockExtractScheduleItem).mockReturnValue(extractedItem({ id: "mock-draft-123" }));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(false);
    vi.mocked(shouldFallbackToMock).mockReturnValue(false);
  });

  it("opens Sally, edits a proposal, saves it, and shows an accepted-item toast", async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(await screen.findByRole("button", { name: "SPEC" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /My New Project.*0 items/i })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "SPEC" }));

    expect(screen.getByText("Reading page")).toBeInTheDocument();
    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();
    expect(screen.getByText("My New Project")).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Zone"), "Primary Bath");
    await user.selectOptions(screen.getByLabelText("Category"), "Plumbing Fixture");
    await user.clear(screen.getByLabelText("Title"));
    await user.type(screen.getByLabelText("Title"), "Wall faucet revised");
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => expect(screen.queryByLabelText("Sally capture panel")).not.toBeInTheDocument());
    expect(screen.getByText("Item added")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "SPEC" })).not.toHaveClass("spec-button--specd");
  });

  it("does not show undo while creating a new proposal", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByDisplayValue("Wall Faucet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Undo" })).not.toBeInTheDocument();
  });

  it("supports selecting an existing zone and adding a new zone", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await screen.findByDisplayValue("Wall Faucet");

    await user.selectOptions(screen.getByLabelText("Zone"), "Primary Bath");
    expect(screen.getByLabelText("Zone")).toHaveValue("Primary Bath");

    await user.selectOptions(screen.getByLabelText("Zone"), "__add_new__");
    await user.type(screen.getByLabelText("New zone"), "Guest Bath");
    await user.click(screen.getByRole("button", { name: "Add zone" }));

    expect(screen.getByLabelText("Zone")).toHaveValue("Guest Bath");
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
    storageState["sally.scheduleItems"] = [
      {
        id: "item-1",
        capturedAt: "2026-04-21T12:00:00.000Z",
        zone: "Primary Bath",
        title: "Wall Faucet",
        manufacturer: "Example Co.",
        modelNumber: "WF-200",
        category: "Faucet",
        description: "Wall-mounted faucet.",
        finish: "Polished Chrome",
        requiredAddOns: [],
        optionalCompanions: [],
        sourceUrl: window.location.href,
        sourceTitle: document.title,
        sourcePdfLinks: []
      }
    ];

    render(<App />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "SPEC" })).not.toHaveClass("spec-button--specd")
    );
  });

  it("opens the schedule viewer from the Sally panel", async () => {
    const user = userEvent.setup();
    const printSpy = vi.spyOn(window, "print").mockImplementation(() => undefined);
    const printDocument = {
      close: vi.fn(),
      write: vi.fn()
    };
    const printWindow = {
      document: printDocument,
      focus: vi.fn(),
      print: vi.fn()
    };
    const openSpy = vi.spyOn(window, "open").mockReturnValue(printWindow as unknown as Window);
    storageState["sally.scheduleItems"] = [
      {
        id: "item-1",
        capturedAt: "2026-04-21T12:00:00.000Z",
        zone: "Primary Bath",
        title: "Wall Faucet",
        manufacturer: "Example Co.",
        modelNumber: "WF-200",
        category: "Faucet",
        description: "Wall-mounted faucet.",
        finish: "Polished Chrome",
        requiredAddOns: ["Rough valve body"],
        optionalCompanions: [],
        sourceUrl: "https://example.com/products/wf-200",
        sourceTitle: "Example Co. WF-200 Wall Faucet",
        sourceImageUrl: "https://example.com/faucet.jpg",
        sourcePdfLinks: []
      }
    ];

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await user.click(await screen.findByRole("button", { name: "View Items" }));

    expect(screen.getByLabelText("Captured schedule")).toBeInTheDocument();
    expect(screen.queryByLabelText("Sally capture panel")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Rename My New Project" })).toBeInTheDocument();
    expect(screen.getByText("Wall Faucet")).toBeInTheDocument();
    expect(screen.getByText("Primary Bath")).toBeInTheDocument();
    expect(screen.getByText("Example Co.")).toBeInTheDocument();
    expect(screen.getByText("WF-200")).toBeInTheDocument();
    const thumbnailLink = screen.getByRole("link", { name: "Wall Faucet thumbnail" });
    expect(thumbnailLink).toHaveAttribute("href", "https://example.com/products/wf-200");
    expect(thumbnailLink.querySelector("img")).toHaveAttribute(
      "src",
      "https://example.com/faucet.jpg"
    );

    await user.click(screen.getByRole("button", { name: "Rename My New Project" }));
    await user.clear(screen.getByLabelText("Project name"));
    await user.type(screen.getByLabelText("Project name"), "Lake House");
    await user.keyboard("{Enter}");

    expect(screen.getByRole("button", { name: "Rename Lake House" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Print" }));
    expect(openSpy).toHaveBeenCalledWith("", "_blank", "width=1100,height=800");
    expect(printDocument.write).toHaveBeenCalledWith(expect.stringContaining("Lake House"));
    expect(printDocument.write).toHaveBeenCalledWith(expect.stringContaining("Wall Faucet"));
    expect(printDocument.write).toHaveBeenCalledWith(expect.stringContaining("Primary Bath"));
    expect(printWindow.print).toHaveBeenCalledOnce();
    expect(printSpy).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Remove Wall Faucet" }));
    await waitFor(() => expect(screen.queryByText("Wall Faucet")).not.toBeInTheDocument());
    expect(screen.getByText("No accepted items yet.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close schedule" }));
    expect(screen.queryByLabelText("Captured schedule")).not.toBeInTheDocument();
  });

  it("removes only one duplicate item from the viewer", async () => {
    const user = userEvent.setup();
    storageState["sally.scheduleItems"] = [
      {
        id: "item-1",
        capturedAt: "2026-04-21T12:00:00.000Z",
        zone: "Primary Bath",
        title: "Wall Faucet",
        manufacturer: "Example Co.",
        modelNumber: "WF-200",
        category: "Faucet",
        description: "Wall-mounted faucet.",
        finish: "Polished Chrome",
        requiredAddOns: [],
        optionalCompanions: [],
        sourceUrl: "https://example.com/products/wf-200",
        sourceTitle: "Example Product",
        sourcePdfLinks: []
      },
      {
        id: "item-1-2",
        capturedAt: "2026-04-21T12:01:00.000Z",
        zone: "Powder Room",
        title: "Wall Faucet",
        manufacturer: "Example Co.",
        modelNumber: "WF-200",
        category: "Faucet",
        description: "Wall-mounted faucet.",
        finish: "Polished Chrome",
        requiredAddOns: [],
        optionalCompanions: [],
        sourceUrl: "https://example.com/products/wf-200",
        sourceTitle: "Example Product",
        sourcePdfLinks: []
      }
    ];

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));
    await user.click(await screen.findByRole("button", { name: "View Items" }));
    expect(screen.getAllByText("Wall Faucet")).toHaveLength(2);

    await user.click(screen.getAllByRole("button", { name: "Remove Wall Faucet" })[0]);

    expect(screen.getAllByText("Wall Faucet")).toHaveLength(1);
    expect(screen.getByText("Powder Room")).toBeInTheDocument();
    expect(screen.queryByText("Primary Bath")).not.toBeInTheDocument();
  });

  it("uses mock extraction in dev when fallback is explicitly enabled", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Extraction backend is unreachable."));
    vi.mocked(mockExtractScheduleItem).mockReturnValue(
      extractedItem({ id: "mock-draft-123", title: "Mock fallback faucet" })
    );
    vi.mocked(shouldAllowMockFallback).mockReturnValue(true);
    vi.mocked(shouldFallbackToMock).mockReturnValue(true);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByDisplayValue("Mock fallback faucet")).toBeInTheDocument();
    expect(screen.getByText("Using local mock fallback.")).toBeInTheDocument();
  });

  it("does not silently fall back in production-facing mode", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Backend unavailable"));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(false);
    vi.mocked(shouldFallbackToMock).mockReturnValue(true);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    await waitFor(() => expect(screen.queryByLabelText("Sally capture panel")).not.toBeInTheDocument());
    expect(screen.queryByDisplayValue("Wall Faucet")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Extraction error")).toBeInTheDocument();
    expect(screen.getAllByText("Backend unavailable")).toHaveLength(2);
    expect(mockExtractScheduleItem).not.toHaveBeenCalled();
  });

  it("does not use mock fallback for backend extraction errors even when fallback is enabled", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Model rejected the page."));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(true);
    vi.mocked(shouldFallbackToMock).mockReturnValue(false);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByLabelText("Extraction error")).toBeInTheDocument();
    expect(screen.getAllByText("Model rejected the page.")).toHaveLength(2);
    expect(mockExtractScheduleItem).not.toHaveBeenCalled();
  });

  it("keeps extraction failures user-visible and recoverable", async () => {
    const user = userEvent.setup();
    vi.mocked(extractScheduleItem)
      .mockRejectedValueOnce(new Error("Backend unavailable"))
      .mockResolvedValueOnce(extractedItem({ title: "Recovered faucet" }));
    vi.mocked(shouldFallbackToMock).mockReturnValue(false);

    render(<App />);

    await user.click(await screen.findByRole("button", { name: "SPEC" }));

    expect(await screen.findByLabelText("Extraction error")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry extraction" }));

    expect(await screen.findByDisplayValue("Recovered faucet")).toBeInTheDocument();
  });

  it("does not clear a newer toast when an older timer expires", async () => {
    vi.useFakeTimers();
    vi.mocked(extractScheduleItem).mockRejectedValue(new Error("Extraction backend is unreachable."));
    vi.mocked(shouldAllowMockFallback).mockReturnValue(true);
    vi.mocked(shouldFallbackToMock).mockReturnValue(true);
    vi.mocked(mockExtractScheduleItem).mockReturnValueOnce(extractedItem({ title: "Fallback item" }));

    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "SPEC" }));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });
    expect(screen.getByText("Using local mock fallback.")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    fireEvent.click(screen.getByRole("button", { name: "OK" }));
    await act(async () => {});
    expect(screen.getByText("Item added")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2200);
    });
    expect(screen.getByText("Item added")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(screen.queryByText("Item added")).not.toBeInTheDocument();
    vi.useRealTimers();
  });
});
